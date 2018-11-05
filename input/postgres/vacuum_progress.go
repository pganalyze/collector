package postgres

import (
	"database/sql"
	"fmt"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

const vacuumProgressSQLpg95 string = `
SELECT (extract(epoch from a.query_start)::int::text || to_char(pid, 'FM000000'))::bigint AS vacuum_identity,
			 (extract(epoch from COALESCE(backend_start, pg_postmaster_start_time()))::int::text || to_char(pid, 'FM000000'))::bigint AS backend_identity,
			 a.datname,
			 (SELECT regexp_matches(query, 'autovacuum: VACUUM (ANALYZE )?([^\.]+).(.+)'))[2] AS nspname,
			 (SELECT regexp_matches(query, 'autovacuum: VACUUM (ANALYZE )?([^\.]+).(.+)'))[3] AS relname,
			 COALESCE(a.usename, '') AS usename,
			 a.query_start AS started_at,
			 a.query LIKE 'autovacuum: VACUUM%%' AS autovacuum,
			 '' AS phase,
			 0 AS heap_blks_total,
			 0 AS heap_blks_scanned,
			 0 AS heap_blks_vacuumed,
			 0 AS index_vacuum_count,
			 0 AS max_dead_tuples,
			 0 AS num_dead_tuples
	FROM %s a
 WHERE a.query LIKE 'autovacuum: VACUUM%%'
`

const vacuumProgressSQLDefault string = `
SELECT (extract(epoch from a.query_start)::int::text || to_char(pid, 'FM000000'))::bigint AS vacuum_identity,
			 (extract(epoch from COALESCE(backend_start, pg_postmaster_start_time()))::int::text || to_char(pid, 'FM000000'))::bigint AS backend_identity,
			 a.datname,
			 COALESCE(n.nspname, (SELECT regexp_matches(query, 'autovacuum: VACUUM (ANALYZE )?([^\.]+).(.+)'))[2]) AS nspname,
			 COALESCE(c.relname, (SELECT regexp_matches(query, 'autovacuum: VACUUM (ANALYZE )?([^\.]+).(.+)'))[3]) AS relname,
			 COALESCE(a.usename, '') AS usename,
			 a.query_start AS started_at,
			 a.query LIKE 'autovacuum: VACUUM%%' AS autovacuum,
			 COALESCE(v.phase, '') AS phase,
			 COALESCE(v.heap_blks_total, 0) AS heap_blks_total,
			 COALESCE(v.heap_blks_scanned, 0) AS heap_blks_scanned,
			 COALESCE(v.heap_blks_vacuumed, 0) AS heap_blks_vacuumed,
			 COALESCE(v.index_vacuum_count, 0) AS index_vacuum_count,
			 COALESCE(v.max_dead_tuples, 0) AS max_dead_tuples,
			 COALESCE(v.num_dead_tuples, 0) AS num_dead_tuples
	FROM %s v
			 JOIN %s a USING (pid)
			 LEFT JOIN pg_class c ON (c.oid = v.relid)
			 LEFT JOIN pg_namespace n ON (n.oid = c.relnamespace)
`

func GetVacuumProgress(logger *util.Logger, db *sql.DB, postgresVersion state.PostgresVersion) ([]state.PostgresVacuumProgress, error) {
	var activitySourceTable string
	var sql string

	if statsHelperExists(db, "get_stat_activity") {
		activitySourceTable = "pganalyze.get_stat_activity()"
	} else {
		activitySourceTable = "pg_stat_activity"
	}

	if postgresVersion.Numeric < state.PostgresVersion96 {
		sql = fmt.Sprintf(vacuumProgressSQLpg95, activitySourceTable)
	} else {
		var vacuumSourceTable string
		if statsHelperExists(db, "get_stat_progress_vacuum") {
			vacuumSourceTable = "pganalyze.get_stat_progress_vacuum()"
		} else {
			vacuumSourceTable = "pg_stat_progress_vacuum"
		}
		sql = fmt.Sprintf(vacuumProgressSQLDefault, vacuumSourceTable, activitySourceTable)
	}

	stmt, err := db.Prepare(QueryMarkerSQL + sql)
	if err != nil {
		return nil, err
	}

	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		return nil, err
	}

	var vacuums []state.PostgresVacuumProgress

	for rows.Next() {
		var row state.PostgresVacuumProgress

		err := rows.Scan(&row.VacuumIdentity, &row.BackendIdentity, &row.DatabaseName,
			&row.SchemaName, &row.RelationName, &row.RoleName, &row.StartedAt, &row.Autovacuum,
			&row.Phase, &row.HeapBlksTotal, &row.HeapBlksScanned, &row.HeapBlksVacuumed,
			&row.IndexVacuumCount, &row.MaxDeadTuples, &row.NumDeadTuples)
		if err != nil {
			return nil, err
		}

		vacuums = append(vacuums, row)
	}

	rows.Close()

	for idx, row := range vacuums {
		if row.SchemaName == "pg_toast" {
			schemaName, relationName, err := resolveToastTable(db, row.RelationName)
			if err != nil && schemaName != "" && relationName != "" {
				row.SchemaName = schemaName
				row.RelationName = relationName
				row.Toast = true
			}
			vacuums[idx] = row
		}
	}

	return vacuums, nil
}

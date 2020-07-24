package postgres

import (
	"database/sql"
	"fmt"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

const vacuumProgressSQLpg95 string = `
SELECT (EXTRACT(epoch FROM a.query_start)::int::text || pg_catalog.to_char(pid, 'FM0000000'))::bigint AS vacuum_identity,
			 (EXTRACT(epoch FROM COALESCE(backend_start, pg_catalog.pg_postmaster_start_time()))::int::text || pg_catalog.to_char(pid, 'FM0000000'))::bigint AS backend_identity,
			 a.datname,
			 (SELECT pg_catalog.regexp_matches(query, 'autovacuum: VACUUM (ANALYZE )?([^\.]+).([^\(]+)( \(to prevent wraparound\))?'))[2] AS nspname,
			 CASE
			   WHEN ($1 = '' OR
				   (COALESCE(n.nspname,
					  (SELECT pg_catalog.regexp_matches(query, 'autovacuum: VACUUM (ANALYZE )?([^\.]+).([^\(]+)( \(to prevent wraparound\))?'))[2])
					  || '.' ||
					  COALESCE(c.relname,
					  (SELECT pg_catalog.regexp_matches(query, 'autovacuum: VACUUM (ANALYZE )?([^\.]+).([^\(]+)( \(to prevent wraparound\))?'))[3])) !~* $1)
			   THEN COALESCE(c.relname,
				   (SELECT pg_catalog.regexp_matches(query, 'autovacuum: VACUUM (ANALYZE )?([^\.]+).([^\(]+)( \(to prevent wraparound\))?'))[3])
			   ELSE
				   ''
		   END AS relname,
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
SELECT (EXTRACT(epoch FROM a.query_start)::int::text || pg_catalog.to_char(pid, 'FM0000000'))::bigint AS vacuum_identity,
			 (EXTRACT(epoch FROM COALESCE(backend_start, pg_catalog.pg_postmaster_start_time()))::int::text || pg_catalog.to_char(pid, 'FM0000000'))::bigint AS backend_identity,
			 a.datname,
			 COALESCE(n.nspname, (SELECT pg_catalog.regexp_matches(query, 'autovacuum: VACUUM (ANALYZE )?([^\.]+).([^\(]+)( \(to prevent wraparound\))?'))[2]) AS nspname,
			 CASE
				 WHEN ($1 = '' OR
					 (COALESCE(n.nspname,
						(SELECT pg_catalog.regexp_matches(query, 'autovacuum: VACUUM (ANALYZE )?([^\.]+).([^\(]+)( \(to prevent wraparound\))?'))[2])
						|| '.' ||
						COALESCE(c.relname,
						(SELECT pg_catalog.regexp_matches(query, 'autovacuum: VACUUM (ANALYZE )?([^\.]+).([^\(]+)( \(to prevent wraparound\))?'))[3])) !~* $1)
				 THEN COALESCE(c.relname,
				   (SELECT pg_catalog.regexp_matches(query, 'autovacuum: VACUUM (ANALYZE )?([^\.]+).([^\(]+)( \(to prevent wraparound\))?'))[3])
				 ELSE
				   ''
			 END AS relname,
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
			 LEFT JOIN pg_catalog.pg_class c ON (c.oid = v.relid)
			 LEFT JOIN pg_catalog.pg_namespace n ON (n.oid = c.relnamespace)
 WHERE v.relid IS NOT NULL OR a.query <> '<insufficient privilege>'
`

func GetVacuumProgress(logger *util.Logger, db *sql.DB, postgresVersion state.PostgresVersion, ignoreRegexp string) ([]state.PostgresVacuumProgress, error) {
	var activitySourceTable string
	var sql string

	if statsHelperExists(db, "get_stat_activity") {
		activitySourceTable = "pganalyze.get_stat_activity()"
	} else {
		activitySourceTable = "pg_catalog.pg_stat_activity"
	}

	if postgresVersion.Numeric < state.PostgresVersion96 {
		sql = fmt.Sprintf(vacuumProgressSQLpg95, activitySourceTable)
	} else {
		var vacuumSourceTable string
		if statsHelperExists(db, "get_stat_progress_vacuum") {
			vacuumSourceTable = "pganalyze.get_stat_progress_vacuum()"
		} else {
			vacuumSourceTable = "pg_catalog.pg_stat_progress_vacuum"
		}
		sql = fmt.Sprintf(vacuumProgressSQLDefault, vacuumSourceTable, activitySourceTable)
	}

	stmt, err := db.Prepare(QueryMarkerSQL + sql)
	if err != nil {
		return nil, err
	}

	defer stmt.Close()

	rows, err := stmt.Query(ignoreRegexp)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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

	for idx, row := range vacuums {
		if row.SchemaName == "pg_toast" {
			schemaName, relationName, err := resolveToastTable(db, row.RelationName)
			if err != nil {
				logger.PrintVerbose("Failed to resolve TOAST table \"%s\": %s", row.RelationName, err)
			} else if schemaName != "" && relationName != "" {
				row.SchemaName = schemaName
				row.RelationName = relationName
				row.Toast = true
				vacuums[idx] = row
			}
		}
	}

	return vacuums, nil
}

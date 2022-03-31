package postgres

import (
	"database/sql"
	"fmt"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

const vacuumProgressSQLpg95 string = `
WITH activity AS (
	SELECT pg_catalog.to_char(pid, 'FM0000000') AS padded_pid,
	       EXTRACT(epoch FROM a.query_start)::int::text AS query_start_epoch,
				 EXTRACT(epoch FROM COALESCE(backend_start, pg_catalog.pg_postmaster_start_time()))::int::text AS backend_start_epoch,
				 a.datname,
				 (SELECT pg_catalog.regexp_matches(query, 'autovacuum: VACUUM (ANALYZE )?([^\.]+).([^\(]+)( \(to prevent wraparound\))?'))[2] AS nspname,
				 (SELECT pg_catalog.regexp_matches(query, 'autovacuum: VACUUM (ANALYZE )?([^\.]+).([^\(]+)( \(to prevent wraparound\))?'))[3] AS relname,
				 COALESCE(a.usename, '') AS usename,
				 a.query_start,
				 a.query LIKE 'autovacuum: VACUUM%%' AS autovacuum
    FROM %s a
	 WHERE a.query LIKE 'autovacuum: VACUUM%%'
)
SELECT (a.query_start_epoch || a.padded_pid)::bigint AS vacuum_identity,
			 (a.backend_start_epoch || a.padded_pid)::bigint AS backend_identity,
			 a.datname,
			 a.nspname,
			 CASE
			   WHEN ($1 = '' OR (a.nspname || '.' || a.relname) !~* $1) THEN a.relname
			   ELSE ''
		   END AS relname,
       a.usename,
			 a.query_start AS started_at,
			 a.autovacuum,
			 '' AS phase,
			 0 AS heap_blks_total,
			 0 AS heap_blks_scanned,
			 0 AS heap_blks_vacuumed,
			 0 AS index_vacuum_count,
			 0 AS max_dead_tuples,
			 0 AS num_dead_tuples
	FROM activity a
`

const vacuumProgressSQLDefault string = `
WITH activity AS (
	SELECT pg_catalog.to_char(pid, 'FM0000000') AS padded_pid,
	       EXTRACT(epoch FROM a.query_start)::int::text AS query_start_epoch,
				 EXTRACT(epoch FROM COALESCE(backend_start, pg_catalog.pg_postmaster_start_time()))::int::text AS backend_start_epoch,
				 a.datname,
				 (SELECT pg_catalog.regexp_matches(query, 'autovacuum: VACUUM (ANALYZE )?([^\.]+).([^\(]+)( \(to prevent wraparound\))?'))[2] AS nspname,
				 (SELECT pg_catalog.regexp_matches(query, 'autovacuum: VACUUM (ANALYZE )?([^\.]+).([^\(]+)( \(to prevent wraparound\))?'))[3] AS relname,
				 COALESCE(a.usename, '') AS usename,
				 a.query_start,
				 a.query LIKE 'autovacuum: VACUUM%%' AS autovacuum,
				 a.query,
				 a.pid
  FROM %s a
)
SELECT (query_start_epoch || padded_pid)::bigint AS vacuum_identity,
			 (backend_start_epoch || padded_pid)::bigint AS backend_identity,
			 a.datname,
			 COALESCE(n.nspname, a.nspname) AS nspname,
			 CASE
				 WHEN ($1 = '' OR (COALESCE(n.nspname, a.nspname) || '.' || COALESCE(c.relname, a.relname)) !~* $1) THEN COALESCE(c.relname, a.relname)
				 ELSE ''
			 END AS relname,
			 a.usename,
			 a.query_start AS started_at,
			 a.autovacuum,
			 COALESCE(v.phase, '') AS phase,
			 COALESCE(v.heap_blks_total, 0) AS heap_blks_total,
			 COALESCE(v.heap_blks_scanned, 0) AS heap_blks_scanned,
			 COALESCE(v.heap_blks_vacuumed, 0) AS heap_blks_vacuumed,
			 COALESCE(v.index_vacuum_count, 0) AS index_vacuum_count,
			 COALESCE(v.max_dead_tuples, 0) AS max_dead_tuples,
			 COALESCE(v.num_dead_tuples, 0) AS num_dead_tuples
	FROM %s v
			 JOIN activity a USING (pid)
			 LEFT JOIN pg_catalog.pg_class c ON (c.oid = v.relid)
			 LEFT JOIN pg_catalog.pg_namespace n ON (n.oid = c.relnamespace)
 WHERE c.oid IS NOT NULL OR (a.query <> '<insufficient privilege>' AND a.nspname IS NOT NULL AND a.relname IS NOT NULL)
`

func GetVacuumProgress(logger *util.Logger, db *sql.DB, postgresVersion state.PostgresVersion, ignoreRegexp string) ([]state.PostgresVacuumProgress, error) {
	var activitySourceTable string
	var sql string

	if StatsHelperExists(db, "get_stat_activity") {
		activitySourceTable = "pganalyze.get_stat_activity()"
	} else {
		activitySourceTable = "pg_catalog.pg_stat_activity"
	}

	if postgresVersion.Numeric < state.PostgresVersion96 {
		sql = fmt.Sprintf(vacuumProgressSQLpg95, activitySourceTable)
	} else {
		var vacuumSourceTable string
		if StatsHelperExists(db, "get_stat_progress_vacuum") {
			vacuumSourceTable = "pganalyze.get_stat_progress_vacuum()"
		} else {
			vacuumSourceTable = "pg_catalog.pg_stat_progress_vacuum"
		}
		sql = fmt.Sprintf(vacuumProgressSQLDefault, activitySourceTable, vacuumSourceTable)
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

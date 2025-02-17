package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pganalyze/collector/state"
)

const vacuumProgressSQLdefaultFields = `
COALESCE(v.max_dead_tuples, 0) AS max_dead_item_ids,
COALESCE(v.num_dead_tuples, 0) AS num_dead_item_ids,
0,
0,
0,
0`

const vacuumProgressSQLpg17Fields = `
0,
COALESCE(v.num_dead_item_ids, 0) AS num_dead_item_ids,
COALESCE(v.dead_tuple_bytes, 0) AS dead_tuple_bytes,
COALESCE(v.max_dead_tuple_bytes, 0) AS max_dead_tuple_bytes,
COALESCE(v.indexes_total, 0) AS indexes_total,
COALESCE(v.indexes_processed, 0) AS indexes_processed`

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
			 %s
	FROM %s v
			 JOIN activity a USING (pid)
			 LEFT JOIN pg_catalog.pg_class c ON (c.oid = v.relid)
			 LEFT JOIN pg_catalog.pg_namespace n ON (n.oid = c.relnamespace)
 WHERE c.oid IS NOT NULL OR (a.query <> '<insufficient privilege>' AND a.nspname IS NOT NULL AND a.relname IS NOT NULL)
`

func GetVacuumProgress(ctx context.Context, c *Collection, db *sql.DB) ([]state.PostgresVacuumProgress, error) {
	var activitySourceTable string
	var sql string

	if StatsHelperExists(ctx, db, "get_stat_activity") {
		activitySourceTable = "pganalyze.get_stat_activity()"
	} else {
		activitySourceTable = "pg_catalog.pg_stat_activity"
	}

	var fields string
	if c.PostgresVersion.Numeric >= state.PostgresVersion17 {
		fields = vacuumProgressSQLpg17Fields
	} else {
		fields = vacuumProgressSQLdefaultFields
	}

	var vacuumSourceTable string
	if StatsHelperExists(ctx, db, "get_stat_progress_vacuum") {
		vacuumSourceTable = "pganalyze.get_stat_progress_vacuum()"
	} else {
		vacuumSourceTable = "pg_catalog.pg_stat_progress_vacuum"
	}
	sql = fmt.Sprintf(vacuumProgressSQLDefault, activitySourceTable, fields, vacuumSourceTable)

	stmt, err := db.PrepareContext(ctx, QueryMarkerSQL+sql)
	if err != nil {
		return nil, err
	}

	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, c.Config.IgnoreSchemaRegexp)
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
			&row.IndexVacuumCount, &row.MaxDeadItemIds, &row.NumDeadItemIds, &row.DeadTupleBytes,
			&row.MaxDeadTupleBytes, &row.IndexesTotal, &row.IndexesProcessed,
		)
		if err != nil {
			return nil, err
		}

		vacuums = append(vacuums, row)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	for idx, row := range vacuums {
		if row.SchemaName == "pg_toast" {
			schemaName, relationName, err := resolveToastTable(ctx, db, row.RelationName)
			if err != nil {
				c.Logger.PrintVerbose("Failed to resolve TOAST table \"%s\": %s", row.RelationName, err)
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

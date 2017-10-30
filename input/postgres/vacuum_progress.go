package postgres

import (
	"database/sql"
	"fmt"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

const vacuumProgressSQL string = `
SELECT (extract(epoch from a.query_start)::int::text || to_char(pid, 'FM000000'))::bigint,
			 (extract(epoch from COALESCE(backend_start, pg_postmaster_start_time()))::int::text || to_char(pid, 'FM000000'))::bigint,
			 v.datname,
			 n.nspname,
			 c.relname,
			 a.usename,
			 a.query_start,
			 a.query LIKE 'autovacuum:%%',
			 v.phase,
			 v.heap_blks_total,
			 v.heap_blks_scanned,
			 v.heap_blks_vacuumed,
			 v.index_vacuum_count,
			 v.max_dead_tuples,
			 v.num_dead_tuples
	FROM %s v
			 JOIN %s a USING (pid)
			 LEFT JOIN pg_class c ON (c.oid = v.relid)
			 LEFT JOIN pg_namespace n ON (n.oid = c.relnamespace)
`

func GetVacuumProgress(logger *util.Logger, db *sql.DB, postgresVersion state.PostgresVersion) ([]state.PostgresVacuumProgress, error) {
	var activitySourceTable string
	var vacuumSourceTable string

	if postgresVersion.Numeric < state.PostgresVersion96 {
		return nil, nil
	}

	if statsHelperExists(db, "get_stat_progress_vacuum") {
		vacuumSourceTable = "pganalyze.get_stat_progress_vacuum()"
	} else {
		vacuumSourceTable = "pg_stat_progress_vacuum"
	}

	if statsHelperExists(db, "get_stat_activity") {
		activitySourceTable = "pganalyze.get_stat_activity()"
	} else {
		activitySourceTable = "pg_stat_activity"
	}

	stmt, err := db.Prepare(QueryMarkerSQL + fmt.Sprintf(vacuumProgressSQL, vacuumSourceTable, activitySourceTable))
	if err != nil {
		return nil, err
	}

	defer stmt.Close()

	rows, err := stmt.Query()
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

	return vacuums, nil
}

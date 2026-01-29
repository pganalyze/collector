package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/lib/pq"
	"github.com/pganalyze/collector/state"
)

const pgBlockingPidsField = `
CASE
	WHEN COALESCE(wait_event_type, '') = 'Lock' THEN pg_blocking_pids(pid)
END
`

const activitySQL string = `
SELECT (extract(epoch from COALESCE(backend_start, pg_catalog.pg_postmaster_start_time()))::int::text || pg_catalog.to_char(pid, 'FM0000000'))::bigint,
	datid, datname, usesysid, usename, pid, application_name, client_addr::text, client_port,
	backend_start, xact_start, query_start, state_change, COALESCE(wait_event_type, '') = 'Lock' as waiting,
	backend_xid, backend_xmin, wait_event_type, wait_event, backend_type, %s, state, query, %s
FROM %s
WHERE pid IS NOT NULL`

func GetBackends(ctx context.Context, c *Collection, db *sql.DB) ([]state.PostgresBackend, error) {
	var blockingPidsField string
	var queryIdField string
	var sourceTable string

	if c.GlobalOpts.CollectPostgresLocks {
		blockingPidsField = pgBlockingPidsField
	} else {
		blockingPidsField = "NULL"
	}

	if c.PostgresVersion.Numeric >= state.PostgresVersion14 {
		queryIdField = "coalesce(query_id, 0)"
	} else {
		queryIdField = "0"
	}

	if c.HelperExists("get_stat_activity", nil) {
		sourceTable = "pganalyze.get_stat_activity()" // TODO: where is this defined?
	} else {
		sourceTable = "pg_catalog.pg_stat_activity"
	}

	stmt, err := db.PrepareContext(ctx, QueryMarkerSQL+fmt.Sprintf(activitySQL, blockingPidsField, queryIdField, sourceTable))
	if err != nil {
		return nil, err
	}

	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var activities []state.PostgresBackend

	for rows.Next() {
		var row state.PostgresBackend

		err := rows.Scan(&row.Identity, &row.DatabaseOid, &row.DatabaseName,
			&row.RoleOid, &row.RoleName, &row.Pid, &row.ApplicationName, &row.ClientAddr,
			&row.ClientPort, &row.BackendStart, &row.XactStart, &row.QueryStart,
			&row.StateChange, &row.Waiting, &row.BackendXid, &row.BackendXmin,
			&row.WaitEventType, &row.WaitEvent, &row.BackendType, pq.Array(&row.BlockedByPids),
			&row.State, &row.Query, &row.QueryId)
		if err != nil {
			return nil, err
		}

		// Special case to avoid errors for certain backends with weird names
		if c.Config.SystemType == "azure_database" && row.BackendType.Valid {
			row.BackendType.String = strings.ToValidUTF8(row.BackendType.String, "")
		}

		activities = append(activities, row)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return activities, nil
}

package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/lib/pq"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

const activitySQLDefaultOptionalFields = "waiting, NULL, NULL, NULL, NULL, NULL"
const activitySQLpg94OptionalFields = "waiting, backend_xid, backend_xmin, NULL, NULL, NULL"
const activitySQLpg96OptionalFields = `COALESCE(wait_event_type, '') = 'Lock' as waiting, backend_xid, backend_xmin, wait_event_type, wait_event, NULL`
const activitySQLpg10OptionalFields = `COALESCE(wait_event_type, '') = 'Lock' as waiting, backend_xid, backend_xmin, wait_event_type, wait_event, backend_type`

const pgBlockingPidsField = `
CASE
	WHEN COALESCE(wait_event_type, '') = 'Lock' THEN pg_blocking_pids(pid)
END
`

const activitySQL string = `
SELECT (extract(epoch from COALESCE(backend_start, pg_catalog.pg_postmaster_start_time()))::int::text || pg_catalog.to_char(pid, 'FM0000000'))::bigint,
	datid, datname, usesysid, usename, pid, application_name, client_addr::text, client_port,
	backend_start, xact_start, query_start, state_change, %s, %s, state, query
FROM %s
WHERE pid IS NOT NULL`

func GetBackends(ctx context.Context, logger *util.Logger, db *sql.DB, postgresVersion state.PostgresVersion, systemType string, collectPostgresLocks bool) ([]state.PostgresBackend, error) {
	var optionalFields string
	var blockingPidsField string
	var sourceTable string

	if postgresVersion.Numeric >= state.PostgresVersion10 {
		optionalFields = activitySQLpg10OptionalFields
	} else if postgresVersion.Numeric >= state.PostgresVersion96 {
		optionalFields = activitySQLpg96OptionalFields
	} else if postgresVersion.Numeric >= state.PostgresVersion94 {
		optionalFields = activitySQLpg94OptionalFields
	} else {
		optionalFields = activitySQLDefaultOptionalFields
	}
	if collectPostgresLocks && postgresVersion.Numeric >= state.PostgresVersion96 {
		blockingPidsField = pgBlockingPidsField
	} else {
		blockingPidsField = "NULL"
	}

	if StatsHelperExists(ctx, db, "get_stat_activity") {
		sourceTable = "pganalyze.get_stat_activity()"
	} else {
		sourceTable = "pg_catalog.pg_stat_activity"
	}

	stmt, err := db.PrepareContext(ctx, QueryMarkerSQL+fmt.Sprintf(activitySQL, optionalFields, blockingPidsField, sourceTable))
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
			&row.State, &row.Query)
		if err != nil {
			return nil, err
		}

		// Special case to avoid errors for certain backends with weird names
		if systemType == "azure_database" && row.BackendType.Valid {
			row.BackendType.String = strings.ToValidUTF8(row.BackendType.String, "")
		}

		activities = append(activities, row)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return activities, nil
}

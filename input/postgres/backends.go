package postgres

import (
	"database/sql"
	"fmt"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

const activitySQLDefaultOptionalFields = "waiting, NULL, NULL, NULL, NULL, NULL"
const activitySQLpg94OptionalFields = "waiting, backend_xid, backend_xmin, NULL, NULL, NULL"
const activitySQLpg96OptionalFields = "wait_event IS NOT NULL, backend_xid, backend_xmin, wait_event_type, wait_event, NULL"
const activitySQLpg10OptionalFields = "wait_event IS NOT NULL, backend_xid, backend_xmin, wait_event_type, wait_event, backend_type"

const activitySQL string = `SELECT (extract(epoch from COALESCE(backend_start, pg_postmaster_start_time()))::int::text || to_char(pid, 'FM000000'))::bigint,
				datid, datname, usesysid, usename, pid, application_name, client_addr::text, client_port,
				backend_start, xact_start, query_start, state_change, %s, state, query
	 FROM %s
	WHERE pid IS NOT NULL`

func GetBackends(logger *util.Logger, db *sql.DB, postgresVersion state.PostgresVersion) ([]state.PostgresBackend, error) {
	var optionalFields string
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

	if statsHelperExists(db, "get_stat_activity") {
		sourceTable = "pganalyze.get_stat_activity()"
	} else {
		sourceTable = "pg_stat_activity"
	}

	stmt, err := db.Prepare(QueryMarkerSQL + fmt.Sprintf(activitySQL, optionalFields, sourceTable))
	if err != nil {
		return nil, err
	}

	defer stmt.Close()

	rows, err := stmt.Query()
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
			&row.WaitEventType, &row.WaitEvent, &row.BackendType, &row.State, &row.Query)
		if err != nil {
			return nil, err
		}

		activities = append(activities, row)
	}

	return activities, nil
}

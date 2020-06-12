package postgres

import (
	"database/sql"
	"fmt"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

const replicationSQLPg10 string = `
SELECT in_recovery,
			 CASE WHEN in_recovery THEN NULL ELSE pg_catalog.pg_current_wal_lsn() END AS current_xlog_location,
			 COALESCE(receive_location, '0/0') >= replay_location AS is_streaming,
			 receive_location,
			 replay_location,
			 pg_catalog.pg_wal_lsn_diff(receive_location, replay_location) AS apply_byte_lag,
			 replay_ts,
			 EXTRACT(epoch FROM pg_catalog.now() - pg_catalog.pg_last_xact_replay_timestamp())::int AS replay_ts_age
	FROM (SELECT pg_catalog.pg_is_in_recovery() AS in_recovery,
							 pg_catalog.pg_last_wal_receive_lsn() AS receive_location,
							 pg_catalog.pg_last_wal_replay_lsn() AS replay_location,
							 pg_catalog.pg_last_xact_replay_timestamp() AS replay_ts) r`

const replicationSQLPg9 string = `
SELECT in_recovery,
			 CASE WHEN in_recovery THEN NULL ELSE pg_catalog.pg_current_xlog_location() END AS current_xlog_location,
			 COALESCE(receive_location, '0/0') >= replay_location AS is_streaming,
			 receive_location,
			 replay_location,
			 pg_catalog.pg_xlog_location_diff(receive_location, replay_location) AS apply_byte_lag,
			 replay_ts,
			 EXTRACT(epoch FROM pg_catalog.now() - pg_catalog.pg_last_xact_replay_timestamp())::int AS replay_ts_age
	FROM (SELECT pg_catalog.pg_is_in_recovery() AS in_recovery,
							 pg_catalog.pg_last_xlog_receive_location() AS receive_location,
							 pg_catalog.pg_last_xlog_replay_location() AS replay_location,
							 pg_catalog.pg_last_xact_replay_timestamp() AS replay_ts) r`

const replicationStandbySQLPg10 string = `
SELECT client_addr,
			 usesysid,
			 pid,
			 application_name,
			 client_hostname,
			 client_port,
			 backend_start,
			 sync_priority,
			 sync_state,
			 state,
			 sent_lsn,
			 write_lsn,
			 flush_lsn,
			 replay_lsn,
			 pg_catalog.pg_wal_lsn_diff(sent_lsn, replay_lsn) AS remote_byte_lag,
			 pg_catalog.pg_wal_lsn_diff(pg_catalog.pg_current_wal_lsn(), sent_lsn) AS local_byte_lag
	FROM %s
 WHERE client_addr IS NOT NULL`

const replicationStandbySQLPg9 string = `
SELECT client_addr,
			 usesysid,
			 pid,
			 application_name,
			 client_hostname,
			 client_port,
			 backend_start,
			 sync_priority,
			 sync_state,
			 state,
			 sent_location,
			 write_location,
			 flush_location,
			 replay_location,
			 pg_catalog.pg_xlog_location_diff(sent_location, replay_location) AS remote_byte_lag,
			 pg_catalog.pg_xlog_location_diff(pg_catalog.pg_current_xlog_location(), sent_location) AS local_byte_lag
	FROM %s
 WHERE client_addr IS NOT NULL`

func GetReplication(logger *util.Logger, db *sql.DB, postgresVersion state.PostgresVersion, systemType string) (state.PostgresReplication, error) {
	var err error
	var repl state.PostgresReplication
	var sourceTable string
	var replicationStandbySQL string
	var replicationSQL string

	if postgresVersion.IsAwsAurora {
		// Most replication functions are not supported on AWS Aurora Postgres
		return repl, nil
	}

	if statsHelperExists(db, "get_stat_replication") {
		logger.PrintVerbose("Found pganalyze.get_stat_replication() stats helper")
		sourceTable = "pganalyze.get_stat_replication()"
	} else {
		if systemType != "heroku" && !connectedAsSuperUser(db, systemType) && !connectedAsMonitoringRole(db) {
			logger.PrintInfo("Warning: You are not connecting as superuser. Please setup" +
				" the monitoring helper functions (https://github.com/pganalyze/collector#setting-up-a-restricted-monitoring-user)" +
				" or connect as superuser, to get replication statistics.")
		}
		sourceTable = "pg_stat_replication"
	}

	if postgresVersion.Numeric >= state.PostgresVersion10 {
		replicationStandbySQL = replicationStandbySQLPg10
		replicationSQL = replicationSQLPg10
	} else {
		replicationStandbySQL = replicationStandbySQLPg9
		replicationSQL = replicationSQLPg9
	}

	err = db.QueryRow(QueryMarkerSQL+replicationSQL).Scan(
		&repl.InRecovery, &repl.CurrentXlogLocation, &repl.IsStreaming,
		&repl.ReceiveLocation, &repl.ReplayLocation, &repl.ApplyByteLag,
		&repl.ReplayTimestamp, &repl.ReplayTimestampAge,
	)
	if err != nil {
		return repl, err
	}

	rows, err := db.Query(QueryMarkerSQL + fmt.Sprintf(replicationStandbySQL, sourceTable))
	if err != nil {
		return repl, err
	}
	defer rows.Close()

	for rows.Next() {
		var s state.PostgresReplicationStandby

		err := rows.Scan(&s.ClientAddr, &s.RoleOid, &s.Pid, &s.ApplicationName, &s.ClientHostname,
			&s.ClientPort, &s.BackendStart, &s.SyncPriority, &s.SyncState, &s.State,
			&s.SentLocation, &s.WriteLocation, &s.FlushLocation, &s.ReplayLocation,
			&s.RemoteByteLag, &s.LocalByteLag)
		if err != nil {
			return repl, err
		}

		repl.Standbys = append(repl.Standbys, s)
	}

	return repl, nil
}

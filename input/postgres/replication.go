package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

const replicationSQL string = `
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

const replicationStandbySQL string = `
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

func GetReplication(ctx context.Context, logger *util.Logger, db *sql.DB, postgresVersion state.PostgresVersion, systemType string) (state.PostgresReplication, error) {
	var err error
	var repl state.PostgresReplication
	var sourceTable string

	if postgresVersion.IsAwsAurora {
		// Most replication functions are not supported on AWS Aurora Postgres
		return repl, nil
	}

	if StatsHelperExists(ctx, db, "get_stat_replication") {
		logger.PrintVerbose("Found pganalyze.get_stat_replication() stats helper")
		sourceTable = "pganalyze.get_stat_replication()"
	} else {
		if systemType != "heroku" && !connectedAsSuperUser(ctx, db, systemType) && !connectedAsMonitoringRole(ctx, db) {
			logger.PrintInfo("Warning: You are not connecting as superuser. Please setup" +
				" the monitoring helper functions (https://github.com/pganalyze/collector#setting-up-a-restricted-monitoring-user)" +
				" or connect as superuser, to get replication statistics.")
		}
		sourceTable = "pg_stat_replication"
	}

	err = db.QueryRowContext(ctx, QueryMarkerSQL+replicationSQL).Scan(
		&repl.InRecovery, &repl.CurrentXlogLocation, &repl.IsStreaming,
		&repl.ReceiveLocation, &repl.ReplayLocation, &repl.ApplyByteLag,
		&repl.ReplayTimestamp, &repl.ReplayTimestampAge,
	)
	if err != nil {
		return repl, err
	}

	rows, err := db.QueryContext(ctx, QueryMarkerSQL+fmt.Sprintf(replicationStandbySQL, sourceTable))
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

	if err = rows.Err(); err != nil {
		return repl, err
	}

	return repl, nil
}

func GetIsReplica(ctx context.Context, logger *util.Logger, db *sql.DB) (bool, error) {
	isAwsAurora, err := GetIsAwsAurora(ctx, db)
	if err != nil {
		logger.PrintVerbose("Error checking Postgres version: %s", err)
		return false, err
	}

	if isAwsAurora {
		// AWS Aurora is always considered a primary for purposes of the
		// skip_if_replica flag
		return false, nil
	}

	return getIsReplica(ctx, db)
}

func getIsReplica(ctx context.Context, db *sql.DB) (bool, error) {
	var isReplica bool
	err := db.QueryRowContext(ctx, QueryMarkerSQL+"SELECT pg_catalog.pg_is_in_recovery()").Scan(&isReplica)
	return isReplica, err
}

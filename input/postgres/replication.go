package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

const replicationSQLPostgres string = `
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

// See https://docs.aws.amazon.com/AmazonRDS/latest/AuroraUserGuide/aurora_replica_status.html
const replicationSQLAurora string = `
SELECT session_id <> 'MASTER_SESSION_ID' AS in_recovery,
			 CASE WHEN session_id <> 'MASTER_SESSION_ID' THEN NULL ELSE durable_lsn END AS current_xlog_location,
			 CASE WHEN session_id <> 'MASTER_SESSION_ID' THEN true ELSE NULL END AS is_streaming,
			 highest_lsn_rcvd AS receive_location,
			 CASE WHEN session_id <> 'MASTER_SESSION_ID' THEN durable_lsn ELSE NULL END AS replay_location,
			 highest_lsn_rcvd - durable_lsn AS apply_byte_lag,
			 NULL AS replay_ts,
			 (replica_lag_in_msec / 1000)::int AS replay_ts_age
	FROM pg_catalog.aurora_replica_status() WHERE server_id = pg_catalog.aurora_db_instance_identifier()`

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

func GetReplication(ctx context.Context, c *Collection, db *sql.DB) (state.PostgresReplication, error) {
	var err error
	var repl state.PostgresReplication
	var sourceTable string
	var replicationSQL string

	if c.PostgresVersion.IsAwsAurora {
		// Old Aurora releases don't have a way to self-identify the instance, which is needed to get replication metrics
		if !auroraDbInstanceIdentifierExists(ctx, db) {
			return repl, nil
		}
		replicationSQL = replicationSQLAurora
	} else {
		replicationSQL = replicationSQLPostgres
	}

	err = db.QueryRowContext(ctx, QueryMarkerSQL+replicationSQL).Scan(
		&repl.InRecovery, &repl.CurrentXlogLocation, &repl.IsStreaming,
		&repl.ReceiveLocation, &repl.ReplayLocation, &repl.ApplyByteLag,
		&repl.ReplayTimestamp, &repl.ReplayTimestampAge,
	)
	if err != nil {
		return repl, err
	}

	// Skip follower statistics on Aurora for now - there might be a benefit to support this for monitoring
	// logical replication in the future, but it requires a bit more work since Aurora will error out
	// if you call pg_catalog.pg_current_wal_lsn() when wal_level is not logical.
	if c.PostgresVersion.IsAwsAurora {
		return repl, nil
	}

	if c.HelperExists("get_stat_replication", nil) {
		c.Logger.PrintVerbose("Found pganalyze.get_stat_replication() stats helper")
		sourceTable = "pganalyze.get_stat_replication()"
	} else {
		if c.Config.SystemType != "heroku" && !c.ConnectedAsSuperUser && !c.ConnectedAsMonitoringRole {
			c.Logger.PrintInfo("Warning: Monitoring user may have insufficient permissions to retrieve replication statistics.\n" +
				"You are not connecting as a user with the pg_monitor role or a superuser." +
				" Please make sure the monitoring user used by the collector has been granted the pg_monitor role or is a superuser.")
			if c.Config.SystemType == "aiven" {
				c.Logger.PrintInfo("For aiven, you can also set up the monitoring helper functions (https://pganalyze.com/docs/install/aiven/01_create_monitoring_user).")
			}
		}
		sourceTable = "pg_stat_replication"
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
		return getIsReplicaAurora(ctx, db)
	}

	return getIsReplica(ctx, db)
}

func getIsReplica(ctx context.Context, db *sql.DB) (bool, error) {
	var isReplica bool
	err := db.QueryRowContext(ctx, QueryMarkerSQL+"SELECT pg_catalog.pg_is_in_recovery()").Scan(&isReplica)
	return isReplica, err
}

func getIsReplicaAurora(ctx context.Context, db *sql.DB) (bool, error) {
	// The function aurora_db_instance_identifier() is not available on very old Aurora versions,
	// assume the instance is always a primary in those cases, see
	// https://docs.aws.amazon.com/AmazonRDS/latest/AuroraUserGuide/aurora_db_instance_identifier.html
	if !auroraDbInstanceIdentifierExists(ctx, db) {
		return false, nil
	}
	var isReplica bool
	err := db.QueryRowContext(ctx, QueryMarkerSQL+"SELECT session_id <> 'MASTER_SESSION_ID' FROM pg_catalog.aurora_replica_status() WHERE server_id = pg_catalog.aurora_db_instance_identifier()").Scan(&isReplica)
	return isReplica, err
}

const auroraDbInstanceIdentifierSQL string = `
SELECT 1 AS available
	FROM pg_catalog.aurora_list_builtins()
 WHERE "Name" = 'aurora_db_instance_identifier'
`

func auroraDbInstanceIdentifierExists(ctx context.Context, db *sql.DB) bool {
	var available bool

	err := db.QueryRowContext(ctx, QueryMarkerSQL+auroraDbInstanceIdentifierSQL).Scan(&available)
	if err != nil {
		return false
	}

	return available
}

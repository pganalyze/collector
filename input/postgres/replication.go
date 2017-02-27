package postgres

import (
	"database/sql"

	"github.com/pganalyze/collector/state"
)

const replicationSQL string = `
SELECT in_recovery,
			 CASE WHEN in_recovery THEN NULL ELSE pg_current_xlog_location() END AS current_xlog_location,
			 COALESCE(receive_location, '0/0') >= replay_location AS is_streaming,
			 receive_location,
			 replay_location,
			 pg_xlog_location_diff(receive_location, replay_location) AS apply_byte_lag,
			 replay_ts,
			 extract(epoch from now() - pg_last_xact_replay_timestamp())::int AS replay_ts_age
	FROM (SELECT pg_is_in_recovery() AS in_recovery,
							 pg_last_xlog_receive_location() AS receive_location,
							 pg_last_xlog_replay_location() AS replay_location,
							 pg_last_xact_replay_timestamp() AS replay_ts) r`

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
			 sent_location,
			 write_location,
			 flush_location,
			 replay_location,
			 pg_xlog_location_diff(sent_location, replay_location) AS byte_lag
	FROM pg_stat_replication`

func GetReplication(db *sql.DB) (state.PostgresReplication, error) {
	var err error
	var repl state.PostgresReplication

	err = db.QueryRow(QueryMarkerSQL+replicationSQL).Scan(
		&repl.InRecovery, &repl.CurrentXlogLocation, &repl.IsStreaming,
		&repl.ReceiveLocation, &repl.ReplayLocation, &repl.ApplyByteLag,
		&repl.ReplayTimestamp, &repl.ReplayTimestampAge,
	)
	if err != nil {
		return repl, err
	}

	rows, err := db.Query(QueryMarkerSQL + replicationStandbySQL)
	if err != nil {
		return repl, err
	}
	defer rows.Close()

	for rows.Next() {
		var s state.PostgresReplicationStandby

		err := rows.Scan(&s.ClientAddr, &s.RoleOid, &s.Pid, &s.ApplicationName, &s.ClientHostname,
			&s.ClientPort, &s.BackendStart, &s.SyncPriority, &s.SyncState, &s.State,
			&s.SentLocation, &s.WriteLocation, &s.FlushLocation, &s.ReplayLocation,
			&s.ByteLag)
		if err != nil {
			return repl, err
		}

		repl.Standbys = append(repl.Standbys, s)
	}

	return repl, nil
}

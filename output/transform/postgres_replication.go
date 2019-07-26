package transform

import (
	"github.com/golang/protobuf/ptypes"
	snapshot "github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
)

func transformPostgresReplication(s snapshot.FullSnapshot, transientState state.TransientState, roleOidToIdx OidToIdx) snapshot.FullSnapshot {
	r := transientState.Replication
	s.Replication = &snapshot.Replication{InRecovery: r.InRecovery}

	if r.CurrentXlogLocation.Valid {
		s.Replication.CurrentXlogLocation = r.CurrentXlogLocation.String
	}

	if r.IsStreaming.Valid {
		s.Replication.IsStreaming = r.IsStreaming.Bool
	}

	if r.ReceiveLocation.Valid {
		s.Replication.ReceiveLocation = r.ReceiveLocation.String
	}

	if r.ReplayLocation.Valid {
		s.Replication.ReplayLocation = r.ReplayLocation.String
	}

	if r.ApplyByteLag.Valid {
		s.Replication.ApplyByteLag = r.ApplyByteLag.Int64
	}

	if r.ReplayTimestamp.Valid {
		s.Replication.ReplayTimestamp, _ = ptypes.TimestampProto(r.ReplayTimestamp.Time)
	}

	if r.ReplayTimestampAge.Valid {
		s.Replication.ReplayTimestampAge = r.ReplayTimestampAge.Int64
	}

	for _, standby := range r.Standbys {
		idx := int32(len(s.Replication.StandbyReferences))
		s.Replication.StandbyReferences = append(s.Replication.StandbyReferences,
			&snapshot.StandbyReference{ClientAddr: standby.ClientAddr})

		info := &snapshot.StandbyInformation{
			StandbyIdx:      idx,
			RoleIdx:         roleOidToIdx[standby.RoleOid],
			Pid:             standby.Pid,
			ApplicationName: standby.ApplicationName,
			ClientPort:      standby.ClientPort,
			SyncPriority:    standby.SyncPriority,
			SyncState:       standby.SyncState,
		}
		if standby.ClientHostname.Valid {
			info.ClientHostname = standby.ClientHostname.String
		}
		info.BackendStart, _ = ptypes.TimestampProto(standby.BackendStart)
		s.Replication.StandbyInformations = append(s.Replication.StandbyInformations, info)

		stats := snapshot.StandbyStatistic{
			State: standby.State,
		}
		if standby.SentLocation.Valid {
			stats.SentLocation = standby.SentLocation.String
		}
		if standby.WriteLocation.Valid {
			stats.WriteLocation = standby.WriteLocation.String
		}
		if standby.FlushLocation.Valid {
			stats.FlushLocation = standby.FlushLocation.String
		}
		if standby.ReplayLocation.Valid {
			stats.ReplayLocation = standby.ReplayLocation.String
		}
		if standby.ByteLag.Valid {
			stats.ByteLag = standby.ByteLag.Int64
		} else {
			stats.ByteLag = -1
		}

		s.Replication.StandbyStatistics = append(s.Replication.StandbyStatistics,
			&stats)
	}

	return s
}

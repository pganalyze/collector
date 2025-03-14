package transform

import (
	snapshot "github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
)

func transformPostgresBackendCounts(s snapshot.FullSnapshot, transientState state.TransientState, roleOidToIdx OidToIdx, databaseOidToIdx OidToIdx) snapshot.FullSnapshot {
	for _, backendCount := range transientState.BackendCounts {
		backendCountStatistic := snapshot.BackendCountStatistic{
			WaitingForLock: backendCount.WaitingForLock,
			Count:          backendCount.Count,
		}

		if backendCount.DatabaseOid.Valid {
			backendCountStatistic.DatabaseIdx = databaseOidToIdx[state.Oid(backendCount.DatabaseOid.Int64)]
			backendCountStatistic.HasDatabaseIdx = true
		}

		if backendCount.RoleOid.Valid {
			backendCountStatistic.RoleIdx = roleOidToIdx[state.Oid(backendCount.RoleOid.Int64)]
			backendCountStatistic.HasRoleIdx = true
		}

		switch backendCount.State {
		case "unknown":
			backendCountStatistic.State = snapshot.BackendCountStatistic_UNKNOWN_STATE
		case "active":
			backendCountStatistic.State = snapshot.BackendCountStatistic_ACTIVE
		case "idle":
			backendCountStatistic.State = snapshot.BackendCountStatistic_IDLE
		case "idle in transaction":
			backendCountStatistic.State = snapshot.BackendCountStatistic_IDLE_IN_TRANSACTION
		case "idle in transaction (aborted)":
			backendCountStatistic.State = snapshot.BackendCountStatistic_IDLE_IN_TRANSACTION_ABORTED
		case "fastpath function call":
			backendCountStatistic.State = snapshot.BackendCountStatistic_FASTPATH_FUNCTION_CALL
		case "disabled":
			backendCountStatistic.State = snapshot.BackendCountStatistic_DISABLED
		}

		switch backendCount.BackendType {
		case "unknown":
			backendCountStatistic.BackendType = snapshot.BackendCountStatistic_UNKNOWN_TYPE
		case "autovacuum launcher":
			backendCountStatistic.BackendType = snapshot.BackendCountStatistic_AUTOVACUUM_LAUNCHER
		case "autovacuum worker":
			backendCountStatistic.BackendType = snapshot.BackendCountStatistic_AUTOVACUUM_WORKER
		case "background worker":
			backendCountStatistic.BackendType = snapshot.BackendCountStatistic_BACKGROUND_WORKER
		case "background writer":
			backendCountStatistic.BackendType = snapshot.BackendCountStatistic_BACKGROUND_WRITER
		case "client backend":
			backendCountStatistic.BackendType = snapshot.BackendCountStatistic_CLIENT_BACKEND
		case "checkpointer":
			backendCountStatistic.BackendType = snapshot.BackendCountStatistic_CHECKPOINTER
		case "startup":
			backendCountStatistic.BackendType = snapshot.BackendCountStatistic_STARTUP
		case "walreceiver":
			backendCountStatistic.BackendType = snapshot.BackendCountStatistic_WALRECEIVER
		case "walsender":
			backendCountStatistic.BackendType = snapshot.BackendCountStatistic_WALSENDER
		case "walwriter":
			backendCountStatistic.BackendType = snapshot.BackendCountStatistic_WALWRITER
		case "slotsync worker":
			backendCountStatistic.BackendType = snapshot.BackendCountStatistic_SLOTSYNC_WORKER
		}

		s.BackendCountStatistics = append(s.BackendCountStatistics, &backendCountStatistic)
	}
	return s
}

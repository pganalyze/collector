package transform

import (
	"github.com/golang/protobuf/ptypes"
	snapshot "github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
)

func transformBackendWithoutRefs(backend state.PostgresBackend) snapshot.Backend {
	var b snapshot.Backend

	b.Identity = backend.Identity
	b.Pid = backend.Pid
	b.BlockedByPids = backend.BlockedByPids
	b.BlockingPids = backend.BlockingPids
	b.IndirectlyBlockedByPids = backend.IndirectlyBlockedByPids
	b.IndirectlyBlockingPids = backend.IndirectlyBlockingPids

	if backend.ApplicationName.Valid {
		b.ApplicationName = backend.ApplicationName.String
	}

	if backend.ClientAddr.Valid {
		b.ClientAddr = backend.ClientAddr.String
	}

	if backend.BackendStart.Valid {
		b.BackendStart, _ = ptypes.TimestampProto(backend.BackendStart.Time)
	}

	if backend.XactStart.Valid {
		b.XactStart, _ = ptypes.TimestampProto(backend.XactStart.Time)
	}

	if backend.QueryStart.Valid {
		b.QueryStart, _ = ptypes.TimestampProto(backend.QueryStart.Time)
	}

	if backend.StateChange.Valid {
		b.StateChange, _ = ptypes.TimestampProto(backend.StateChange.Time)
	}

	if backend.Waiting.Valid {
		b.Waiting = backend.Waiting.Bool
	}

	if backend.State.Valid {
		b.State = backend.State.String
	}

	if backend.WaitEventType.Valid {
		b.WaitEventType = backend.WaitEventType.String
	}

	if backend.WaitEvent.Valid {
		b.WaitEvent = backend.WaitEvent.String
	}

	if backend.BackendType.Valid {
		b.BackendType = backend.BackendType.String
	}

	return b
}

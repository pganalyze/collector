package transform

import (
	snapshot "github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func transformBackendWithoutRefs(backend state.PostgresBackend) snapshot.Backend {
	var b snapshot.Backend

	b.Identity = backend.Identity
	b.Pid = backend.Pid
	b.BlockedByPids = backend.BlockedByPids

	if backend.ApplicationName.Valid {
		b.ApplicationName = backend.ApplicationName.String
	}

	if backend.ClientAddr.Valid {
		b.ClientAddr = backend.ClientAddr.String
	}

	if backend.BackendStart.Valid {
		b.BackendStart = timestamppb.New(backend.BackendStart.Time)
	}

	if backend.XactStart.Valid {
		b.XactStart = timestamppb.New(backend.XactStart.Time)
	}

	if backend.QueryStart.Valid {
		b.QueryStart = timestamppb.New(backend.QueryStart.Time)
	}

	if backend.StateChange.Valid {
		b.StateChange = timestamppb.New(backend.StateChange.Time)
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

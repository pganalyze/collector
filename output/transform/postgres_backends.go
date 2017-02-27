package transform

import (
	"github.com/golang/protobuf/ptypes"
	snapshot "github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func transformPostgresBackends(s snapshot.FullSnapshot, transientState state.TransientState, roleOidToIdx OidToIdx, databaseOidToIdx OidToIdx) snapshot.FullSnapshot {
	for _, backend := range transientState.Backends {
		b := snapshot.Backend{DatabaseIdx: databaseOidToIdx[backend.DatabaseOid], RoleIdx: roleOidToIdx[backend.RoleOid]}

		if backend.NormalizedQuery.Valid {
			key := statementKey{
				databaseOid: backend.DatabaseOid,
				userOid:     backend.RoleOid,
				fingerprint: util.FingerprintQuery(backend.NormalizedQuery.String),
			}
			value := statementValue{
				statement: state.PostgresStatement{NormalizedQuery: backend.NormalizedQuery.String},
			}

			b.QueryIdx = upsertQueryReferenceAndInformation(&s, roleOidToIdx, databaseOidToIdx, key, value)
		}

		b.Pid = backend.Pid

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

		s.Backends = append(s.Backends, &b)
	}

	return s
}

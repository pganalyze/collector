package transform

import (
	"time"

	"github.com/golang/protobuf/ptypes"
	snapshot "github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
)

type OidToIdx map[state.Oid]int32

func transformPostgres(s snapshot.FullSnapshot, newState state.PersistedState, diffState state.DiffState, transientState state.TransientState) snapshot.FullSnapshot {
	s, roleOidToIdx := transformPostgresRoles(s, transientState)
	s, databaseOidToIdx := transformPostgresDatabases(s, diffState, transientState, roleOidToIdx)
	s, typeOidToIdx := transformPostgresTypes(s, transientState, databaseOidToIdx)

	s = transformPostgresVersion(s, transientState)
	s = transformPostgresConfig(s, transientState)
	s = transformPostgresServerStats(s, diffState, transientState)
	s = transformPostgresReplication(s, transientState, roleOidToIdx)
	s = transformPostgresStatements(s, newState, diffState, transientState, roleOidToIdx, databaseOidToIdx)
	s = transformPostgresRelations(s, newState, diffState, databaseOidToIdx, typeOidToIdx, s.ServerStatistic.CurrentXactId)
	s = transformPostgresFunctions(s, newState, diffState, roleOidToIdx, databaseOidToIdx)
	s = transformPostgresBackendCounts(s, transientState, roleOidToIdx, databaseOidToIdx)
	s = transformPostgresExtensions(s, transientState, databaseOidToIdx)

	return s
}

func transformPostgresRoles(s snapshot.FullSnapshot, transientState state.TransientState) (snapshot.FullSnapshot, OidToIdx) {
	roleOidToIdx := make(OidToIdx)

	for _, role := range transientState.Roles {
		ref := snapshot.RoleReference{Name: role.Name}
		idx := int32(len(s.RoleReferences))
		s.RoleReferences = append(s.RoleReferences, &ref)
		roleOidToIdx[role.Oid] = idx
	}

	for _, role := range transientState.Roles {
		info := snapshot.RoleInformation{
			RoleIdx:            roleOidToIdx[role.Oid],
			Inherit:            role.Inherit,
			Login:              role.Login,
			CreateDb:           role.CreateDb,
			CreateRole:         role.CreateRole,
			SuperUser:          role.SuperUser,
			Replication:        role.Replication,
			BypassRls:          role.BypassRLS,
			ConnectionLimit:    role.ConnectionLimit,
			PasswordValidUntil: snapshot.NullTimeToNullTimestamp(role.PasswordValidUntil),
			Config:             role.Config,
		}

		for _, oid := range role.MemberOf {
			info.MemberOf = append(info.MemberOf, roleOidToIdx[oid])
		}

		s.RoleInformations = append(s.RoleInformations, &info)
	}

	return s, roleOidToIdx
}

func transformPostgresDatabases(s snapshot.FullSnapshot, diffState state.DiffState, transientState state.TransientState, roleOidToIdx OidToIdx) (snapshot.FullSnapshot, OidToIdx) {
	databaseOidToIdx := make(OidToIdx)

	for _, database := range transientState.Databases {
		ref := snapshot.DatabaseReference{Name: database.Name}
		idx := int32(len(s.DatabaseReferences))
		s.DatabaseReferences = append(s.DatabaseReferences, &ref)
		databaseOidToIdx[database.Oid] = idx
	}

	for _, database := range transientState.Databases {
		collectedLocalCatalog := false
		for _, databaseOid := range transientState.DatabaseOidsWithLocalCatalog {
			if databaseOid == database.Oid {
				collectedLocalCatalog = true
				break
			}
		}

		info := snapshot.DatabaseInformation{
			DatabaseIdx:               databaseOidToIdx[database.Oid],
			OwnerRoleIdx:              roleOidToIdx[database.OwnerRoleOid],
			Encoding:                  database.Encoding,
			Collate:                   database.Collate,
			CType:                     database.CType,
			IsTemplate:                database.IsTemplate,
			AllowConnections:          database.AllowConnections,
			ConnectionLimit:           database.ConnectionLimit,
			FrozenXid:                 uint32(database.FrozenXID),
			MinimumMultixactXid:       uint32(database.MinimumMultixactXID),
			CollectedLocalCatalogData: collectedLocalCatalog,
		}

		s.DatabaseInformations = append(s.DatabaseInformations, &info)

		stats, exist := diffState.DatabaseStats[database.Oid]
		if exist {
			stat := snapshot.DatabaseStatistic{
				DatabaseIdx:  databaseOidToIdx[database.Oid],
				FrozenxidAge: stats.FrozenXIDAge,
				MinmxidAge:   stats.MinMXIDAge,
				XactCommit:   stats.XactCommit,
				XactRollback: stats.XactRollback,
			}
			s.DatabaseStatictics = append(s.DatabaseStatictics, &stat)
		}
	}

	return s, databaseOidToIdx
}

func transformPostgresConfig(s snapshot.FullSnapshot, transientState state.TransientState) snapshot.FullSnapshot {
	for _, setting := range transientState.Settings {
		info := snapshot.Setting{Name: setting.Name}

		if setting.CurrentValue.Valid {
			info.CurrentValue = setting.CurrentValue.String
		}
		if setting.Unit.Valid {
			info.Unit = &snapshot.NullString{Valid: true, Value: setting.Unit.String}
		}
		if setting.BootValue.Valid {
			info.BootValue = &snapshot.NullString{Valid: true, Value: setting.BootValue.String}
		}
		if setting.ResetValue.Valid {
			info.ResetValue = &snapshot.NullString{Valid: true, Value: setting.ResetValue.String}
		}
		if setting.Source.Valid {
			info.Source = &snapshot.NullString{Valid: true, Value: setting.Source.String}
		}
		if setting.SourceFile.Valid {
			info.SourceFile = &snapshot.NullString{Valid: true, Value: setting.SourceFile.String}
		}
		if setting.SourceLine.Valid {
			info.SourceLine = &snapshot.NullString{Valid: true, Value: setting.SourceLine.String}
		}

		s.Settings = append(s.Settings, &info)
	}

	return s
}

func transformPostgresVersion(s snapshot.FullSnapshot, transientState state.TransientState) snapshot.FullSnapshot {
	s.PostgresVersion = &snapshot.PostgresVersion{
		Full:    transientState.Version.Full,
		Short:   transientState.Version.Short,
		Numeric: int64(transientState.Version.Numeric),
	}
	return s
}

func transformPostgresServerStats(s snapshot.FullSnapshot, diffState state.DiffState, transientState state.TransientState) snapshot.FullSnapshot {
	s.ServerStatistic = &snapshot.ServerStatistic{
		CurrentXactId:   int64(transientState.ServerStats.CurrentXactId),
		NextMultiXactId: int64(transientState.ServerStats.NextMultiXactId),
	}

	// Historic statement stats which are sent now since we got the query text only now
	for timeKey, diffedStats := range transientState.HistoricServerIoStats {
		// Ignore any data older than an hour, as a safety measure in case of many
		// failed full snapshot runs (which don't reset state)
		if time.Since(timeKey.CollectedAt).Hours() >= 1 {
			continue
		}

		h := snapshot.HistoricServerIoStatistics{}
		h.CollectedAt, _ = ptypes.TimestampProto(timeKey.CollectedAt)
		h.CollectedIntervalSecs = timeKey.CollectedIntervalSecs

		for k, stats := range diffedStats {
			statsOut := &snapshot.ServerIoStatistic{
				BackendType: k.BackendType,
				IoObject:    k.IoObject,
				IoContext:   k.IoContext,
				OpBytes:     stats.OpBytes,
			}
			if stats.Reads.Valid {
				statsOut.Reads = stats.Reads.Int64
			} else {
				statsOut.Reads = -1
			}
			if stats.Writes.Valid {
				statsOut.Writes = stats.Writes.Int64
			} else {
				statsOut.Writes = -1
			}
			if stats.Extends.Valid {
				statsOut.Extends = stats.Extends.Int64
			} else {
				statsOut.Extends = -1
			}
			if stats.Evictions.Valid {
				statsOut.Evictions = stats.Evictions.Int64
			} else {
				statsOut.Evictions = -1
			}
			if stats.Reuses.Valid {
				statsOut.Reuses = stats.Reuses.Int64
			} else {
				statsOut.Reuses = -1
			}
			if stats.Fsyncs.Valid {
				statsOut.Fsyncs = stats.Fsyncs.Int64
			} else {
				statsOut.Fsyncs = -1
			}
			h.Statistics = append(h.Statistics, statsOut)
		}
		s.HistoricServerIoStatistics = append(s.HistoricServerIoStatistics, &h)
	}

	return s
}

func transformPostgresExtensions(s snapshot.FullSnapshot, transientState state.TransientState, databaseOidToIdx OidToIdx) snapshot.FullSnapshot {
	for _, extension := range transientState.Extensions {
		info := snapshot.Extension{
			DatabaseIdx:   databaseOidToIdx[extension.DatabaseOid],
			ExtensionName: extension.ExtensionName,
			Version:       extension.Version,
			SchemaName:    extension.SchemaName,
		}

		s.Extensions = append(s.Extensions, &info)
	}
	return s
}

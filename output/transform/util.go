package transform

import (
	"bytes"
	"encoding/binary"

	snapshot "github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

type statementKey struct {
	databaseOid state.Oid
	userOid     state.Oid
	fingerprint uint64
}

type statementValue struct {
	statement      state.PostgresStatement
	statementStats state.DiffedPostgresStatementStats
	queryIDs       []int64
}

func upsertQueryReferenceAndInformation(s *snapshot.FullSnapshot, statementTexts state.PostgresStatementTextMap, roleOidToIdx OidToIdx, databaseOidToIdx OidToIdx, key statementKey, value statementValue) int32 {
	fpBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(fpBuf, key.fingerprint)
	newRef := snapshot.QueryReference{
		DatabaseIdx: databaseOidToIdx[key.databaseOid],
		RoleIdx:     roleOidToIdx[key.userOid],
		Fingerprint: fpBuf,
	}

	for idx, ref := range s.QueryReferences {
		if ref.DatabaseIdx == newRef.DatabaseIdx && ref.RoleIdx == newRef.RoleIdx &&
			bytes.Equal(ref.Fingerprint, newRef.Fingerprint) {
			return int32(idx)
		}
	}

	idx := int32(len(s.QueryReferences))
	s.QueryReferences = append(s.QueryReferences, &newRef)

	// Information
	normalizedQuery := ""
	if value.statement.QueryTextUnavailable {
		normalizedQuery = "<query text unavailable>"
	} else if value.statement.InsufficientPrivilege {
		normalizedQuery = "<insufficient privilege>"
	} else if value.statement.Collector {
		normalizedQuery = "<pganalyze-collector>"
	} else {
		normalizedQuery, _ = statementTexts[key.fingerprint]
	}
	queryInformation := snapshot.QueryInformation{
		QueryIdx:        idx,
		NormalizedQuery: normalizedQuery,
		QueryIds:        value.queryIDs,
	}
	s.QueryInformations = append(s.QueryInformations, &queryInformation)

	return idx
}

func upsertQueryReferenceAndInformationSimple(server *state.Server, refs []*snapshot.QueryReference, infos []*snapshot.QueryInformation, roleIdx int32, databaseIdx int32, originalQuery string, trackActivityQuerySize int) (int32, []*snapshot.QueryReference, []*snapshot.QueryInformation) {
	fingerprint := util.FingerprintQuery(originalQuery)

	fpBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(fpBuf, fingerprint)
	newRef := snapshot.QueryReference{
		DatabaseIdx: databaseIdx,
		RoleIdx:     roleIdx,
		Fingerprint: fpBuf,
	}

	for idx, ref := range refs {
		if ref.DatabaseIdx == newRef.DatabaseIdx && ref.RoleIdx == newRef.RoleIdx &&
			bytes.Equal(ref.Fingerprint, newRef.Fingerprint) {
			return int32(idx), refs, infos
		}
	}

	idx := int32(len(refs))
	refs = append(refs, &newRef)

	// Information
	queryInformation := snapshot.QueryInformation{
		QueryIdx:        idx,
		NormalizedQuery: util.NormalizeQuery(originalQuery, server.Config.FilterQueryText, trackActivityQuerySize),
	}
	infos = append(infos, &queryInformation)

	return idx, refs, infos
}

func upsertRoleReference(refs []*snapshot.RoleReference, roleName string) (int32, []*snapshot.RoleReference) {
	newRef := snapshot.RoleReference{Name: roleName}

	for idx, ref := range refs {
		if ref.Name == newRef.Name {
			return int32(idx), refs
		}
	}

	idx := int32(len(refs))
	refs = append(refs, &newRef)

	return idx, refs
}

func upsertDatabaseReference(refs []*snapshot.DatabaseReference, databaseName string) (int32, []*snapshot.DatabaseReference) {
	newRef := snapshot.DatabaseReference{Name: databaseName}

	for idx, ref := range refs {
		if ref.Name == newRef.Name {
			return int32(idx), refs
		}
	}

	idx := int32(len(refs))
	refs = append(refs, &newRef)

	return idx, refs
}

func upsertRelationReference(refs []*snapshot.RelationReference, databaseIdx int32, schemaName string, relationName string) (int32, []*snapshot.RelationReference) {
	newRef := snapshot.RelationReference{DatabaseIdx: databaseIdx, SchemaName: schemaName, RelationName: relationName}

	for idx, ref := range refs {
		if ref.DatabaseIdx == newRef.DatabaseIdx && ref.SchemaName == newRef.SchemaName && ref.RelationName == newRef.RelationName {
			return int32(idx), refs
		}
	}

	idx := int32(len(refs))
	refs = append(refs, &newRef)

	return idx, refs
}

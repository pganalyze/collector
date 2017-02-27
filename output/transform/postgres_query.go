package transform

import (
	"bytes"

	snapshot "github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
)

type statementKey struct {
	databaseOid state.Oid
	userOid     state.Oid
	fingerprint [21]byte
}

type statementValue struct {
	statement      state.PostgresStatement
	statementStats state.DiffedPostgresStatementStats
	queryIDs       []int64
}

func upsertQueryReferenceAndInformation(s *snapshot.FullSnapshot, roleOidToIdx OidToIdx, databaseOidToIdx OidToIdx, key statementKey, value statementValue) int32 {
	newRef := snapshot.QueryReference{
		DatabaseIdx: databaseOidToIdx[key.databaseOid],
		RoleIdx:     roleOidToIdx[key.userOid],
		Fingerprint: key.fingerprint[:],
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
	queryInformation := snapshot.QueryInformation{
		QueryIdx:        idx,
		NormalizedQuery: value.statement.NormalizedQuery,
		QueryIds:        value.queryIDs,
	}
	s.QueryInformations = append(s.QueryInformations, &queryInformation)

	return idx
}

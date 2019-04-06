package transform

import (
	"bytes"

	snapshot "github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
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

func upsertQueryReferenceAndInformation(s *snapshot.FullSnapshot, statementTexts state.PostgresStatementTextMap, roleOidToIdx OidToIdx, databaseOidToIdx OidToIdx, key statementKey, value statementValue) int32 {
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
	normalizedQuery := ""
	if value.statement.Unidentified {
		normalizedQuery = "<unidentified queryid>"
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

func upsertQueryReferenceAndInformationSimple(refs []*snapshot.QueryReference, infos []*snapshot.QueryInformation, roleIdx int32, databaseIdx int32, originalQuery string) (int32, []*snapshot.QueryReference, []*snapshot.QueryInformation) {
	fingerprint := util.FingerprintQuery(originalQuery)

	newRef := snapshot.QueryReference{
		DatabaseIdx: databaseIdx,
		RoleIdx:     roleIdx,
		Fingerprint: fingerprint[:],
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
		NormalizedQuery: util.NormalizeQuery(originalQuery),
	}
	infos = append(infos, &queryInformation)

	return idx, refs, infos
}

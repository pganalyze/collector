package transform

import (
	"github.com/golang/protobuf/ptypes"
	snapshot "github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
)

func transformPostgresQuerySamples(s snapshot.FullSnapshot, transientState state.TransientState, roleNameToIdx NameToIdx, databaseNameToIdx NameToIdx) snapshot.FullSnapshot {
	for _, sampleIn := range transientState.QuerySamples {
		occurredAt, _ := ptypes.TimestampProto(sampleIn.OccurredAt)
		roleIdx, hasRoleIdx := roleNameToIdx[sampleIn.Username]
		databaseIdx, hasDatabaseIdx := databaseNameToIdx[sampleIn.Database]

		if !hasRoleIdx || !hasDatabaseIdx || sampleIn.Query == "" {
			continue
		}

		queryIdx := upsertQueryReferenceAndInformationSimple(
			&s,
			roleIdx,
			databaseIdx,
			sampleIn.Query,
		)

		sample := snapshot.QuerySample{
			QueryIdx:      queryIdx,
			OccurredAt:    occurredAt,
			RuntimeMs:     sampleIn.RuntimeMs,
			OriginalQuery: sampleIn.Query,

			HasExplain:    sampleIn.HasExplain,
			ExplainOutput: sampleIn.ExplainOutput,
			ExplainError:  sampleIn.ExplainError,
		}
		s.QuerySamples = append(s.QuerySamples, &sample)
	}

	return s
}

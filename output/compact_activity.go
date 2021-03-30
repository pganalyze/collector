package output

import (
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/output/transform"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	pg_query "github.com/pganalyze/pg_query_go/v2"
)

func SubmitCompactActivitySnapshot(server *state.Server, grant state.Grant, collectionOpts state.CollectionOpts, logger *util.Logger, activityState state.TransientActivityState) error {
	as, r := transform.ActivityStateToCompactActivitySnapshot(server, activityState)

	if server.Config.FilterQuerySample == "all" {
		for idx, backend := range as.Backends {
			if backend.QueryText != "" {
				as.Backends[idx].QueryText, _ = pg_query.Normalize(backend.QueryText)
			}
		}
	}

	s := pganalyze_collector.CompactSnapshot{
		BaseRefs: &r,
		Data:     &pganalyze_collector.CompactSnapshot_ActivitySnapshot{ActivitySnapshot: &as},
	}
	return uploadAndSubmitCompactSnapshot(s, grant, server, collectionOpts, logger, activityState.CollectedAt, false, "activity")
}

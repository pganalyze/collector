package output

import (
	"context"

	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/output/transform"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func SubmitCompactActivitySnapshot(ctx context.Context, server *state.Server, grant state.Grant, collectionOpts state.CollectionOpts, logger *util.Logger, activityState state.TransientActivityState) error {
	as, r := transform.ActivityStateToCompactActivitySnapshot(server, activityState)

	if server.Config.FilterQuerySample != "none" {
		for idx, backend := range as.Backends {
			// Normalize can be slow, protect against edge cases here by checking for cancellations
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				if backend.QueryText != "" {
					as.Backends[idx].QueryText = util.NormalizeQuery(backend.QueryText, "unparseable", activityState.TrackActivityQuerySize)
				}
			}
		}
	}

	s := pganalyze_collector.CompactSnapshot{
		BaseRefs: &r,
		Data:     &pganalyze_collector.CompactSnapshot_ActivitySnapshot{ActivitySnapshot: &as},
	}
	return uploadAndSubmitCompactSnapshot(ctx, s, grant, server, collectionOpts, logger, activityState.CollectedAt, false, "activity")
}

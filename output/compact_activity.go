package output

import (
	"context"

	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/output/transform"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func SubmitCompactActivitySnapshot(ctx context.Context, server *state.Server, grant state.Grant, collectionOpts state.CollectionOpts, logger *util.Logger, activityState state.TransientActivityState) error {
	as, r := transform.ActivityStateToCompactActivitySnapshot(server, activityState)

	if server.Config.FilterQuerySample != "" && server.Config.FilterQuerySample != "none" {
		for idx, backend := range as.Backends {
			// Normalize can be slow, protect against edge cases here by checking for cancellations
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				if backend.QueryText != "" {
					// We pass "unparseable" here as the implied value of the filter_query_text setting, since for historic
					// reasons this conditional here is based on filter_query_sample. The intent is that if the query is
					// unparseable, it turns into "<truncated query>" or "<unparseable query>", not the original text.
					as.Backends[idx].QueryText = util.NormalizeQuery(backend.QueryText, "unparseable", activityState.TrackActivityQuerySize)
				}
			}
		}
	}

	server.QueryRunsMutex.Lock()
	for _, query := range server.QueryRuns {
		as.QueryRuns = append(as.QueryRuns, &pganalyze_collector.QueryRun{
			Id:         query.Id,
			StartedAt:  timestamppb.New(query.StartedAt),
			FinishedAt: timestamppb.New(query.FinishedAt),
			Result:     query.Result,
			Error:      query.Error,
			BackendPid: int32(query.BackendPid),
		})
	}
	server.QueryRunsMutex.Unlock()

	s := pganalyze_collector.CompactSnapshot{
		BaseRefs: &r,
		Data:     &pganalyze_collector.CompactSnapshot_ActivitySnapshot{ActivitySnapshot: &as},
	}
	return uploadAndSubmitCompactSnapshot(ctx, s, grant, server, collectionOpts, logger, activityState.CollectedAt, false, "activity")
}

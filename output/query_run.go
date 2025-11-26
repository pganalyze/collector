package output

import (
	"context"
	"time"

	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func SubmitQueryRunSnapshot(ctx context.Context, server *state.Server, collectionOpts state.CollectionOpts, logger *util.Logger, query state.QueryRun) {
	s := pganalyze_collector.CompactSnapshot{
		Data: &pganalyze_collector.CompactSnapshot_QueryRunSnapshot{QueryRunSnapshot: &pganalyze_collector.QueryRunSnapshot{QueryRun: &pganalyze_collector.QueryRun{
			Id:         query.Id,
			StartedAt:  timestamppb.New(query.StartedAt),
			FinishedAt: timestamppb.New(query.FinishedAt),
			Result:     query.Result,
			Error:      query.Error,
			BackendPid: int32(query.BackendPid),
		}}},
	}
	uploadAndSubmitCompactSnapshot(ctx, s, server, collectionOpts, logger, time.Now(), false, "query_run")
}

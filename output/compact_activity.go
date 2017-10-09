package output

import (
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/output/transform"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func SubmitCompactActivitySnapshot(server state.Server, grant state.Grant, collectionOpts state.CollectionOpts, logger *util.Logger, activityState state.ActivityState) error {
	as, r := transform.ActivityStateToCompactActivitySnapshot(activityState)

	s := pganalyze_collector.CompactSnapshot{
		BaseRefs: &r,
		Data:     &pganalyze_collector.CompactSnapshot_ActivitySnapshot{ActivitySnapshot: &as},
	}
	return uploadAndSubmitCompactSnapshot(s, grant.S3(), server, collectionOpts, logger, activityState.CollectedAt, false, "activity")
}

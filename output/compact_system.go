package output

import (
	"time"

	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/output/transform"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func SubmitCompactSystemSnapshot(server state.Server, grant state.Grant, collectionOpts state.CollectionOpts, logger *util.Logger, systemState state.SystemState, collectedAt time.Time) error {
	ss := transform.SystemStateToCompactSystemSnapshot(systemState)

	s := pganalyze_collector.CompactSnapshot{
		Data: &pganalyze_collector.CompactSnapshot_SystemSnapshot{SystemSnapshot: &ss},
	}
	return uploadAndSubmitCompactSnapshot(s, grant, server, collectionOpts, logger, collectedAt, false, "system")
}

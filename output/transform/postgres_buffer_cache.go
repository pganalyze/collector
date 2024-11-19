package transform

import (
	snapshot "github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
)

func transformPostgresBufferCache(s snapshot.FullSnapshot, ts state.TransientState, databaseOidToIdx OidToIdx) snapshot.FullSnapshot {
	for databaseOid, bufferCache := range ts.BufferCache {
		databaseIdx, ok := databaseOidToIdx[databaseOid]
		if ok {
			var untrackedBytes int64
			for _, bytes := range bufferCache {
				untrackedBytes += bytes
				continue
			}
			s.DatabaseStatictics[databaseIdx].UntrackedCacheBytes = untrackedBytes
		}

	}
	return s
}

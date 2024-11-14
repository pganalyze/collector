package postgres

import (
	"context"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

// See also https://www.postgresql.org/docs/current/pgbuffercache.html
const bufferCacheSQL string = `
SELECT reldatabase, relfilenode, count(*) * current_setting('block_size')::int
FROM pg_buffercache
GROUP BY 1, 2`

const bufferCacheSizeSQL string = `
SELECT pg_size_bytes(unit) * pg_size_bytes(setting) / 1024 / 1024 / 1024
FROM pg_settings
WHERE name = 'shared_buffers'
`

func GetBufferCache(ctx context.Context, server *state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger, postgresVersion state.PostgresVersion, channel chan state.BufferCache) {
	bufferCache := make(state.BufferCache)
	db, err := EstablishConnection(ctx, server, logger, globalCollectionOpts, "")
	if err != nil {
		logger.PrintError("GetBufferCache: %s", err)
		channel <- bufferCache
		return
	}

	extensionEnabled := false
	db.QueryRowContext(ctx, QueryMarkerSQL+"SELECT true FROM pg_extension WHERE extname = 'pg_buffercache'").Scan(&extensionEnabled)
	if !extensionEnabled {
		channel <- bufferCache
		return
	}

	sizeGB := 0
	db.QueryRowContext(ctx, QueryMarkerSQL+bufferCacheSizeSQL).Scan(&sizeGB)
	if sizeGB > server.Config.MaxBufferCacheMonitoringGB {
		logger.PrintWarning("GetBufferCache: skipping collection. To enable, set max_buffer_cache_monitoring_gb to a value over %d", sizeGB)
		channel <- bufferCache
		return
	}

	stmt, err := db.PrepareContext(ctx, QueryMarkerSQL+bufferCacheSQL)
	if err != nil {
		logger.PrintError("GetBufferCache: %s", err)
		channel <- bufferCache
		return
	}
	defer stmt.Close()
	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		logger.PrintError("GetBufferCache: %s", err)
		channel <- bufferCache
		return
	}
	defer rows.Close()
	for rows.Next() {
		var reldatabase state.Oid
		var relfilenode state.Oid
		var bytes int64
		err = rows.Scan(&reldatabase, &relfilenode, &bytes)
		if err != nil {
			logger.PrintError("GetBufferCache: %s", err)
			channel <- bufferCache
			return
		}
		db, ok := bufferCache[reldatabase]
		if ok {
			db[relfilenode] = bytes
		} else {
			bufferCache[reldatabase] = map[state.Oid]int64{relfilenode: bytes}
		}
	}
	channel <- bufferCache
}

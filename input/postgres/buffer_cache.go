package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/pganalyze/collector/state"
)

const bufferCacheExtensionSQL string = `
SELECT COALESCE((
	SELECT nspname
	FROM pg_catalog.pg_extension
	INNER JOIN pg_catalog.pg_namespace n ON extnamespace = n.oid
	WHERE extname = 'pg_buffercache'
), '')
`

const bufferCacheSizeSQL string = `
SELECT pg_catalog.pg_size_bytes(unit) * pg_catalog.pg_size_bytes(setting) / 1024 / 1024 / 1024
FROM pg_catalog.pg_settings
WHERE name = 'shared_buffers'
`

// https://www.postgresql.org/docs/current/pgbuffercache.html
const bufferCacheSQL string = `
SELECT reldatabase, relfilenode, count(*) * current_setting('block_size')::int
FROM %s.pg_buffercache
WHERE reldatabase IS NOT NULL -- filters out unused pages
GROUP BY 1, 2
`

func GetBufferCache(ctx context.Context, c *Collection, server *state.Server, opts state.CollectionOpts, channel chan state.BufferCache) {
	start := time.Now()
	bufferCache := make(state.BufferCache)
	db, err := EstablishConnection(ctx, server, c.Logger, opts, "")
	if err != nil {
		c.Logger.PrintError("GetBufferCache: %s", err)
		channel <- bufferCache
		return
	}
	defer db.Close()

	schemaName := ""
	db.QueryRowContext(ctx, QueryMarkerSQL+bufferCacheExtensionSQL).Scan(&schemaName)
	if schemaName == "" {
		channel <- bufferCache
		return
	}

	sizeGB := 0
	db.QueryRowContext(ctx, QueryMarkerSQL+bufferCacheSizeSQL).Scan(&sizeGB)
	if sizeGB > server.Config.MaxBufferCacheMonitoringGB {
		if opts.TestRun {
			c.Logger.PrintWarning("GetBufferCache: skipping collection. To enable, set max_buffer_cache_monitoring_gb to a value over %d", sizeGB)
		}
		channel <- bufferCache
		return
	}

	stmt, err := db.PrepareContext(ctx, QueryMarkerSQL+fmt.Sprintf(bufferCacheSQL, schemaName))
	if err != nil {
		c.Logger.PrintError("GetBufferCache: %s", err)
		channel <- bufferCache
		return
	}
	defer stmt.Close()
	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		c.Logger.PrintError("GetBufferCache: %s", err)
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
			c.Logger.PrintError("GetBufferCache: %s", err)
			channel <- bufferCache
			return
		}
		b, ok := bufferCache[reldatabase]
		if ok {
			b[relfilenode] = bytes
		} else {
			bufferCache[reldatabase] = map[state.Oid]int64{relfilenode: bytes}
		}
	}

	c.Logger.PrintVerbose("GetBufferCache: finished after %s", time.Since(start))
	channel <- bufferCache
}

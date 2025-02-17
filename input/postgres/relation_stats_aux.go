package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pganalyze/collector/state"
)

const citusRelationSizeSQL = `
SELECT logicalrelid::oid,
			 CASE
			   WHEN coalesce(current_setting('citus.shard_replication_factor')::integer, 1) = 1
				 THEN pg_catalog.citus_table_size(logicalrelid)
				 ELSE 0
			 END AS citus_table_size
	FROM pg_catalog.pg_dist_partition dp
			 INNER JOIN pg_catalog.pg_class c ON (dp.logicalrelid::oid = c.oid)
			 INNER JOIN pg_catalog.pg_namespace n ON (c.relnamespace = n.oid)
 WHERE ($1 = '' OR (n.nspname || '.' || c.relname) !~* $1)
`

func collectCitusRelationStats(c *Collection) bool {
	return c.PostgresVersion.IsCitus && c.Config.DisableCitusSchemaStats != "all"
}

func handleRelationStatsAux(ctx context.Context, c *Collection, db *sql.DB, relStats state.PostgresRelationStatsMap) (state.PostgresRelationStatsMap, error) {
	if !collectCitusRelationStats(c) {
		return relStats, nil
	}

	stmt, err := db.PrepareContext(ctx, QueryMarkerSQL+citusRelationSizeSQL)
	if err != nil {
		return relStats, fmt.Errorf("RelationStatsExt/Prepare: %s", err)
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, c.Config.IgnoreSchemaRegexp)
	if err != nil {
		return relStats, fmt.Errorf("RelationStatsExt/Query: %s", err)
	}
	defer rows.Close()

	for rows.Next() {
		var oid state.Oid
		var sizeBytes int64

		err = rows.Scan(&oid, &sizeBytes)
		if err != nil {
			return relStats, fmt.Errorf("RelationStatsExt/Scan: %s", err)
		}
		s := relStats[oid]
		s.SizeBytes = sizeBytes
		s.ToastSizeBytes = 0
		relStats[oid] = s
	}

	if err = rows.Err(); err != nil {
		return relStats, fmt.Errorf("RelationStatsExt/Rows: %s", err)
	}

	return relStats, nil
}

const citusIndexSizeSQL = `
WITH dist_idx_shard_stats_raw AS (
	SELECT
		dp.logicalrelid::regclass,
		(pg_catalog.run_command_on_shards(logicalrelid::regclass::text, $$
			SELECT
				jsonb_agg(shard_idx_stats)
			FROM (
				SELECT
					relnamespace::regnamespace AS idx_shard_schema,
					indexrelid::regclass AS idx_shard_name,
					COALESCE(pg_catalog.pg_relation_size(indexrelid), 0) AS idx_shard_bytes
				FROM
					pg_stat_user_indexes pgsui INNER JOIN pg_class pgc ON pgc.oid = pgsui.relid
				WHERE
					relid = '%s'::regclass
			) AS shard_idx_stats
		$$)).*
	FROM
		pg_dist_partition dp INNER JOIN pg_catalog.pg_class c ON (dp.logicalrelid::oid = c.oid)
			 INNER JOIN pg_catalog.pg_namespace n ON (c.relnamespace = n.oid)
	WHERE
		($1 = '' OR (n.nspname || '.' || c.relname) !~* $1)
), dist_idx_shard_stats AS (
	SELECT
		shardid,
		bool_and(success) OVER() AS all_success,
		jsonb_array_elements(result::jsonb) AS shard_info
	FROM
		dist_idx_shard_stats_raw
	WHERE
		result <> ''
)
SELECT
	pgc.oid,
	sum((shard_info ->> 'idx_shard_bytes')::bigint) AS total_size_bytes
FROM
  pg_class pgc INNER JOIN dist_idx_shard_stats ON (
		pgc.relkind = 'i'
			AND pgc.relnamespace::regnamespace::text = (shard_info ->> 'idx_shard_schema')
			AND pgc.relname = pg_catalog.regexp_replace(shard_info ->> 'idx_shard_name', '_' || shardid || '$', '')
	)
WHERE
	all_success
GROUP BY
  oid;
`

func collectCitusIndexStats(c *Collection) bool {
	return c.PostgresVersion.IsCitus && c.Config.DisableCitusSchemaStats != "all" && c.Config.DisableCitusSchemaStats != "index"
}

func handleIndexStatsAux(ctx context.Context, c *Collection, db *sql.DB, idxStats state.PostgresIndexStatsMap) (state.PostgresIndexStatsMap, error) {
	if !collectCitusIndexStats(c) {
		return idxStats, nil
	}

	stmt, err := db.PrepareContext(ctx, QueryMarkerSQL+citusIndexSizeSQL)
	if err != nil {
		return idxStats, fmt.Errorf("IndexStatsExt/Prepare: %s", err)
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, c.Config.IgnoreSchemaRegexp)
	if err != nil {
		return idxStats, fmt.Errorf("IndexStatsExt/Query: %s", err)
	}
	defer rows.Close()

	for rows.Next() {
		var oid state.Oid
		var sizeBytes int64

		err = rows.Scan(&oid, &sizeBytes)
		if err != nil {
			return idxStats, fmt.Errorf("IndexStatsExt/Scan: %s", err)
		}
		s := idxStats[oid]
		s.SizeBytes = sizeBytes
		idxStats[oid] = s
	}

	if err = rows.Err(); err != nil {
		return idxStats, fmt.Errorf("IndexStatsExt/Rows: %s", err)
	}

	return idxStats, nil
}

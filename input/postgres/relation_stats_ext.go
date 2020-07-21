package postgres

import (
	"database/sql"
	"fmt"

	"github.com/pganalyze/collector/state"
)

const citusRelationSizeSQL = `
SELECT logicalrelid::oid,
       pg_catalog.citus_table_size(logicalrelid)
	FROM pg_catalog.pg_dist_partition dp
			 INNER JOIN pg_catalog.pg_class c ON (dp.logicalrelid::oid == c.oid)
			 INNER JOIN pg_catalog.pg_namespace n ON (c.relnamespace = n.oid)
 WHERE ($1 = '' OR (n.nspname || '.' || c.relname) !~* $1)
`

func handleRelationStatsExt(db *sql.DB, relStats state.PostgresRelationStatsMap, postgresVersion state.PostgresVersion, ignoreRegexp string) (state.PostgresRelationStatsMap, error) {
	if postgresVersion.IsCitus {
		stmt, err := db.Prepare(QueryMarkerSQL + citusRelationSizeSQL)
		if err != nil {
			return relStats, fmt.Errorf("RelationStatsExt/Prepare: %s", err)
		}
		defer stmt.Close()

		rows, err := stmt.Query()
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
	}

	return relStats, nil
}

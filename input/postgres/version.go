package postgres

import (
	"context"
	"database/sql"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

// GetPostgresVersion - Reads the version of the connected PostgreSQL server
func GetPostgresVersion(ctx context.Context, logger *util.Logger, db *sql.DB) (version state.PostgresVersion, err error) {
	err = db.QueryRowContext(ctx, QueryMarkerSQL+"SELECT pg_catalog.version()").Scan(&version.Full)
	if err != nil {
		return
	}

	err = db.QueryRowContext(ctx, QueryMarkerSQL+"SHOW server_version").Scan(&version.Short)
	if err != nil {
		return
	}

	err = db.QueryRowContext(ctx, QueryMarkerSQL+"SHOW server_version_num").Scan(&version.Numeric)
	if err != nil {
		return
	}

	isAwsAurora, err := GetIsAwsAurora(ctx, db)
	if err != nil {
		return
	}
	version.IsAwsAurora = isAwsAurora

	err = db.QueryRowContext(ctx, QueryMarkerSQL+"SELECT pg_catalog.count(1) = 1 FROM pg_extension WHERE extname = 'citus'").Scan(&version.IsCitus)
	if err != nil {
		return
	}

	logger.PrintVerbose("Detected PostgreSQL Version %d (%s)", version.Numeric, version.Full)

	return
}

func GetIsAwsAurora(ctx context.Context, db *sql.DB) (bool, error) {
	var isAurora bool
	err := db.QueryRowContext(ctx, QueryMarkerSQL+"SELECT pg_catalog.count(1) = 1 FROM pg_settings WHERE name = 'rds.extensions' AND setting LIKE '%aurora_stat_utils%'").Scan(&isAurora)
	return isAurora, err
}

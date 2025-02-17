package postgres

import (
	"context"
	"database/sql"
	"strings"

	"github.com/pganalyze/collector/state"
)

// getPostgresVersion - Reads the version of the connected PostgreSQL server
func getPostgresVersion(ctx context.Context, db *sql.DB) (version state.PostgresVersion, err error) {
	err = db.QueryRowContext(ctx, QueryMarkerSQL+"SELECT pg_catalog.version()").Scan(&version.Full)
	if err != nil {
		return
	}
	version.IsEPAS = strings.Contains(version.Full, "EnterpriseDB Advanced Server")

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

	return
}

func GetIsAwsAurora(ctx context.Context, db *sql.DB) (bool, error) {
	var isAurora bool
	err := db.QueryRowContext(ctx, QueryMarkerSQL+"SELECT pg_catalog.count(1) = 1 FROM pg_settings WHERE name = 'rds.extensions' AND setting LIKE '%aurora_stat_utils%'").Scan(&isAurora)
	return isAurora, err
}

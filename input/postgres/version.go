package postgres

import (
	"database/sql"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

// GetPostgresVersion - Reads the version of the connected PostgreSQL server
func GetPostgresVersion(logger *util.Logger, db *sql.DB) (version state.PostgresVersion, err error) {
	err = db.QueryRow(QueryMarkerSQL + "SELECT pg_catalog.version()").Scan(&version.Full)
	if err != nil {
		return
	}

	err = db.QueryRow(QueryMarkerSQL + "SHOW server_version").Scan(&version.Short)
	if err != nil {
		return
	}

	err = db.QueryRow(QueryMarkerSQL + "SHOW server_version_num").Scan(&version.Numeric)
	if err != nil {
		return
	}

	isAwsAurora, err := GetIsAwsAurora(db)
	if err != nil {
		return
	}
	version.IsAwsAurora = isAwsAurora

	err = db.QueryRow(QueryMarkerSQL + "SELECT pg_catalog.count(1) = 1 FROM pg_extension WHERE extname = 'citus'").Scan(&version.IsCitus)
	if err != nil {
		return
	}

	logger.PrintVerbose("Detected PostgreSQL Version %d (%s)", version.Numeric, version.Full)

	return
}

func GetIsAwsAurora(db *sql.DB) (bool, error) {
	var isAurora bool
	err := db.QueryRow(QueryMarkerSQL + "SELECT pg_catalog.count(1) = 1 FROM pg_settings WHERE name = 'rds.extensions' AND setting LIKE '%aurora_stat_utils%'").Scan(&isAurora)
	return isAurora, err
}

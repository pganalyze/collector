package postgres

import (
	"database/sql"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

// GetPostgresVersion - Reads the version of the connected PostgreSQL server
func GetPostgresVersion(logger *util.Logger, db *sql.DB) (version state.PostgresVersion, err error) {
	err = db.QueryRow(QueryMarkerSQL + "SELECT version()").Scan(&version.Full)
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

	logger.PrintVerbose("Detected PostgreSQL Version %d (%s)", version.Numeric, version.Full)

	return
}

package dbstats

import (
	"database/sql"

	"github.com/pganalyze/collector/util"
)

const (
	PostgresVersion92 = 90200
	PostgresVersion93 = 90300
	PostgresVersion94 = 90400
	PostgresVersion95 = 90500
	PostgresVersion96 = 90600

	// MinRequiredPostgresVersion - We require PostgreSQL 9.2 or newer, since pg_stat_statements only started being usable then
	MinRequiredPostgresVersion = PostgresVersion92
)

type PostgresVersion struct {
	Full    string `json:"full"`
	Short   string `json:"short"`
	Numeric int    `json:"numeric"`
}

// GetPostgresVersion - Reads the version of the connected PostgreSQL server
func GetPostgresVersion(logger *util.Logger, db *sql.DB) (version PostgresVersion, err error) {
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

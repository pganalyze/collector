package postgres

import (
	"database/sql"

	"github.com/pganalyze/collector/util"
)

// GetDataDirectory - Finds the location of the data directory
func GetDataDirectory(logger *util.Logger, db *sql.DB) (dataDirectory string, err error) {
	err = db.QueryRow(QueryMarkerSQL + "SHOW data_directory").Scan(&dataDirectory)
	if err != nil {
		return
	}

	return
}

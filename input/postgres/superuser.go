package postgres

import (
	"database/sql"
	"fmt"
)

const connectedAsSuperUserSQL string = `SELECT current_setting('is_superuser') = 'on'`

func connectedAsSuperUser(db *sql.DB) bool {
	var enabled bool

	err := db.QueryRow(QueryMarkerSQL + connectedAsSuperUserSQL).Scan(&enabled)
	if err != nil {
		return false
	}

	return enabled
}

const statsHelperSQL string = `
SELECT 1 AS enabled
	FROM pg_proc
	JOIN pg_namespace ON (pronamespace = pg_namespace.oid)
 WHERE nspname = 'pganalyze' AND proname = '%s'
`

func statsHelperExists(db *sql.DB, statsHelper string) bool {
	var enabled bool

	err := db.QueryRow(QueryMarkerSQL + fmt.Sprintf(statsHelperSQL, statsHelper)).Scan(&enabled)
	if err != nil {
		return false
	}

	return enabled
}

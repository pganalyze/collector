package postgres

import (
	"database/sql"
	"fmt"
)

const connectedAsSuperUserSQL string = `SELECT current_setting('is_superuser') = 'on'`

const connectedAsRdsSuperUserSQL string = `
SELECT pg_has_role(oid, 'MEMBER') FROM pg_roles WHERE rolname = 'rds_superuser'
`

func connectedAsSuperUser(db *sql.DB, isAmazonRds bool) bool {
	var enabled bool

	if isAmazonRds {
		err := db.QueryRow(QueryMarkerSQL + connectedAsRdsSuperUserSQL).Scan(&enabled)
		if err != nil {
			return false
		}
		return enabled
	}

	err := db.QueryRow(QueryMarkerSQL + connectedAsSuperUserSQL).Scan(&enabled)
	if err != nil {
		return false
	}

	return enabled
}

const connectedAsMonitoringRoleSQL string = `
SELECT pg_has_role(oid, 'MEMBER') FROM pg_roles WHERE rolname = 'pg_monitor'
`

func connectedAsMonitoringRole(db *sql.DB) bool {
	var enabled bool

	err := db.QueryRow(QueryMarkerSQL + connectedAsMonitoringRoleSQL).Scan(&enabled)
	if err != nil {
		return false
	}

	return enabled
}

const statsHelperSQL string = `
SELECT 1 AS enabled
	FROM pg_catalog.pg_proc p
	JOIN pg_catalog.pg_namespace n ON (p.pronamespace = n.oid)
 WHERE n.nspname = 'pganalyze' AND p.proname = '%s'
`

func statsHelperExists(db *sql.DB, statsHelper string) bool {
	var enabled bool

	err := db.QueryRow(QueryMarkerSQL + fmt.Sprintf(statsHelperSQL, statsHelper)).Scan(&enabled)
	if err != nil {
		return false
	}

	return enabled
}

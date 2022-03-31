package postgres

import (
	"database/sql"
	"fmt"
)

const connectedAsSuperUserSQL string = `SELECT current_setting('is_superuser') = 'on'`

const connectedAsRdsSuperUserSQL string = `
SELECT pg_has_role(oid, 'MEMBER') FROM pg_roles WHERE rolname = 'rds_superuser'
`

const connectedAsAzurePostgresAdmin string = `
SELECT pg_has_role(oid, 'MEMBER') FROM pg_roles WHERE rolname = 'azure_pg_admin'
`

const connectedAsCloudSQLSuperuser string = `
SELECT pg_has_role(oid, 'MEMBER') FROM pg_roles WHERE rolname = 'cloudsqlsuperuser'
`

func connectedAsSuperUser(db *sql.DB, systemType string) bool {
	var enabled bool

	if systemType == "amazon_rds" {
		err := db.QueryRow(QueryMarkerSQL + connectedAsRdsSuperUserSQL).Scan(&enabled)
		if err != nil {
			return false
		}
		return enabled
	}

	if systemType == "azure_database" {
		err := db.QueryRow(QueryMarkerSQL + connectedAsAzurePostgresAdmin).Scan(&enabled)
		if err != nil {
			return false
		}
		return enabled
	}

	if systemType == "google_cloudsql" {
		err := db.QueryRow(QueryMarkerSQL + connectedAsCloudSQLSuperuser).Scan(&enabled)
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

func isCloudInternalDatabase(systemType string, databaseName string) bool {
	if systemType == "amazon_rds" {
		return databaseName == "rdsadmin"
	}
	if systemType == "azure_database" {
		return databaseName == "azure_maintenance"
	}
	if systemType == "google_cloudsql" {
		return databaseName == "cloudsqladmin"
	}
	return false
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

func StatsHelperExists(db *sql.DB, statsHelper string) bool {
	var enabled bool

	err := db.QueryRow(QueryMarkerSQL + fmt.Sprintf(statsHelperSQL, statsHelper)).Scan(&enabled)
	if err != nil {
		return false
	}

	return enabled
}

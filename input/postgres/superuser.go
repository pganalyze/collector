package postgres

import (
	"context"
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

func connectedAsSuperUser(ctx context.Context, db *sql.DB, systemType string) bool {
	var enabled bool

	if systemType == "amazon_rds" {
		err := db.QueryRowContext(ctx, QueryMarkerSQL+connectedAsRdsSuperUserSQL).Scan(&enabled)
		if err != nil {
			return false
		}
		return enabled
	}

	if systemType == "azure_database" {
		err := db.QueryRowContext(ctx, QueryMarkerSQL+connectedAsAzurePostgresAdmin).Scan(&enabled)
		if err != nil {
			return false
		}
		return enabled
	}

	if systemType == "google_cloudsql" {
		err := db.QueryRowContext(ctx, QueryMarkerSQL+connectedAsCloudSQLSuperuser).Scan(&enabled)
		if err != nil {
			return false
		}
		return enabled
	}

	err := db.QueryRowContext(ctx, QueryMarkerSQL+connectedAsSuperUserSQL).Scan(&enabled)
	if err != nil {
		return false
	}

	return enabled
}

const connectedAsMonitoringRoleSQL string = `
SELECT pg_has_role(oid, 'MEMBER') FROM pg_roles WHERE rolname = 'pg_monitor'
`

func connectedAsMonitoringRole(ctx context.Context, db *sql.DB) bool {
	var enabled bool

	err := db.QueryRowContext(ctx, QueryMarkerSQL+connectedAsMonitoringRoleSQL).Scan(&enabled)
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

func StatsHelperExists(ctx context.Context, db *sql.DB, statsHelper string) bool {
	var enabled bool

	err := db.QueryRowContext(ctx, QueryMarkerSQL+fmt.Sprintf(statsHelperSQL, statsHelper)).Scan(&enabled)
	if err != nil {
		return false
	}

	return enabled
}

const statsHelperReturnTypeSQL string = `
SELECT pg_catalog.format_type(prorettype, null)
FROM pg_catalog.pg_proc p
JOIN pg_catalog.pg_namespace n ON (p.pronamespace = n.oid)
WHERE n.nspname = 'pganalyze' AND p.proname = '%s'
`

func StatsHelperReturnType(ctx context.Context, db *sql.DB, statsHelper string) string {
	var source string

	err := db.QueryRowContext(ctx, QueryMarkerSQL+fmt.Sprintf(statsHelperReturnTypeSQL, statsHelper)).Scan(&source)
	if err != nil {
		return ""
	}

	return source
}

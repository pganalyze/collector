package postgres

import (
	"context"
	"database/sql"
	"fmt"
)

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

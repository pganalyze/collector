package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

const backendCountsSQL string = `
 SELECT datid,
				usesysid,
				COALESCE(state, 'unknown'),
				COALESCE(backend_type, 'unknown'), COALESCE(wait_event_type, '') = 'Lock' AS waiting_for_lock,
				pg_catalog.count(*)
	 FROM %s
	GROUP BY 1, 2, 3, 4, 5`

func GetBackendCounts(ctx context.Context, c *Collection, db *sql.DB) ([]state.PostgresBackendCount, error) {
	var sourceTable string

	if c.HelperExists("get_stat_activity", nil) {
		sourceTable = "pganalyze.get_stat_activity()"
	} else {
		sourceTable = "pg_catalog.pg_stat_activity"
	}

	rows, err := db.QueryContext(ctx, QueryMarkerSQL+fmt.Sprintf(backendCountsSQL, sourceTable))
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var backendCounts []state.PostgresBackendCount

	for rows.Next() {
		var row state.PostgresBackendCount

		err := rows.Scan(&row.DatabaseOid, &row.RoleOid, &row.State, &row.BackendType,
			&row.WaitingForLock, &row.Count)
		if err != nil {
			return nil, err
		}

		// backend_type can contain invalid UTF-8 on some platforms (e.g. Azure
		// management backends, Supabase's Supavisor), which would fail snapshot
		// marshaling; scrub it uniformly to the replacement character.
		row.BackendType = strings.ToValidUTF8(row.BackendType, util.InvalidUTF8Replacement)

		backendCounts = append(backendCounts, row)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return backendCounts, nil
}

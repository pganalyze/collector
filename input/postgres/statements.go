package postgres

import (
	"database/sql"
	"fmt"
	"hash/fnv"
	"strings"

	"github.com/guregu/null"
	"github.com/lib/pq"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

const statementSQLDefaultOptionalFields = "NULL, NULL, NULL, NULL, NULL"
const statementSQLpg94OptionalFields = "queryid, NULL, NULL, NULL, NULL"
const statementSQLpg95OptionalFields = "queryid, min_time, max_time, mean_time, stddev_time"

const statementSQL string = `
SELECT dbid, userid, query, calls, total_time, rows, shared_blks_hit, shared_blks_read,
			 shared_blks_dirtied, shared_blks_written, local_blks_hit, local_blks_read,
			 local_blks_dirtied, local_blks_written, temp_blks_read, temp_blks_written,
			 blk_read_time, blk_write_time, %s
	FROM %s
 WHERE query IS NULL OR (query !~* '^%s' AND query <> '<insufficient privilege>'
			 AND query NOT LIKE 'DEALLOCATE %%')`

const statementStatsHelperSQL string = `
SELECT 1 AS enabled
	FROM pg_proc
	JOIN pg_namespace ON (pronamespace = pg_namespace.oid)
 WHERE nspname = 'pganalyze' AND proname = 'get_stat_statements'
       %s
`

func statementStatsHelperExists(db *sql.DB, showtext bool) bool {
	var enabled bool
	var additionalWhere string

	if !showtext {
		additionalWhere = "AND pronargs = 1"
	}

	err := db.QueryRow(QueryMarkerSQL + fmt.Sprintf(statementStatsHelperSQL, additionalWhere)).Scan(&enabled)
	if err != nil {
		return false
	}

	return enabled
}

func GetStatements(logger *util.Logger, db *sql.DB, postgresVersion state.PostgresVersion, showtext bool) (state.PostgresStatementMap, state.PostgresStatementStatsMap, error) {
	var err error
	var optionalFields string
	var sourceTable string

	if postgresVersion.Numeric >= state.PostgresVersion95 {
		optionalFields = statementSQLpg95OptionalFields
	} else if postgresVersion.Numeric >= state.PostgresVersion94 {
		optionalFields = statementSQLpg94OptionalFields
	} else {
		optionalFields = statementSQLDefaultOptionalFields
	}

	usingStatsHelper := false

	if statementStatsHelperExists(db, showtext) {
		usingStatsHelper = true
		if !showtext {
			logger.PrintVerbose("Found pganalyze.get_stat_statements(false) stats helper")
			sourceTable = "pganalyze.get_stat_statements(false)"
		} else {
			logger.PrintVerbose("Found pganalyze.get_stat_statements() stats helper")
			sourceTable = "pganalyze.get_stat_statements()"
		}
	} else {
		if !connectedAsSuperUser(db) {
			logger.PrintInfo("Warning: You are not connecting as superuser. Please setup" +
				" the monitoring helper functions (https://github.com/pganalyze/collector#setting-up-a-restricted-monitoring-user)" +
				" or connect as superuser, to get query statistics for all roles.")
		}
		if !showtext {
			sourceTable = "public.pg_stat_statements(false)"
		} else {
			sourceTable = "public.pg_stat_statements"
		}
	}

	queryMarkerRegex := strings.Trim(QueryMarkerSQL, " ")
	queryMarkerRegex = strings.Replace(queryMarkerRegex, "*", "\\*", -1)
	queryMarkerRegex = strings.Replace(queryMarkerRegex, "/", "\\/", -1)

	sql := QueryMarkerSQL + fmt.Sprintf(statementSQL, optionalFields, sourceTable, queryMarkerRegex)

	stmt, err := db.Prepare(sql)
	if err != nil {
		errCode := err.(*pq.Error).Code
		if !usingStatsHelper && (errCode == "42P01" || errCode == "42883") { // undefined_table / undefined_function
			logger.PrintInfo("pg_stat_statements does not exist, trying to create extension...")

			_, err = db.Exec(QueryMarkerSQL + "CREATE EXTENSION IF NOT EXISTS pg_stat_statements")
			if err != nil {
				return nil, nil, err
			}

			stmt, err = db.Prepare(sql)
			if err != nil {
				return nil, nil, err
			}
		} else {
			return nil, nil, err
		}
	}

	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	statements := make(state.PostgresStatementMap)
	statementStats := make(state.PostgresStatementStatsMap)

	for rows.Next() {
		var key state.PostgresStatementKey
		var queryID null.Int
		var normalizedQuery null.String
		var stats state.PostgresStatementStats

		err = rows.Scan(&key.DatabaseOid, &key.UserOid, &normalizedQuery, &stats.Calls, &stats.TotalTime, &stats.Rows,
			&stats.SharedBlksHit, &stats.SharedBlksRead, &stats.SharedBlksDirtied, &stats.SharedBlksWritten,
			&stats.LocalBlksHit, &stats.LocalBlksRead, &stats.LocalBlksDirtied, &stats.LocalBlksWritten,
			&stats.TempBlksRead, &stats.TempBlksWritten, &stats.BlkReadTime, &stats.BlkWriteTime,
			&queryID, &stats.MinTime, &stats.MaxTime, &stats.MeanTime, &stats.StddevTime)
		if err != nil {
			return nil, nil, err
		}

		if queryID.Valid {
			key.QueryID = queryID.Int64
		} else if normalizedQuery.Valid {
			// Note: This is a heuristic for old Postgres versions and will not work for duplicate queries (e.g. when tables are dropped and recreated)
			h := fnv.New64a()
			h.Write([]byte(normalizedQuery.String))
			key.QueryID = int64(h.Sum64())
		} else {
			// We can't process this entry, most likely a permission problem with reading the query ID
			continue
		}

		if showtext {
			statements[key] = state.PostgresStatement{NormalizedQuery: normalizedQuery.String}
		}
		statementStats[key] = stats
	}

	return statements, statementStats, nil
}

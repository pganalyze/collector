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
 WHERE query !~* '^%s' AND query <> '<insufficient privilege>'
			 AND query NOT LIKE 'DEALLOCATE %%'`

func GetStatements(logger *util.Logger, db *sql.DB, postgresVersion state.PostgresVersion) (state.PostgresStatementMap, state.PostgresStatementStatsMap, error) {
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

	if statsHelperExists(db, "get_stat_statements") {
		logger.PrintVerbose("Found pganalyze.get_stat_statements() stats helper")
		sourceTable = "pganalyze.get_stat_statements()"
	} else {
		if !connectedAsSuperUser(db) {
			logger.PrintInfo("Warning: You are not connecting as superuser. Please setup" +
				" the monitoring helper functions (https://github.com/pganalyze/collector#setting-up-a-restricted-monitoring-user)" +
				" or connect as superuser, to get query statistics for all roles.")
		}
		sourceTable = "pg_stat_statements"
	}

	queryMarkerRegex := strings.Trim(QueryMarkerSQL, " ")
	queryMarkerRegex = strings.Replace(queryMarkerRegex, "*", "\\*", -1)
	queryMarkerRegex = strings.Replace(queryMarkerRegex, "/", "\\/", -1)

	sql := QueryMarkerSQL + fmt.Sprintf(statementSQL, optionalFields, sourceTable, queryMarkerRegex)

	stmt, err := db.Prepare(sql)
	if err != nil {
		if sourceTable == "pg_stat_statements" && err.(*pq.Error).Code == "42P01" { // undefined_table
			logger.PrintInfo("pg_stat_statements relation does not exist, trying to create extension...")

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
		var statement state.PostgresStatement
		var stats state.PostgresStatementStats

		err = rows.Scan(&key.DatabaseOid, &key.UserOid, &statement.NormalizedQuery, &stats.Calls, &stats.TotalTime, &stats.Rows,
			&stats.SharedBlksHit, &stats.SharedBlksRead, &stats.SharedBlksDirtied, &stats.SharedBlksWritten,
			&stats.LocalBlksHit, &stats.LocalBlksRead, &stats.LocalBlksDirtied, &stats.LocalBlksWritten,
			&stats.TempBlksRead, &stats.TempBlksWritten, &stats.BlkReadTime, &stats.BlkWriteTime,
			&queryID, &stats.MinTime, &stats.MaxTime, &stats.MeanTime, &stats.StddevTime)
		if err != nil {
			return nil, nil, err
		}

		if queryID.Valid {
			key.QueryID = queryID.Int64
		} else {
			// Note: This is a heuristic for old Postgres versions and will not work for duplicate queries (e.g. when tables are dropped and recreated)
			h := fnv.New64a()
			h.Write([]byte(statement.NormalizedQuery))
			key.QueryID = int64(h.Sum64())
		}

		statements[key] = statement
		statementStats[key] = stats
	}

	return statements, statementStats, nil
}

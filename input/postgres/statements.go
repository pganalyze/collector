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
	FROM %s`

const statementStatsHelperSQL string = `
SELECT 1 AS enabled
	FROM pg_catalog.pg_proc p
	JOIN pg_catalog.pg_namespace n ON (p.pronamespace = n.oid)
 WHERE n.nspname = 'pganalyze' AND p.proname = 'get_stat_statements'
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

func collectorStatement(query string) bool {
	return strings.HasPrefix(query, QueryMarkerSQL)
}

func insufficientPrivilege(query string) bool {
	return query == "<insufficient privilege>"
}

func ResetStatements(logger *util.Logger, db *sql.DB, systemType string) error {
	var method string
	if statsHelperExists(db, "reset_stat_statements") {
		logger.PrintVerbose("Found pganalyze.reset_stat_statements() stats helper")
		method = "pganalyze.reset_stat_statements()"
	} else {
		if !connectedAsSuperUser(db, systemType) && !connectedAsMonitoringRole(db) {
			logger.PrintInfo("Warning: You are not connecting as superuser. Please setup" +
				" contact support to get advice on setting up stat statements reset")
		}
		method = "pg_stat_statements_reset()"
	}
	_, err := db.Exec(QueryMarkerSQL + "SELECT " + method)
	if err != nil {
		return err
	}
	return nil
}

func GetStatements(logger *util.Logger, db *sql.DB, globalCollectionOpts state.CollectionOpts, postgresVersion state.PostgresVersion, showtext bool, systemType string) (state.PostgresStatementMap, state.PostgresStatementTextMap, state.PostgresStatementStatsMap, error) {
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
		if systemType != "heroku" && !connectedAsSuperUser(db, systemType) && !connectedAsMonitoringRole(db) && globalCollectionOpts.TestRun {
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

	sql := QueryMarkerSQL + fmt.Sprintf(statementSQL, optionalFields, sourceTable)

	stmt, err := db.Prepare(sql)
	if err != nil {
		errCode := err.(*pq.Error).Code
		if !usingStatsHelper && (errCode == "42P01" || errCode == "42883") { // undefined_table / undefined_function
			logger.PrintInfo("pg_stat_statements does not exist, trying to create extension...")

			_, err = db.Exec(QueryMarkerSQL + "CREATE EXTENSION IF NOT EXISTS pg_stat_statements SCHEMA public")
			if err != nil {
				return nil, nil, nil, err
			}

			stmt, err = db.Prepare(sql)
			if err != nil {
				return nil, nil, nil, err
			}
		} else {
			return nil, nil, nil, err
		}
	}

	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		errCode := err.(*pq.Error).Code
		if errCode == "55000" { // object_not_in_prerequisite_state
			if globalCollectionOpts.TestRun {
				logger.PrintWarning("Could not collect query statistics: pg_stat_statements must be added to shared_preload_libraries")
			}
			// We intentionally don't return an error here, as we want the rest of
			// processing to continue without requiring a reboot
			return nil, nil, nil, nil
		}
		return nil, nil, nil, err
	}
	defer rows.Close()

	statements := make(state.PostgresStatementMap)
	statementTexts := make(state.PostgresStatementTextMap)
	statementStats := make(state.PostgresStatementStatsMap)

	for rows.Next() {
		var key state.PostgresStatementKey
		var queryID null.Int
		var receivedQuery null.String
		var stats state.PostgresStatementStats

		err = rows.Scan(&key.DatabaseOid, &key.UserOid, &receivedQuery, &stats.Calls, &stats.TotalTime, &stats.Rows,
			&stats.SharedBlksHit, &stats.SharedBlksRead, &stats.SharedBlksDirtied, &stats.SharedBlksWritten,
			&stats.LocalBlksHit, &stats.LocalBlksRead, &stats.LocalBlksDirtied, &stats.LocalBlksWritten,
			&stats.TempBlksRead, &stats.TempBlksWritten, &stats.BlkReadTime, &stats.BlkWriteTime,
			&queryID, &stats.MinTime, &stats.MaxTime, &stats.MeanTime, &stats.StddevTime)
		if err != nil {
			return nil, nil, nil, err
		}

		if queryID.Valid {
			key.QueryID = queryID.Int64
		} else if receivedQuery.Valid {
			// Note: This is a heuristic for old Postgres versions and will not work for duplicate queries (e.g. when tables are dropped and recreated)
			h := fnv.New64a()
			h.Write([]byte(receivedQuery.String))
			key.QueryID = int64(h.Sum64())
		} else {
			// We can't process this entry, most likely a permission problem with reading the query ID
			continue
		}

		if showtext {
			fp := util.FingerprintQuery(receivedQuery.String)
			stmt := state.PostgresStatement{Fingerprint: fp}
			if insufficientPrivilege(receivedQuery.String) {
				stmt.InsufficientPrivilege = true
			} else if collectorStatement(receivedQuery.String) {
				stmt.Collector = true
				stmt.Fingerprint = util.FingerprintQuery("<pganalyze-collector>")
			} else {
				_, ok := statementTexts[fp]
				if !ok {
					statementTexts[fp] = receivedQuery.String
				}
			}

			statements[key] = stmt
		}
		statementStats[key] = stats
	}
	err = rows.Err()
	if err != nil {
		return nil, nil, nil, err
	}

	for fp, text := range statementTexts {
		statementTexts[fp] = util.NormalizeQuery(text)
	}

	return statements, statementTexts, statementStats, nil
}

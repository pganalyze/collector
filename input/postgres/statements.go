package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/guregu/null"
	"github.com/lib/pq"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

// pg_stat_statements 1.3+ (Postgres 9.5+)
const statementSQLOptionalFieldsMinorVersion3 = "queryid, min_time, max_time, mean_time, stddev_time"
const statementSQLTotalTimeFieldDefault = "total_time"

// pg_stat_statements 1.8+ (Postgres 13+)
const statementSQLOptionalFieldsMinorVersion8 = "queryid, min_exec_time, max_exec_time, mean_exec_time, stddev_exec_time"
const statementSQLTotalTimeFieldMinorVersion8 = "total_exec_time"

const statementSQL string = `
SELECT dbid, userid, query, calls, %s, rows, shared_blks_hit, shared_blks_read,
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

const statementExtensionVersionSQL string = `
SELECT nspname,
       split_part(extversion, '.', 2)
  FROM pg_extension pge
 INNER JOIN pg_namespace pgn ON pge.extnamespace = pgn.oid
 WHERE pge.extname = 'pg_stat_statements'
`

func statementStatsHelperExists(ctx context.Context, db *sql.DB, showtext bool) bool {
	var enabled bool
	var additionalWhere string

	if !showtext {
		additionalWhere = "AND pronargs = 1"
	}

	err := db.QueryRowContext(ctx, QueryMarkerSQL+fmt.Sprintf(statementStatsHelperSQL, additionalWhere)).Scan(&enabled)
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

func ResetStatements(ctx context.Context, logger *util.Logger, db *sql.DB, systemType string) error {
	var method string
	if StatsHelperExists(ctx, db, "reset_stat_statements") {
		logger.PrintVerbose("Found pganalyze.reset_stat_statements() stats helper")
		method = "pganalyze.reset_stat_statements()"
	} else {
		if !connectedAsSuperUser(ctx, db, systemType) && !connectedAsMonitoringRole(ctx, db) {
			logger.PrintInfo("Warning: You are not connecting as superuser. Please" +
				" contact support to get advice on setting up stat statements reset")
		}
		method = "pg_stat_statements_reset()"
	}
	_, err := db.ExecContext(ctx, QueryMarkerSQL+"SELECT "+method)
	if err != nil {
		return err
	}
	return nil
}

func GetStatements(ctx context.Context, server *state.Server, logger *util.Logger, db *sql.DB, globalCollectionOpts state.CollectionOpts, postgresVersion state.PostgresVersion, showtext bool, systemType string) (state.PostgresStatementMap, state.PostgresStatementTextMap, state.PostgresStatementStatsMap, error) {
	var err error
	var totalTimeField string
	var optionalFields string
	var sourceTable string
	var extSchema string
	var extMinorVersion int16
	var foundExtMinorVersion int16

	if postgresVersion.Numeric >= state.PostgresVersion13 {
		extMinorVersion = 8
	} else {
		extMinorVersion = 3
	}

	err = db.QueryRowContext(ctx, QueryMarkerSQL+statementExtensionVersionSQL).Scan(&extSchema, &foundExtMinorVersion)
	if err != nil && err != sql.ErrNoRows {
		return nil, nil, nil, err
	}

	if err == sql.ErrNoRows {
		logger.PrintInfo("pg_stat_statements does not exist, trying to create extension...")
		_, err = db.ExecContext(ctx, QueryMarkerSQL+"CREATE EXTENSION IF NOT EXISTS pg_stat_statements SCHEMA public")
		if err != nil {
			return nil, nil, nil, err
		}
		extSchema = "public"
		foundExtMinorVersion = extMinorVersion
	}

	if foundExtMinorVersion >= 8 {
		optionalFields = statementSQLOptionalFieldsMinorVersion8
	} else if foundExtMinorVersion >= 3 {
		optionalFields = statementSQLOptionalFieldsMinorVersion3
	} else {
		return nil, nil, nil, fmt.Errorf("pg_stat_statements extension not supported (1.%d installed, 1.3+ supported). To update run `ALTER EXTENSION pg_stat_statements UPDATE`", foundExtMinorVersion)
	}

	if foundExtMinorVersion >= 8 {
		totalTimeField = statementSQLTotalTimeFieldMinorVersion8
	} else {
		totalTimeField = statementSQLTotalTimeFieldDefault
	}

	if globalCollectionOpts.TestRun && foundExtMinorVersion < extMinorVersion {
		logger.PrintInfo("pg_stat_statements extension outdated (1.%d installed, 1.%d available). To update run `ALTER EXTENSION pg_stat_statements UPDATE`", foundExtMinorVersion, extMinorVersion)
	}

	usingStatsHelper := false
	if statementStatsHelperExists(ctx, db, showtext) {
		usingStatsHelper = true
		if !showtext {
			logger.PrintVerbose("Found pganalyze.get_stat_statements(false) stats helper")
			sourceTable = "pganalyze.get_stat_statements(false)"
		} else {
			logger.PrintVerbose("Found pganalyze.get_stat_statements() stats helper")
			sourceTable = "pganalyze.get_stat_statements()"
		}
	} else {
		if systemType != "heroku" && !connectedAsSuperUser(ctx, db, systemType) && !connectedAsMonitoringRole(ctx, db) && globalCollectionOpts.TestRun {
			logger.PrintInfo("Warning: You are not connecting as superuser. Please setup" +
				" the monitoring helper functions (https://github.com/pganalyze/collector#setting-up-a-restricted-monitoring-user)" +
				" or connect as superuser, to get query statistics for all roles.")
		}
		if !showtext {
			sourceTable = extSchema + ".pg_stat_statements(false)"
		} else {
			sourceTable = extSchema + ".pg_stat_statements"
		}
	}

	querySql := QueryMarkerSQL + fmt.Sprintf(statementSQL, totalTimeField, optionalFields, sourceTable)

	stmt, err := db.PrepareContext(ctx, querySql)
	if err != nil {
		var e *pq.Error
		if !usingStatsHelper && errors.As(err, &e) {
			// If we get ErrNoRows, the extension does not exist, which is one of the expected paths
			if err != nil && err != sql.ErrNoRows {
				return nil, nil, nil, err
			}
		} else {
			return nil, nil, nil, err
		}
	}

	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		var e *pq.Error
		if errors.As(err, &e) && e.Code == "55000" { // object_not_in_prerequisite_state
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

	statementTexts := make(map[state.PostgresStatementKey]string)
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
		} else {
			// We can't process this entry, most likely a permission problem with reading the query ID
			continue
		}

		if showtext {
			statementTexts[key] = receivedQuery.String
		}
		if ignoreIOTiming(postgresVersion, receivedQuery) {
			stats.BlkReadTime = 0
			stats.BlkWriteTime = 0
		}
		statementStats[key] = stats
	}

	if err = rows.Err(); err != nil {
		return nil, nil, nil, err
	}

	statements := make(state.PostgresStatementMap)
	statementTextsByFp := make(state.PostgresStatementTextMap)
	if showtext {
		collectorQueryFingerprint := util.FingerprintText(util.QueryTextCollector)
		insufficientPrivsQueryFingerprint := util.FingerprintText(util.QueryTextInsufficientPrivs)

		for key, text := range statementTexts {
			select {
			// Since fingerprinting/normalizing can take time, explicitly check for cancellations
			case <-ctx.Done():
				return nil, nil, nil, ctx.Err()
			default:
				if insufficientPrivilege(text) {
					statements[key] = state.PostgresStatement{
						InsufficientPrivilege: true,
						Fingerprint:           insufficientPrivsQueryFingerprint,
					}
				} else if collectorStatement(text) {
					statements[key] = state.PostgresStatement{
						Collector:   true,
						Fingerprint: collectorQueryFingerprint,
					}
				} else {
					fp := util.FingerprintQuery(text, server.Config.FilterQueryText, -1)
					statements[key] = state.PostgresStatement{Fingerprint: fp}
					_, ok := statementTextsByFp[fp]
					if !ok {
						statementTextsByFp[fp] = util.NormalizeQuery(text, server.Config.FilterQueryText, -1)
					}
				}
			}
		}
	}

	return statements, statementTextsByFp, statementStats, nil
}

func ignoreIOTiming(postgresVersion state.PostgresVersion, receivedQuery null.String) bool {
	// Currently, Aurora gives wildly incorrect blk_read_time and blk_write_time values
	// for utility statements; ignore I/O timing in this situation.
	if !postgresVersion.IsAwsAurora || !receivedQuery.Valid {
		return false
	}

	isUtil, err := util.IsUtilityStmt(receivedQuery.String)
	if err != nil {
		return false
	}

	for _, isOneUtil := range isUtil {
		if isOneUtil {
			return true
		}
	}

	return false
}

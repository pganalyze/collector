package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/guregu/null"
	"github.com/lib/pq"
	"github.com/pganalyze/collector/selftest"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

// pg_stat_statements 1.3+ (Postgres 9.5+)
const statementSQLTotalTimeFieldDefault = "total_time"
const statementSQLIoTimeFieldsDefault = "blk_read_time, blk_write_time"
const statementSQLOptionalFieldsMinorVersion3 = "min_time, max_time, mean_time, stddev_time, NULL"

// pg_stat_statements 1.8+ (Postgres 13+)
const statementSQLTotalTimeFieldMinorVersion8 = "total_exec_time"
const statementSQLOptionalFieldsMinorVersion8 = "min_exec_time, max_exec_time, mean_exec_time, stddev_exec_time, NULL"

// pg_stat_statements 1.9+ (Postgres 14+)
const statementSQLOptionalFieldsMinorVersion9 = "min_exec_time, max_exec_time, mean_exec_time, stddev_exec_time, toplevel"

// pg_stat_statements 1.11+ (Postgres 17+)
const statementSQLIoTimeFieldsMinorVersion11 = "shared_blk_read_time + local_blk_read_time + temp_blk_read_time, shared_blk_write_time + local_blk_write_time + temp_blk_write_time"

const statementSQL string = `
SELECT dbid, userid, query, calls, %s, rows, shared_blks_hit, shared_blks_read,
			 shared_blks_dirtied, shared_blks_written, local_blks_hit, local_blks_read,
			 local_blks_dirtied, local_blks_written, temp_blks_read, temp_blks_written,
			 %s,
			 queryid,
			 %s
	FROM %s`

const statementExtensionVersionSQL string = `
SELECT nspname,
       split_part(extversion, '.', 2)
  FROM pg_extension pge
 INNER JOIN pg_namespace pgn ON pge.extnamespace = pgn.oid
 WHERE pge.extname = 'pg_stat_statements'
`

func collectorStatement(query string) bool {
	return strings.HasPrefix(query, QueryMarkerSQL)
}

func insufficientPrivilege(query string) bool {
	return query == "<insufficient privilege>"
}

func ResetStatements(ctx context.Context, c *Collection, db *sql.DB) error {
	var method string
	if c.HelperExists("reset_stat_statements", nil) {
		c.Logger.PrintVerbose("Found pganalyze.reset_stat_statements() stats helper")
		method = "pganalyze.reset_stat_statements()"
	} else {
		if !c.ConnectedAsSuperUser && !c.ConnectedAsMonitoringRole {
			c.Logger.PrintInfo("Warning: You are not connecting as superuser. Please" +
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

func GetStatements(ctx context.Context, c *Collection, db *sql.DB, showtext bool) (state.PostgresStatementMap, state.PostgresStatementTextMap, state.PostgresStatementStatsMap, error) {
	var err error
	var totalTimeField string
	var ioTimeFields string
	var optionalFields string
	var sourceTable string
	var extSchema string
	var extMinorVersion int16
	var foundExtMinorVersion int16
	var tmpFile *os.File

	if c.PostgresVersion.Numeric >= state.PostgresVersion17 {
		extMinorVersion = 11
	} else if c.PostgresVersion.Numeric >= state.PostgresVersion14 {
		extMinorVersion = 9
	} else if c.PostgresVersion.Numeric >= state.PostgresVersion13 {
		extMinorVersion = 8
	} else {
		extMinorVersion = 3
	}

	err = db.QueryRowContext(ctx, QueryMarkerSQL+statementExtensionVersionSQL).Scan(&extSchema, &foundExtMinorVersion)
	if err != nil && err != sql.ErrNoRows {
		return nil, nil, nil, err
	}

	if err == sql.ErrNoRows {
		c.Logger.PrintInfo("pg_stat_statements does not exist, trying to create extension...")
		_, err = db.ExecContext(ctx, QueryMarkerSQL+"CREATE EXTENSION IF NOT EXISTS pg_stat_statements SCHEMA public")
		if err != nil {
			c.SelfTest.MarkCollectionAspectError(state.CollectionAspectPgStatStatements, "extension does not exist in database %s and could not be created: %s", c.Config.DbName, err)
			c.Logger.PrintInfo("HINT - if you expect the extension to already be installed, please review the pganalyze documentation: https://pganalyze.com/docs/install/troubleshooting/pg_stat_statements")
			return nil, nil, nil, err
		}
		extSchema = "public"
		foundExtMinorVersion = extMinorVersion
	}

	if c.PostgresVersion.Numeric >= state.PostgresVersion14 && foundExtMinorVersion < 9 {
		c.SelfTest.MarkCollectionAspectError(state.CollectionAspectPgStatStatements, "extension version (1.%d) is too old, 1.9 or newer is required.", foundExtMinorVersion)
		c.SelfTest.HintCollectionAspect(state.CollectionAspectPgStatStatements, "Update the extension by running `ALTER EXTENSION pg_stat_statements UPDATE`.")
	}

	if foundExtMinorVersion >= 8 {
		totalTimeField = statementSQLTotalTimeFieldMinorVersion8
	} else {
		totalTimeField = statementSQLTotalTimeFieldDefault
	}

	if foundExtMinorVersion >= 11 {
		ioTimeFields = statementSQLIoTimeFieldsMinorVersion11
	} else {
		ioTimeFields = statementSQLIoTimeFieldsDefault
	}

	if foundExtMinorVersion >= 9 {
		optionalFields = statementSQLOptionalFieldsMinorVersion9
	} else if foundExtMinorVersion >= 8 {
		optionalFields = statementSQLOptionalFieldsMinorVersion8
	} else if foundExtMinorVersion >= 3 {
		optionalFields = statementSQLOptionalFieldsMinorVersion3
	} else {
		c.SelfTest.MarkCollectionAspectError(state.CollectionAspectPgStatStatements, "extension version not supported (1.%d installed, 1.3+ supported)", foundExtMinorVersion)
		return nil, nil, nil, fmt.Errorf("pg_stat_statements extension not supported (1.%d installed, 1.3+ supported). To update run `ALTER EXTENSION pg_stat_statements UPDATE`", foundExtMinorVersion)
	}

	if c.GlobalOpts.TestRun && foundExtMinorVersion < extMinorVersion {
		pgssMsg := fmt.Sprintf("extension outdated (1.%d installed, 1.%d available)", foundExtMinorVersion, extMinorVersion)
		c.Logger.PrintInfo("pg_stat_statements %s. To update run `ALTER EXTENSION pg_stat_statements UPDATE`", pgssMsg)
		if extMinorVersion >= 9 {
			// Using the older version pgss with Postgres 14+ can cause the incorrect query stats
			// when track = all is used + there are toplevel queries and nested queries
			// https://github.com/pganalyze/collector/pull/472#discussion_r1399976152
			c.Logger.PrintError("Outdated pg_stat_statements may cause incorrect query statistics")
			pgssMsg += "; outdated pg_stat_statements may cause incorrect query statistics"
		}
		c.SelfTest.MarkCollectionAspectWarning(state.CollectionAspectPgStatStatements, pgssMsg)
		c.SelfTest.HintCollectionAspect(state.CollectionAspectPgStatStatements, "To update run `ALTER EXTENSION pg_stat_statements UPDATE`")
	}

	usingStatsHelper := false
	if c.HelperExists("get_stat_statements", []string{"boolean"}) || (showtext && c.HelperExists("get_stat_statements", nil)) {
		usingStatsHelper = true
		if !showtext {
			c.Logger.PrintVerbose("Found pganalyze.get_stat_statements(false) stats helper")
			sourceTable = "pganalyze.get_stat_statements(false)"
		} else {
			c.Logger.PrintVerbose("Found pganalyze.get_stat_statements() stats helper")
			sourceTable = "pganalyze.get_stat_statements()"
		}
	} else {
		if c.Config.SystemType != "heroku" && !c.ConnectedAsSuperUser && !c.ConnectedAsMonitoringRole && c.GlobalOpts.TestRun {
			c.SelfTest.MarkCollectionAspectWarning(state.CollectionAspectPgStatStatements, "monitoring user may have insufficient permissions to capture all queries")
			c.SelfTest.HintCollectionAspect(state.CollectionAspectPgStatStatements, "Please set up"+
				" the monitoring helper functions (%s)"+
				" or connect as superuser to get query statistics for all roles.", selftest.URLPrinter.Sprint("https://pganalyze.com/docs/install/aiven/03_create_pg_stat_statements_helpers"))
			c.Logger.PrintInfo("Warning: You are not connecting as superuser. Please setup" +
				" the monitoring helper functions (https://pganalyze.com/docs/install/aiven/03_create_pg_stat_statements_helpers)" +
				" or connect as superuser, to get query statistics for all roles.")
		}
		if !showtext {
			sourceTable = extSchema + ".pg_stat_statements(false)"
		} else {
			sourceTable = extSchema + ".pg_stat_statements"
		}
	}

	querySql := QueryMarkerSQL + fmt.Sprintf(statementSQL, totalTimeField, ioTimeFields, optionalFields, sourceTable)

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
		return nil, nil, nil, err
	}
	defer rows.Close()

	statements := make(state.PostgresStatementMap)
	statementTextsByFp := make(state.PostgresStatementTextMap)
	statementStats := make(state.PostgresStatementStatsMap)

	queryKeys := make([]state.PostgresStatementKey, 0)
	queryStats := make([]state.PostgresStatementStats, 0)
	queryLengths := make([]int, 0)
	if showtext {
		tmpFile, err = os.CreateTemp("", "")
		if err != nil {
			return nil, nil, nil, err
		}
		defer tmpFile.Close()
		defer os.Remove(tmpFile.Name())
	}

	for rows.Next() {
		var key state.PostgresStatementKey
		var queryID null.Int
		var receivedQuery null.String
		var stats state.PostgresStatementStats

		err = rows.Scan(&key.DatabaseOid, &key.UserOid, &receivedQuery, &stats.Calls, &stats.TotalTime, &stats.Rows,
			&stats.SharedBlksHit, &stats.SharedBlksRead, &stats.SharedBlksDirtied, &stats.SharedBlksWritten,
			&stats.LocalBlksHit, &stats.LocalBlksRead, &stats.LocalBlksDirtied, &stats.LocalBlksWritten,
			&stats.TempBlksRead, &stats.TempBlksWritten, &stats.BlkReadTime, &stats.BlkWriteTime,
			&queryID, &stats.MinTime, &stats.MaxTime, &stats.MeanTime, &stats.StddevTime, &key.TopLevel)
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
			queryKeys = append(queryKeys, key)
			queryStats = append(queryStats, stats)
			queryLengths = append(queryLengths, len(receivedQuery.String))
			tmpFile.WriteString(receivedQuery.String)
		} else {
			statementStats[key] = stats
		}
	}

	if err = rows.Err(); err != nil {
		return nil, nil, nil, err
	}

	if showtext {
		tmpFile.Seek(0, io.SeekStart)
		for idx, length := range queryLengths {
			bytes := make([]byte, length)
			_, err = io.ReadFull(tmpFile, bytes)
			if err != nil {
				return nil, nil, nil, err
			}
			query := string(bytes)
			key := queryKeys[idx]
			select {
			// Since normalizing can take time, explicitly check for cancellations
			case <-ctx.Done():
				return nil, nil, nil, ctx.Err()
			default:
				fingerprintAndNormalize(c, key, query, statements, statementTextsByFp)
			}
			stats := queryStats[idx]
			if ignoreIOTiming(c.PostgresVersion, query) {
				stats.BlkReadTime = 0
				stats.BlkWriteTime = 0
			}
			statementStats[key] = stats
		}
	}

	c.SelfTest.MarkCollectionAspectOk(state.CollectionAspectPgStatStatements)

	return statements, statementTextsByFp, statementStats, nil
}

func ignoreIOTiming(postgresVersion state.PostgresVersion, receivedQuery string) bool {
	// Currently, Aurora gives wildly incorrect blk_read_time and blk_write_time values
	// for utility statements; ignore I/O timing in this situation.
	if !postgresVersion.IsAwsAurora || receivedQuery == "" {
		return false
	}

	isUtil, err := util.IsUtilityStmt(receivedQuery)
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

var collectorQueryFingerprint = util.FingerprintText(util.QueryTextCollector)
var insufficientPrivsQueryFingerprint = util.FingerprintText(util.QueryTextInsufficientPrivs)

func fingerprintAndNormalize(c *Collection, key state.PostgresStatementKey, text string, statements state.PostgresStatementMap, statementTextsByFp state.PostgresStatementTextMap) {
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
		fp := util.FingerprintQuery(text, c.Config.FilterQueryText, -1)
		statements[key] = state.PostgresStatement{Fingerprint: fp}
		_, ok := statementTextsByFp[fp]
		if !ok {
			statementTextsByFp[fp] = util.NormalizeQuery(text, c.Config.FilterQueryText, -1)
		}
	}
}

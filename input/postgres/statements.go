package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/guregu/null"
	"github.com/pganalyze/collector/selftest"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	pg_query "github.com/pganalyze/pg_query_go/v6"
)

// pg_stat_statements 1.3+ (Postgres 9.5+)
const statementSQLTopLevelFieldDefault = "NULL"
const statementSQLTotalTimeFieldDefault = "total_time"
const statementSQLIoTimeFieldsDefault = "blk_read_time, blk_write_time"
const statementSQLOptionalFieldsDefault = "min_time, max_time, mean_time, stddev_time"

// pg_stat_statements 1.8+ (Postgres 13+)
const statementSQLTotalTimeFieldMinorVersion8 = "total_exec_time"
const statementSQLOptionalFieldsMinorVersion8 = "min_exec_time, max_exec_time, mean_exec_time, stddev_exec_time"

// pg_stat_statements 1.9+ (Postgres 14+)
const statementSQLTopLevelFieldMinorVersion9 = "toplevel"

// pg_stat_statements 1.11+ (Postgres 17+)
const statementSQLIoTimeFieldsMinorVersion11 = "shared_blk_read_time + local_blk_read_time + temp_blk_read_time, shared_blk_write_time + local_blk_write_time + temp_blk_write_time"

const statementStatsSQL string = `
SELECT dbid, userid, queryid, %s, calls, %s, rows, shared_blks_hit, shared_blks_read,
			 shared_blks_dirtied, shared_blks_written, local_blks_hit, local_blks_read,
			 local_blks_dirtied, local_blks_written, temp_blks_read, temp_blks_written,
			 %s,
			 %s
	FROM %s`

const statementTextSQL string = `
SELECT dbid, userid, queryid, %s, query
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

func GetStatementStats(ctx context.Context, c *Collection, db *sql.DB) (state.PostgresStatementStatsMap, error) {
	sourceTable, foundExtMinorVersion, err := getStatementSource(ctx, c, db, false)
	if err != nil {
		return nil, err
	}

	topLevelField := statementSQLTopLevelFieldDefault
	if foundExtMinorVersion >= 9 {
		topLevelField = statementSQLTopLevelFieldMinorVersion9
	}

	totalTimeField := statementSQLTotalTimeFieldDefault
	if foundExtMinorVersion >= 8 {
		totalTimeField = statementSQLTotalTimeFieldMinorVersion8
	}

	ioTimeFields := statementSQLIoTimeFieldsDefault
	if foundExtMinorVersion >= 11 {
		ioTimeFields = statementSQLIoTimeFieldsMinorVersion11
	}

	optionalFields := statementSQLOptionalFieldsDefault
	if foundExtMinorVersion >= 8 {
		optionalFields = statementSQLOptionalFieldsMinorVersion8
	}

	querySql := QueryMarkerSQL + fmt.Sprintf(statementStatsSQL, topLevelField, totalTimeField, ioTimeFields, optionalFields, sourceTable)
	stmt, err := db.PrepareContext(ctx, querySql)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	statementStats := make(state.PostgresStatementStatsMap)

	for rows.Next() {
		var key state.PostgresStatementKey
		var queryID null.Int
		var stats state.PostgresStatementStats

		err = rows.Scan(&key.DatabaseOid, &key.UserOid, &queryID, &key.TopLevel, &stats.Calls, &stats.TotalTime, &stats.Rows,
			&stats.SharedBlksHit, &stats.SharedBlksRead, &stats.SharedBlksDirtied, &stats.SharedBlksWritten,
			&stats.LocalBlksHit, &stats.LocalBlksRead, &stats.LocalBlksDirtied, &stats.LocalBlksWritten,
			&stats.TempBlksRead, &stats.TempBlksWritten, &stats.BlkReadTime, &stats.BlkWriteTime,
			&stats.MinTime, &stats.MaxTime, &stats.MeanTime, &stats.StddevTime)
		if err != nil {
			return nil, err
		}

		if queryID.Valid {
			key.QueryID = queryID.Int64
		} else {
			// We can't process this entry, most likely a permission problem with reading the query ID
			continue
		}

		statementStats[key] = stats
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	c.SelfTest.MarkCollectionAspectOk(state.CollectionAspectPgStatStatements)

	return statementStats, nil
}

func GetStatementTexts(ctx context.Context, c *Collection, db *sql.DB) (state.PostgresStatementMap, state.PostgresStatementTextMap, error) {
	sourceTable, foundExtMinorVersion, err := getStatementSource(ctx, c, db, true)
	if err != nil {
		return nil, nil, err
	}

	topLevelField := statementSQLTopLevelFieldDefault
	if foundExtMinorVersion >= 9 {
		topLevelField = statementSQLTopLevelFieldMinorVersion9
	}

	querySql := QueryMarkerSQL + fmt.Sprintf(statementTextSQL, topLevelField, sourceTable)
	stmt, err := db.PrepareContext(ctx, querySql)
	if err != nil {
		return nil, nil, err
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var tmpFile *os.File

	tmpFile, err = os.CreateTemp("", util.TempFilePrefix)
	if err != nil {
		return nil, nil, err
	}
	defer tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	statements := make(state.PostgresStatementMap)
	statementTextsByFp := make(state.PostgresStatementTextMap)

	queryKeys := make([]state.PostgresStatementKey, 0)
	queryLengths := make([]int, 0)

	for rows.Next() {
		var key state.PostgresStatementKey
		var queryID null.Int
		var receivedQuery null.String

		err = rows.Scan(&key.DatabaseOid, &key.UserOid, &queryID, &key.TopLevel, &receivedQuery)
		if err != nil {
			return nil, nil, err
		}

		if queryID.Valid {
			key.QueryID = queryID.Int64
		} else {
			// We can't process this entry, most likely a permission problem with reading the query ID
			continue
		}

		queryKeys = append(queryKeys, key)
		queryLengths = append(queryLengths, len(receivedQuery.String))
		tmpFile.WriteString(receivedQuery.String)
	}

	if err = rows.Err(); err != nil {
		return nil, nil, err
	}

	tmpFile.Seek(0, io.SeekStart)
	for idx, length := range queryLengths {
		bytes := make([]byte, length)
		_, err = io.ReadFull(tmpFile, bytes)
		if err != nil {
			return nil, nil, err
		}
		query := string(bytes)
		ignoreIoTiming := ignoreIOTiming(c.PostgresVersion, query)
		key := queryKeys[idx]
		select {
		// Since normalizing can take time, explicitly check for cancellations
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		default:
			fingerprintAndNormalize(c, key, key.QueryID, query, statements, statementTextsByFp, ignoreIoTiming)
		}
	}

	c.SelfTest.MarkCollectionAspectOk(state.CollectionAspectPgStatStatements)

	return statements, statementTextsByFp, nil
}

func getStatementSource(ctx context.Context, c *Collection, db *sql.DB, showtext bool) (string, int16, error) {
	var err error
	var sourceTable string
	var extSchema string
	var extMinorVersion int16
	var foundExtMinorVersion int16

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
		return "", 0, err
	}

	if err == sql.ErrNoRows {
		c.Logger.PrintInfo("pg_stat_statements does not exist, trying to create extension...")
		_, err = db.ExecContext(ctx, QueryMarkerSQL+"CREATE EXTENSION IF NOT EXISTS pg_stat_statements SCHEMA public")
		if err != nil {
			c.SelfTest.MarkCollectionAspectError(state.CollectionAspectPgStatStatements, "extension does not exist in database %s and could not be created: %s", c.Config.DbName, err)
			c.Logger.PrintInfo("HINT - if you expect the extension to already be installed, please review the pganalyze documentation: https://pganalyze.com/docs/install/troubleshooting/pg_stat_statements")
			return "", 0, err
		}
		extSchema = "public"
		foundExtMinorVersion = extMinorVersion
	}

	if foundExtMinorVersion < 3 {
		c.SelfTest.MarkCollectionAspectError(state.CollectionAspectPgStatStatements, "extension version too old in database %s (1.%d installed, 1.3+ required)", c.Config.DbName, foundExtMinorVersion)
		return "", 0, fmt.Errorf("pg_stat_statements version too old in database %s (1.%d installed, 1.3+ required). To update run `ALTER EXTENSION pg_stat_statements UPDATE` in database %s", c.Config.DbName, foundExtMinorVersion, c.Config.DbName)
	}

	if c.GlobalOpts.TestRun {
		if extMinorVersion >= 9 && foundExtMinorVersion < 9 {
			// Using the older version pgss with Postgres 14+ can cause the incorrect query stats
			// when track = all is used + there are toplevel queries and nested queries
			// https://github.com/pganalyze/collector/pull/472#discussion_r1399976152
			c.Logger.PrintError("Outdated pg_stat_statements may cause incorrect query statistics")
			c.SelfTest.MarkCollectionAspectError(state.CollectionAspectPgStatStatements, "extension version too old in database %s (1.%d installed, 1.9+ required). Outdated pg_stat_statements will cause incorrect query statistics.", c.Config.DbName, foundExtMinorVersion)
			c.SelfTest.HintCollectionAspect(state.CollectionAspectPgStatStatements, "Update the extension by running `ALTER EXTENSION pg_stat_statements UPDATE`.")
		} else if foundExtMinorVersion < extMinorVersion {
			pgssMsg := fmt.Sprintf("extension outdated in database %s (1.%d installed, 1.%d available)", c.Config.DbName, foundExtMinorVersion, extMinorVersion)
			c.Logger.PrintInfo("pg_stat_statements %s. To update run `ALTER EXTENSION pg_stat_statements UPDATE`", pgssMsg)
			c.SelfTest.MarkCollectionAspectWarning(state.CollectionAspectPgStatStatements, "%s", pgssMsg)
			c.SelfTest.HintCollectionAspect(state.CollectionAspectPgStatStatements, "To update run `ALTER EXTENSION pg_stat_statements UPDATE`")
		}
	}

	if c.HelperExists("get_stat_statements", []string{"boolean"}) || (showtext && c.HelperExists("get_stat_statements", nil)) {
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
			c.SelfTest.HintCollectionAspect(state.CollectionAspectPgStatStatements, "Please make sure the monitoring user used by the collector has been granted the pg_monitor role or is a superuser.")
			c.Logger.PrintInfo("Warning: Monitoring user may have insufficient permissions to capture all queries.\n" +
				"You are not connecting as a superuser." +
				" Please make sure the monitoring user used by the collector has been granted the pg_monitor role or is a superuser in order to get query statistics for all roles.")
			if c.Config.SystemType == "aiven" {
				docsLink := "https://pganalyze.com/docs/install/aiven/03_create_pg_stat_statements_helpers"
				c.SelfTest.HintCollectionAspect(state.CollectionAspectPgStatStatements, "For aiven, you can also set up the monitoring helper functions (%s).", selftest.URLPrinter.Sprint(docsLink))
				c.Logger.PrintInfo("For Aiven, you can also set up the monitoring helper functions (%s).", docsLink)
			}
		}
		if !showtext {
			sourceTable = extSchema + ".pg_stat_statements(false)"
		} else {
			sourceTable = extSchema + ".pg_stat_statements"
		}
	}

	return sourceTable, foundExtMinorVersion, nil
}

func ignoreIOTiming(postgresVersion state.PostgresVersion, receivedQuery string) bool {
	// Currently, Aurora gives wildly incorrect blk_read_time and blk_write_time values
	// for utility statements; ignore I/O timing in this situation.
	if !postgresVersion.IsAwsAurora || receivedQuery == "" {
		return false
	}

	isUtil, err := pg_query.IsUtilityStmt(receivedQuery)
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

func fingerprintAndNormalize(c *Collection, key state.PostgresStatementKey, queryID int64, text string, statements state.PostgresStatementMap, statementTextsByFp state.PostgresStatementTextMap, ignoreIoTiming bool) {
	if insufficientPrivilege(text) {
		statements[key] = state.PostgresStatement{
			InsufficientPrivilege: true,
			Fingerprint:           insufficientPrivsQueryFingerprint,
			IgnoreIoTiming:        ignoreIoTiming,
		}
	} else if collectorStatement(text) {
		statements[key] = state.PostgresStatement{
			Collector:      true,
			Fingerprint:    collectorQueryFingerprint,
			IgnoreIoTiming: ignoreIoTiming,
		}
	} else {
		fp := c.Fingerprints.LoadOrStore(queryID, text, c.Config.FilterQueryText, -1)
		statements[key] = state.PostgresStatement{Fingerprint: fp, IgnoreIoTiming: ignoreIoTiming}
		_, ok := statementTextsByFp[fp]
		if !ok {
			statementTextsByFp[fp] = util.NormalizeQuery(text, c.Config.FilterQueryText, -1)
		}
	}
}

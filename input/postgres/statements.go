package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/guregu/null"
	"github.com/pganalyze/collector/scheduler"
	"github.com/pganalyze/collector/selftest"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	pg_query "github.com/pganalyze/pg_query_go/v6"
)

// pg_stat_statements 1.3+ (Postgres 9.5+)
const statementSQLTopLevelFieldDefault = "true"
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
       split_part(extversion, '.', 2),
       split_part(pae.default_version, '.', 2)
  FROM pg_extension pge
 INNER JOIN pg_namespace pgn ON pge.extnamespace = pgn.oid
  LEFT JOIN pg_available_extensions pae ON pae.name = pge.extname
 WHERE pge.extname = 'pg_stat_statements'
`

func collectorStatement(query string) bool {
	return strings.HasPrefix(query, QueryMarkerSQL)
}

func insufficientPrivilege(query string) bool {
	return query == "<insufficient privilege>"
}

const resetThreshold = 0.9

func ShouldResetStatements(server *state.Server, ps *state.PersistedState, ts *state.TransientState, size int) (reset bool, err error) {
	config := server.Grant.Load().Config
	lastReset := ps.PgStatStatementsStats.Reset
	resetFreq := config.Features.StatementResetFrequency * scheduler.FullSnapshotMinutes
	maxSize := int(config.Features.StatementMaxSizeMb)
	if !lastReset.Valid {
		return // It's always set on PG14+ with the extension enabled. Older versions aren't supported
	}
	if maxSize == 0 {
		maxSize = 250
	}
	entryCount := len(ts.Statements)
	entryMax := 0
	for _, setting := range ts.Settings {
		if setting.Name == "pg_stat_statements.max" && setting.CurrentValue.Valid {
			entryMax, err = strconv.Atoi(setting.CurrentValue.String)
			if err != nil {
				return
			}
		}
	}
	if entryMax == 0 {
		err = errors.New("Could not find pg_stat_statements.max setting")
		return
	}
	resetAllowed := resetFreq > 0 && time.Since(lastReset.Time).Minutes() >= float64(resetFreq)
	tooMany := float64(entryCount) >= float64(entryMax)*resetThreshold
	tooLarge := size > maxSize*1024*1024
	reset = resetAllowed && (tooMany || tooLarge)
	return
}

func ResetStatements(ctx context.Context, c *Collection, db *sql.DB) (err error) {
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
	_, err = db.ExecContext(ctx, QueryMarkerSQL+"SELECT "+method)
	return
}

func GetStatementStats(ctx context.Context, c *Collection, db *sql.DB) (state.PostgresStatementStatsMap, error) {
	source, err := getStatementSource(ctx, c, db, false)
	if err != nil {
		return nil, err
	}

	topLevelField := statementSQLTopLevelFieldDefault
	if source.MinorVersion >= 9 {
		topLevelField = statementSQLTopLevelFieldMinorVersion9
	}

	totalTimeField := statementSQLTotalTimeFieldDefault
	if source.MinorVersion >= 8 {
		totalTimeField = statementSQLTotalTimeFieldMinorVersion8
	}

	ioTimeFields := statementSQLIoTimeFieldsDefault
	if source.MinorVersion >= 11 {
		ioTimeFields = statementSQLIoTimeFieldsMinorVersion11
	}

	optionalFields := statementSQLOptionalFieldsDefault
	if source.MinorVersion >= 8 {
		optionalFields = statementSQLOptionalFieldsMinorVersion8
	}

	querySql := QueryMarkerSQL + fmt.Sprintf(statementStatsSQL, topLevelField, totalTimeField, ioTimeFields, optionalFields, source.Table)
	rows, err := db.QueryContext(ctx, querySql)
	if err != nil {
		return nil, source.hintOutdatedExtension(c, err)
	}
	defer rows.Close()

	statementStats := make(state.PostgresStatementStatsMap)

	for rows.Next() {
		var key state.PostgresStatementKey
		var queryID null.Int
		var stats state.PostgresStatementStats

		err = rows.Scan(&key.DatabaseOid, &key.UserOid, &queryID, &key.Toplevel, &stats.Calls, &stats.TotalTime, &stats.Rows,
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
		return nil, source.hintOutdatedExtension(c, err)
	}

	c.SelfTest.MarkCollectionAspectOk(state.CollectionAspectPgStatStatements)

	return statementStats, nil
}

func GetStatementTexts(ctx context.Context, c *Collection, db *sql.DB) (statements state.PostgresStatementMap, statementTextsByFp state.PostgresStatementTextMap, querySize int, err error) {
	source, err := getStatementSource(ctx, c, db, true)
	if err != nil {
		return
	}

	topLevelField := statementSQLTopLevelFieldDefault
	if source.MinorVersion >= 9 {
		topLevelField = statementSQLTopLevelFieldMinorVersion9
	}

	querySql := QueryMarkerSQL + fmt.Sprintf(statementTextSQL, topLevelField, source.Table)
	rows, err := db.QueryContext(ctx, querySql)
	if err != nil {
		err = source.hintOutdatedExtension(c, err)
		return
	}
	defer rows.Close()

	var tmpFile *os.File

	tmpFile, err = os.CreateTemp("", util.TempFilePrefix)
	if err != nil {
		return
	}
	defer tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	statements = make(state.PostgresStatementMap)
	statementTextsByFp = make(state.PostgresStatementTextMap)
	queryKeys := make([]state.PostgresStatementKey, 0)
	queryLengths := make([]int, 0)

	for rows.Next() {
		var key state.PostgresStatementKey
		var queryID null.Int
		var receivedQuery null.String

		err = rows.Scan(&key.DatabaseOid, &key.UserOid, &queryID, &key.Toplevel, &receivedQuery)
		if err != nil {
			return
		}
		querySize += len(receivedQuery.String)

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
		err = source.hintOutdatedExtension(c, err)
		return
	}

	tmpFile.Seek(0, io.SeekStart)
	for idx, length := range queryLengths {
		bytes := make([]byte, length)
		_, err = io.ReadFull(tmpFile, bytes)
		if err != nil {
			return
		}
		query := string(bytes)
		ignoreIoTiming := ignoreIOTiming(c.PostgresVersion, query)
		key := queryKeys[idx]
		select {
		// Since normalizing can take time, explicitly check for cancellations
		case <-ctx.Done():
			err = ctx.Err()
			return
		default:
			fingerprintAndNormalize(c, key, key.QueryID, query, statements, statementTextsByFp, ignoreIoTiming)
		}
	}

	c.SelfTest.MarkCollectionAspectOk(state.CollectionAspectPgStatStatements)

	return
}

// statementSource describes where to read pg_stat_statements data from, and which
// version of the extension we are dealing with
type statementSource struct {
	// Table or function to select the statistics from
	Table string
	// Minor version of the extension installed in the current database
	MinorVersion int16
	// Minor version the extension can be updated to in the current database
	AvailableMinorVersion int16
}

// hintOutdatedExtension adds an actionable hint to errors from querying
// pg_stat_statements when the extension in the current database is older than the
// version available on the server. Most commonly this happens after a Postgres major
// version upgrade without a subsequent `ALTER EXTENSION pg_stat_statements UPDATE`,
// where the extension's function signature may no longer match what the loaded library
// expects ("incorrect number of output arguments").
//
// Like the other extension version warnings this is limited to test runs, to avoid
// repeating the same hint in the logs for every snapshot.
func (source statementSource) hintOutdatedExtension(c *Collection, err error) error {
	if !c.GlobalOpts.TestRun || source.MinorVersion >= source.AvailableMinorVersion {
		return err
	}
	c.SelfTest.MarkCollectionAspectError(state.CollectionAspectPgStatStatements, "%s (extension outdated in database %s: 1.%d installed, 1.%d available)", err, c.Config.DbName, source.MinorVersion, source.AvailableMinorVersion)
	c.SelfTest.HintCollectionAspect(state.CollectionAspectPgStatStatements, "To update run `ALTER EXTENSION pg_stat_statements UPDATE`")
	return fmt.Errorf("%w - note pg_stat_statements is outdated in database %s (1.%d installed, 1.%d available), to update run `ALTER EXTENSION pg_stat_statements UPDATE`", err, c.Config.DbName, source.MinorVersion, source.AvailableMinorVersion)
}

func getStatementSource(ctx context.Context, c *Collection, db *sql.DB, showtext bool) (statementSource, error) {
	var err error
	var sourceTable string
	var extSchema string
	var bundledExtMinorVersion int16
	var foundExtMinorVersion int16
	var defaultExtMinorVersion null.String

	// Version of the extension that ships with this Postgres version, used as a
	// fallback when the server does not tell us which version is available
	if c.PostgresVersion.Numeric >= state.PostgresVersion18 {
		bundledExtMinorVersion = 12
	} else if c.PostgresVersion.Numeric >= state.PostgresVersion17 {
		bundledExtMinorVersion = 11
	} else if c.PostgresVersion.Numeric >= state.PostgresVersion14 {
		bundledExtMinorVersion = 9
	} else if c.PostgresVersion.Numeric >= state.PostgresVersion13 {
		bundledExtMinorVersion = 8
	} else {
		bundledExtMinorVersion = 3
	}

	err = db.QueryRowContext(ctx, QueryMarkerSQL+statementExtensionVersionSQL).Scan(&extSchema, &foundExtMinorVersion, &defaultExtMinorVersion)
	if err != nil && err != sql.ErrNoRows {
		return statementSource{}, err
	}

	if err == sql.ErrNoRows {
		c.Logger.PrintInfo("pg_stat_statements does not exist, trying to create extension...")
		_, err = db.ExecContext(ctx, QueryMarkerSQL+"CREATE EXTENSION IF NOT EXISTS pg_stat_statements SCHEMA public")
		if err != nil {
			c.SelfTest.MarkCollectionAspectError(state.CollectionAspectPgStatStatements, "extension does not exist in database %s and could not be created: %s", c.Config.DbName, err)
			c.Logger.PrintInfo("HINT - if you expect the extension to already be installed, please review the pganalyze documentation: https://pganalyze.com/docs/install/troubleshooting/pg_stat_statements")
			return statementSource{}, err
		}
		extSchema = "public"
		foundExtMinorVersion = bundledExtMinorVersion
	}

	// Prefer the version the server reports as available, since that's what
	// `ALTER EXTENSION pg_stat_statements UPDATE` updates to, and it keeps working for
	// Postgres versions that are newer than this collector release. Fall back to the
	// bundled version if the server doesn't report one, or reports one we can't parse
	// (some providers add suffixes to the extension version).
	availableExtMinorVersion := bundledExtMinorVersion
	if defaultExtMinorVersion.Valid {
		parsed, parseErr := strconv.ParseInt(defaultExtMinorVersion.String, 10, 16)
		if parseErr == nil {
			availableExtMinorVersion = int16(parsed)
		} else if c.GlobalOpts.TestRun {
			c.Logger.PrintVerbose("Could not determine available pg_stat_statements version in database %s (unexpected minor version \"%s\"), assuming 1.%d", c.Config.DbName, defaultExtMinorVersion.String, bundledExtMinorVersion)
		}
	}

	if foundExtMinorVersion < 3 {
		c.SelfTest.MarkCollectionAspectError(state.CollectionAspectPgStatStatements, "extension version too old in database %s (1.%d installed, 1.3+ required)", c.Config.DbName, foundExtMinorVersion)
		return statementSource{}, fmt.Errorf("pg_stat_statements version too old in database %s (1.%d installed, 1.3+ required). To update run `ALTER EXTENSION pg_stat_statements UPDATE` in database %s", c.Config.DbName, foundExtMinorVersion, c.Config.DbName)
	}

	if c.GlobalOpts.TestRun {
		if c.PostgresVersion.Numeric >= state.PostgresVersion14 && foundExtMinorVersion < 9 {
			// Using the older version pgss with Postgres 14+ can cause the incorrect query stats
			// when track = all is used + there are toplevel queries and nested queries
			// https://github.com/pganalyze/collector/pull/472#discussion_r1399976152
			c.Logger.PrintError("Outdated pg_stat_statements may cause incorrect query statistics")
			c.SelfTest.MarkCollectionAspectError(state.CollectionAspectPgStatStatements, "extension version too old in database %s (1.%d installed, 1.9+ required). Outdated pg_stat_statements will cause incorrect query statistics.", c.Config.DbName, foundExtMinorVersion)
			c.SelfTest.HintCollectionAspect(state.CollectionAspectPgStatStatements, "Update the extension by running `ALTER EXTENSION pg_stat_statements UPDATE`.")
		} else if foundExtMinorVersion < availableExtMinorVersion {
			pgssMsg := fmt.Sprintf("extension outdated in database %s (1.%d installed, 1.%d available)", c.Config.DbName, foundExtMinorVersion, availableExtMinorVersion)
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

	return statementSource{
		Table:                 sourceTable,
		MinorVersion:          foundExtMinorVersion,
		AvailableMinorVersion: availableExtMinorVersion,
	}, nil
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

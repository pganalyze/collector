package postgres

import (
	"bufio"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pganalyze/collector/logs"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

const LogFileSql = "SELECT name FROM pg_catalog.pg_ls_logdir() WHERE modification > pg_catalog.now() - '2 minute'::interval"

// Read at most the trailing 10 megabytes of each file
const SuperUserReadLogFileSql = `
SELECT (SELECT size FROM pg_catalog.pg_ls_logdir() WHERE name = $1),
  pg_catalog.pg_read_file(
	pg_catalog.current_setting('data_directory') || '/' || pg_catalog.current_setting('log_directory') || '/' || $1,
	(SELECT GREATEST(size - 1024 * 1024 * 10, $2) FROM pg_catalog.pg_ls_logdir() WHERE name = $1),
	1024 * 1024 * 10
  )
;`
const HelperReadLogFile = `
SELECT (SELECT size FROM pg_catalog.pg_ls_logdir() WHERE name = $1),
  pganalyze.read_log_file(
	$1,
	(SELECT GREATEST(size - 1024 * 1024 * 10, $2) FROM pg_catalog.pg_ls_logdir() WHERE name = $1),
	1024 * 1024 * 10
  )
`

// LogPgReadFile - Gets log files using the pg_read_file function
func LogPgReadFile(ctx context.Context, server *state.Server, opts state.CollectionOpts, logger *util.Logger) (state.PersistedLogState, []state.LogFile, []state.PostgresQuerySample, error) {
	var err error
	var psl state.PersistedLogState = server.LogPrevState
	var logFiles []state.LogFile
	var samples []state.PostgresQuerySample

	linesNewerThan := time.Now().Add(-2 * time.Minute)

	db, err := EstablishConnection(ctx, server, logger, opts, "")
	if err != nil {
		logger.PrintWarning("Could not connect to fetch logs: %s", err)
		return server.LogPrevState, nil, nil, err
	}
	defer db.Close()

	h, err := NewCollection(ctx, logger, server, opts, db)
	if err != nil {
		logger.PrintError("Error setting up collection helper: %s", err)
		return server.LogPrevState, nil, nil, err
	}

	rows, err := db.QueryContext(ctx, QueryMarkerSQL+LogFileSql)
	if err != nil {
		err = fmt.Errorf("LogFileSql/Query: %s", err)
		return server.LogPrevState, nil, nil, err
	}
	defer rows.Close()

	var fileNames []string
	for rows.Next() {
		var fileName string
		err = rows.Scan(&fileName)
		if err != nil {
			err = fmt.Errorf("LogFileSql/Scan: %s", err)
			return server.LogPrevState, nil, nil, err
		}
		fileNames = append(fileNames, fileName)
	}

	if err = rows.Err(); err != nil {
		err = fmt.Errorf("LogFileSql/Rows: %s", err)
		return server.LogPrevState, nil, nil, err
	}

	useHelper := h.HelperExists("read_log_file", []string{"text", "bigint", "bigint"})
	var logReadSql = SuperUserReadLogFileSql
	if useHelper {
		logger.PrintVerbose("Found pganalyze.read_log_file() stats helper")
		logReadSql = HelperReadLogFile
	}

	var newMarkers = make(map[string]int64)
	for _, fileName := range fileNames {
		if err != nil {
			err = fmt.Errorf("LogFileSql/Scan: %s", err)
			return server.LogPrevState, nil, nil, err
		}
		var logData string
		var newOffset int64
		prevOffset := psl.ReadFileMarkers[fileName]
		err = db.QueryRowContext(ctx, QueryMarkerSQL+logReadSql, fileName, prevOffset).Scan(&newOffset, &logData)
		if err != nil {
			err = fmt.Errorf("LogReadSql/QueryRow: %s", err)
			return server.LogPrevState, nil, nil, err
		}

		var logFile state.LogFile
		logFile, err = state.NewLogFile(fileName)
		if err != nil {
			err = fmt.Errorf("error initializing log file: %s", err)
			return server.LogPrevState, nil, nil, err
		}

		logReader := bufio.NewReader(strings.NewReader(logData))
		newLogLines, newSamples := logs.ParseAndAnalyzeBuffer(logReader, linesNewerThan, server)
		logFile.LogLines = append(logFile.LogLines, newLogLines...)
		samples = append(samples, newSamples...)

		newMarkers[fileName] = newOffset

		logFiles = append(logFiles, logFile)
	}
	psl.ReadFileMarkers = newMarkers

	return psl, logFiles, samples, err
}

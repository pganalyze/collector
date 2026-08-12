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

// Only consider log files that may have been written to since the previous log
// download, based on the configured log download interval
const LogFileSql = "SELECT name FROM pg_catalog.pg_ls_logdir() WHERE modification > pg_catalog.now() - pg_catalog.make_interval(secs => $1)"

// Read at most the trailing $3 bytes of each file, as determined by
// ServerConfig.MaxLogParsingSize()
const SuperUserReadLogFileSql = `
SELECT (SELECT size FROM pg_catalog.pg_ls_logdir() WHERE name = $1),
  pg_catalog.pg_read_file(
	pg_catalog.current_setting('data_directory') || '/' || pg_catalog.current_setting('log_directory') || '/' || $1,
	(SELECT GREATEST(size - $3, $2) FROM pg_catalog.pg_ls_logdir() WHERE name = $1),
	$3
  )
;`
const HelperReadLogFile = `
SELECT (SELECT size FROM pg_catalog.pg_ls_logdir() WHERE name = $1),
  pganalyze.read_log_file(
	$1,
	(SELECT GREATEST(size - $3, $2) FROM pg_catalog.pg_ls_logdir() WHERE name = $1),
	$3
  )
`

// LogPgReadFile - Gets log files using the pg_read_file function
func LogPgReadFile(ctx context.Context, server *state.Server, opts state.CollectionOpts, logger *util.Logger) (state.PersistedLogState, []state.LogFile, []state.PostgresQuerySample, error) {
	var err error
	var psl state.PersistedLogState = server.LogPrevState
	var logFiles []state.LogFile
	var samples []state.PostgresQuerySample

	logDownloadWindow := server.Config.LogDownloadWindow()
	linesNewerThan := time.Now().Add(-logDownloadWindow)
	maxLogParsingSize := server.Config.MaxLogParsingSize()

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

	rows, err := db.QueryContext(ctx, QueryMarkerSQL+LogFileSql, logDownloadWindow.Seconds())
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
		err = db.QueryRowContext(ctx, QueryMarkerSQL+logReadSql, fileName, prevOffset, maxLogParsingSize).Scan(&newOffset, &logData)
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
		newLogLines, newSamples := logs.ParseAndAnalyzeBuffer(logReader, linesNewerThan, server, opts, logger)
		logFile.LogLines = append(logFile.LogLines, newLogLines...)
		samples = append(samples, newSamples...)

		newMarkers[fileName] = newOffset

		logFiles = append(logFiles, logFile)
	}
	psl.ReadFileMarkers = newMarkers

	return psl, logFiles, samples, err
}

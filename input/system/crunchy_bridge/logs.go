package crunchy_bridge

import (
	"fmt"
	"io/ioutil"
	"time"

	"github.com/pganalyze/collector/input/postgres"
	"github.com/pganalyze/collector/logs"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	uuid "github.com/satori/go.uuid"
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

// DownloadLogFiles - Gets log files for a Crunchy Bridge server
func DownloadLogFiles(server *state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) (state.PersistedLogState, []state.LogFile, []state.PostgresQuerySample, error) {
	var err error
	var psl state.PersistedLogState = server.LogPrevState
	var logFiles []state.LogFile
	var samples []state.PostgresQuerySample

	linesNewerThan := time.Now().Add(-2 * time.Minute)

	db, err := postgres.EstablishConnection(server, logger, globalCollectionOpts, "")
	if err != nil {
		logger.PrintWarning("Could not connect to fetch logs: %s", err)
		return server.LogPrevState, nil, nil, err
	}
	defer db.Close()

	rows, err := db.Query(postgres.QueryMarkerSQL + LogFileSql)
	if err != nil {
		err = fmt.Errorf("LogFileSql/Query: %s", err)
		return server.LogPrevState, nil, nil, err
	}

	var fileNames []string
	for rows.Next() {
		var fileName string
		err = rows.Scan(&fileName)
		fileNames = append(fileNames, fileName)
	}
	rows.Close()

	useHelper := postgres.StatsHelperExists(db, "read_log_file")
	var logReadSql = SuperUserReadLogFileSql
	if useHelper {
		logger.PrintVerbose("Found pganalyze.read_log_file() stats helper")
		logReadSql = HelperReadLogFile
	}

	var newMarkers = make(map[string]int64)
	for _, fileName := range fileNames {
		if err != nil {
			err = fmt.Errorf("LogFileSql/Scan: %s", err)
			goto ErrorCleanup
		}
		var logData string
		var newOffset int64
		prevOffset, _ := psl.ReadFileMarkers[fileName]
		err = db.QueryRow(postgres.QueryMarkerSQL+logReadSql, fileName, prevOffset).Scan(&newOffset, &logData)
		if err != nil {
			err = fmt.Errorf("LogReadSql/QueryRow: %s", err)
			goto ErrorCleanup
		}

		var logFile state.LogFile
		logFile.UUID = uuid.NewV4()
		logFile.TmpFile, err = ioutil.TempFile("", "")
		if err != nil {
			err = fmt.Errorf("Error allocating tempfile for logs: %s", err)
			goto ErrorCleanup
		}
		logFile.OriginalName = fileName

		_, err := logFile.TmpFile.WriteString(logData)
		if err != nil {
			err = fmt.Errorf("Error writing to tempfile: %s", err)
			logFile.Cleanup()
			goto ErrorCleanup
		}

		newLogLines, newSamples, _ := logs.ParseAndAnalyzeBuffer(logData, 0, linesNewerThan)
		logFile.LogLines = append(logFile.LogLines, newLogLines...)
		samples = append(samples, newSamples...)

		newMarkers[fileName] = newOffset

		logFiles = append(logFiles, logFile)
	}
	psl.ReadFileMarkers = newMarkers

	return psl, logFiles, samples, err

ErrorCleanup:
	for _, logFile := range logFiles {
		logFile.Cleanup()
	}

	return server.LogPrevState, nil, nil, err
}

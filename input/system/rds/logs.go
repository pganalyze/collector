package rds

import (
	"fmt"
	"io"
	"io/ioutil"
	"time"

	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/logs"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	"github.com/pganalyze/collector/util/awsutil"
	uuid "github.com/satori/go.uuid"
)

// Read at most the trailing 10 megabytes of the temp file, to avoid OOMs on initial start of the collector
// (where the full RDS log file gets downloaded before the marker gets initialized)
const maxLogParsingSize = 10 * 1024 * 1024

// DownloadLogFiles - Gets log files for an Amazon RDS instance
func DownloadLogFiles(prevState state.PersistedLogState, config config.ServerConfig, logger *util.Logger) (psl state.PersistedLogState, result []state.LogFile, samples []state.PostgresQuerySample, err error) {
	psl = prevState

	sess, awsErr := awsutil.GetAwsSession(config)
	if awsErr != nil {
		err = fmt.Errorf("Error getting session: %s", awsErr)
		return
	}

	rdsSvc := rds.New(sess)

	instance, awsErr := awsutil.FindRdsInstance(config, sess)
	if awsErr != nil {
		err = fmt.Errorf("Error finding RDS instance: %s", awsErr)
		return
	}

	// Retrieve all possibly matching logfiles in the last two minutes, assuming
	// the collector's scheduler that runs more frequently than that
	linesNewerThan := time.Now().Add(-2 * time.Minute)
	lastWritten := linesNewerThan.Unix() * 1000

	params := &rds.DescribeDBLogFilesInput{
		DBInstanceIdentifier: instance.DBInstanceIdentifier,
		FileLastWritten:      &lastWritten,
	}

	resp, awsErr := rdsSvc.DescribeDBLogFiles(params)
	if awsErr != nil {
		err = fmt.Errorf("Error listing RDS log files: %s", awsErr)
		return
	}

	var newMarkers = make(map[string]string)
	var bytesWritten = 0

	for _, rdsLogFile := range resp.DescribeDBLogFiles {
		var lastMarker *string

		prevMarker, ok := psl.AwsMarkers[*rdsLogFile.LogFileName]
		if ok {
			lastMarker = &prevMarker
		}

		var logFile state.LogFile
		logFile.UUID = uuid.NewV4()
		logFile.TmpFile, awsErr = ioutil.TempFile("", "")
		if awsErr != nil {
			err = fmt.Errorf("Error allocating tempfile for logs: %s", awsErr)
			goto ErrorCleanup
		}
		logFile.OriginalName = *rdsLogFile.LogFileName

		for {
			resp, awsErr := rdsSvc.DownloadDBLogFilePortion(&rds.DownloadDBLogFilePortionInput{
				DBInstanceIdentifier: instance.DBInstanceIdentifier,
				LogFileName:          rdsLogFile.LogFileName,
				Marker:               lastMarker, // This is not set for the initial call, so we only get the most recent lines
			})

			if awsErr != nil {
				err = fmt.Errorf("Error downloading logs: %s", awsErr)
				logFile.Cleanup()
				goto ErrorCleanup
			}

			if resp.LogFileData == nil {
				logger.PrintVerbose("Rds/Logs: No log data in response, skipping")
				break
			}

			if len(*resp.LogFileData) > 0 {
				_, goErr := logFile.TmpFile.WriteString(*resp.LogFileData)
				if goErr != nil {
					err = fmt.Errorf("Error writing to tempfile: %s", goErr)
					logFile.Cleanup()
					goto ErrorCleanup
				}
				bytesWritten += len(*resp.LogFileData)
			}

			lastMarker = resp.Marker

			if !*resp.AdditionalDataPending {
				break
			}
		}

		var newLogLines []state.LogLine
		var newSamples []state.PostgresQuerySample

		readStart := bytesWritten - maxLogParsingSize
		if readStart < 0 {
			readStart = 0
		}
		_, goErr := logFile.TmpFile.Seek(int64(readStart), io.SeekStart)
		if goErr != nil {
			err = fmt.Errorf("Error seeking tempfile: %s", goErr)
			logFile.Cleanup()
			goto ErrorCleanup
		}

		buf := make([]byte, bytesWritten - readStart)
		bytesRead, goErr := logFile.TmpFile.Read(buf)
		if goErr != nil {
			err = fmt.Errorf("Error reading %d bytes from tempfile: %s", len(buf), goErr)
			logFile.Cleanup()
			goto ErrorCleanup
		}
		if bytesRead != len(buf) {
			err = fmt.Errorf("Mismatch of bytes read (%d) vs bytes expected (%d) from tempfile", bytesRead, len(buf))
			logFile.Cleanup()
			goto ErrorCleanup
		}

		newLogLines, newSamples, _ = logs.ParseAndAnalyzeBuffer(string(buf), 0, linesNewerThan)
		logFile.LogLines = append(logFile.LogLines, newLogLines...)
		samples = append(samples, newSamples...)

		if lastMarker != nil {
			newMarkers[*rdsLogFile.LogFileName] = *lastMarker
		}

		result = append(result, logFile)
	}
	psl.AwsMarkers = newMarkers

	return

ErrorCleanup:
	for _, logFile := range result {
		logFile.Cleanup()
	}

	return prevState, nil, nil, err
}

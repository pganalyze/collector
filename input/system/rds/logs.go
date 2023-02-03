package rds

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/logs"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	"github.com/pganalyze/collector/util/awsutil"
	uuid "github.com/satori/go.uuid"
)

// DownloadLogFiles - Gets log files for an Amazon RDS instance
func DownloadLogFiles(server *state.Server, logger *util.Logger) (state.PersistedLogState, []state.LogFile, []state.PostgresQuerySample, error) {
	var err error
	var psl state.PersistedLogState = server.LogPrevState
	var logFiles []state.LogFile
	var samples []state.PostgresQuerySample

	sess, err := awsutil.GetAwsSession(server.Config)
	if err != nil {
		err = fmt.Errorf("Error getting session: %s", err)
		return server.LogPrevState, nil, nil, err
	}

	identifier, err := getAwsDbInstanceID(server.Config, sess)
	if err != nil {
		return server.LogPrevState, nil, nil, err
	}

	// Retrieve all possibly matching logfiles in the last two minutes, assuming
	// the collector's scheduler that runs more frequently than that
	linesNewerThan := time.Now().Add(-2 * time.Minute)
	lastWritten := linesNewerThan.Unix() * 1000

	params := &rds.DescribeDBLogFilesInput{
		DBInstanceIdentifier: &identifier,
		FileLastWritten:      &lastWritten,
	}

	rdsSvc := rds.New(sess)

	resp, err := rdsSvc.DescribeDBLogFiles(params)
	if err != nil {
		err = fmt.Errorf("Error listing RDS log files: %s", err)
		return server.LogPrevState, nil, nil, err
	}

	var newMarkers = make(map[string]string)

	for _, rdsLogFile := range resp.DescribeDBLogFiles {
		var lastMarker *string
		var bytesWritten int64

		prevMarker, ok := psl.AwsMarkers[*rdsLogFile.LogFileName]
		if ok {
			lastMarker = &prevMarker
		}

		var tmpFile *os.File
		tmpFile, err = ioutil.TempFile("", "")
		if err != nil {
			err = fmt.Errorf("Error allocating tempfile for logs: %s", err)
			goto ErrorCleanup
		}

		for {
			var newBytesWritten int
			var newMarker *string
			var additionalDataPending bool
			newBytesWritten, newMarker, additionalDataPending, err = downloadRdsLogFilePortion(rdsSvc, tmpFile, logger, &identifier, rdsLogFile.LogFileName, lastMarker)
			if err != nil {
				tmpFile.Close()
				os.Remove(tmpFile.Name())
				goto ErrorCleanup
			}

			bytesWritten += int64(newBytesWritten)
			if newMarker != nil {
				lastMarker = newMarker
			}

			if !additionalDataPending {
				break
			}
		}

		var buf []byte
		buf, tmpFile, err = readLogFilePortion(tmpFile, bytesWritten, logger)
		if err != nil {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
			goto ErrorCleanup
		}

		newLogLines, newSamples, _ := logs.ParseAndAnalyzeBuffer(string(buf), 0, linesNewerThan, server)

		var logFile state.LogFile
		logFile.UUID = uuid.NewV4()
		logFile.TmpFile = tmpFile // Pass responsibility to LogFile for cleaning up the temp file
		logFile.OriginalName = *rdsLogFile.LogFileName
		logFile.LogLines = append(logFile.LogLines, newLogLines...)
		samples = append(samples, newSamples...)

		if lastMarker != nil {
			newMarkers[*rdsLogFile.LogFileName] = *lastMarker
		}

		logFiles = append(logFiles, logFile)
	}
	psl.AwsMarkers = newMarkers

	return psl, logFiles, samples, err

ErrorCleanup:
	for _, logFile := range logFiles {
		logFile.Cleanup()
	}

	return server.LogPrevState, nil, nil, err
}

var DescribeDBClustersErrorCache *util.TTLMap = util.NewTTLMap(10 * 60)

func getAwsDbInstanceID(config config.ServerConfig, sess *session.Session) (string, error) {
	identifier := config.AwsDbInstanceID
	// If this is an Aurora cluster endpoint, we need to find the actual instance ID in order to get the logs
	if identifier == "" {
		if config.AwsDbClusterID == "" {
			return "", fmt.Errorf("Neither AWS instance ID or cluster ID are specified - skipping log download")
		}

		// Remember when an Aurora instance find failed previously to avoid failing on the same
		// DescribeDBClusters call again and again. Note that we don't cache successes because
		// we want to react quickly to failover events.
		cachedError := DescribeDBClustersErrorCache.Get(config.AwsDbClusterID)
		if cachedError != "" {
			return "", errors.New(cachedError)
		}

		instance, err := awsutil.FindRdsInstance(config, sess)
		if err != nil {
			err = fmt.Errorf("Error finding instance for cluster ID \"%s\": %s", config.AwsDbClusterID, err)
			DescribeDBClustersErrorCache.Put(config.AwsDbClusterID, err.Error())
			return "", err
		}

		identifier = *instance.DBInstanceIdentifier
	}

	return identifier, nil
}

func downloadRdsLogFilePortion(rdsSvc *rds.RDS, tmpFile *os.File, logger *util.Logger, identifier *string, logFileName *string, lastMarker *string) (newBytesWritten int, newMarker *string, additionalDataPending bool, err error) {
	var resp *rds.DownloadDBLogFilePortionOutput
	resp, err = rdsSvc.DownloadDBLogFilePortion(&rds.DownloadDBLogFilePortionInput{
		DBInstanceIdentifier: identifier,
		LogFileName:          logFileName,
		Marker:               lastMarker, // This is not set for the initial call, so we only get the most recent lines
	})

	if err != nil {
		err = fmt.Errorf("Error downloading logs: %s", err)
		return
	}

	if resp.LogFileData == nil {
		logger.PrintVerbose("Rds/Logs: No log data in response, skipping")
		return
	}

	if len(*resp.LogFileData) > 0 {
		newBytesWritten, err = tmpFile.WriteString(*resp.LogFileData)
		if err != nil {
			err = fmt.Errorf("Error writing to tempfile: %s", err)
			return
		}
	}

	newMarker = resp.Marker
	additionalDataPending = *resp.AdditionalDataPending

	return
}

// Analyze and submit at most the trailing 10 megabytes of the retrieved RDS log file portions
//
// This avoids an OOM in two edge cases:
// 1) When starting the collector, as we always load the last 10,000 lines (which may be very long)
// 2) When extremely large values are output in a single log event (e.g. query parameters in a DETAIL line)
//
// We intentionally throw away data here (and warn the user about it), since the alternative
// is often a collector crash (due to OOM), which would be less desirable.
const maxLogParsingSize = 10 * 1024 * 1024

func readLogFilePortion(tmpFile *os.File, bytesWritten int64, logger *util.Logger) ([]byte, *os.File, error) {
	var err error
	var readSize int64

	exceededMaxParsingSize := bytesWritten > maxLogParsingSize
	if exceededMaxParsingSize {
		logger.PrintWarning("RDS log file portion exceeded more than 10 MB of data in 30 second interval, collecting most recent data only (skipping %d bytes)", bytesWritten-maxLogParsingSize)
		readSize = maxLogParsingSize
	} else {
		readSize = bytesWritten
	}

	// Read the data into memory for analysis
	_, err = tmpFile.Seek(bytesWritten-readSize, io.SeekStart)
	if err != nil {
		return nil, tmpFile, fmt.Errorf("Error seeking tempfile: %s", err)
	}
	buf := make([]byte, readSize)
	_, err = io.ReadFull(tmpFile, buf)
	if err != nil {
		return nil, tmpFile, fmt.Errorf("Error reading %d bytes from tempfile: %s", len(buf), err)
	}

	// If necessary, recreate tempfile with just the data we're analyzing
	// (this supports the later read of the temp file during the log upload)
	if exceededMaxParsingSize {
		truncatedTmpFile, err := ioutil.TempFile("", "")
		if err != nil {
			return nil, tmpFile, fmt.Errorf("Error allocating tempfile for logs: %s", err)
		}

		_, err = truncatedTmpFile.Write(buf)
		if err != nil {
			truncatedTmpFile.Close()
			os.Remove(truncatedTmpFile.Name())
			return nil, tmpFile, fmt.Errorf("Error writing to tempfile: %s", err)
		}

		// We succeeded, so remove the previous file and use the new one going forward
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		tmpFile = truncatedTmpFile
	}

	return buf, tmpFile, nil
}

package rds

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/logs"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	"github.com/pganalyze/collector/util/awsutil"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
)

// DownloadLogFiles - Gets log files for an Amazon RDS instance
func DownloadLogFiles(ctx context.Context, server *state.Server, logger *util.Logger) (state.PersistedLogState, []state.LogFile, []state.PostgresQuerySample, error) {
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
		var content strings.Builder
		var lastMarker *string
		prevMarker, ok := psl.AwsMarkers[*rdsLogFile.LogFileName]
		if ok {
			lastMarker = &prevMarker
		}

		for {
			var fileContent *string
			var newMarker *string
			var additionalDataPending bool
			fileContent, newMarker, additionalDataPending, err = downloadRdsLogFilePortion(rdsSvc, logger, &identifier, rdsLogFile.LogFileName, lastMarker)
			if err != nil {
				return server.LogPrevState, nil, nil, err
			}
			content.WriteString(*fileContent)
			if newMarker != nil {
				lastMarker = newMarker
			}
			if !additionalDataPending {
				break
			}
		}

		stream := bufio.NewReader(strings.NewReader(content.String()))
		newLogLines, newSamples := logs.ParseAndAnalyzeBuffer(stream, linesNewerThan, server)

		var logFile state.LogFile
		logFile, err = state.NewLogFile(*rdsLogFile.LogFileName)
		if err != nil {
			err = fmt.Errorf("error initializing log file: %s", err)
			return server.LogPrevState, nil, nil, err
		}
		logFile.LogLines = append(logFile.LogLines, newLogLines...)
		samples = append(samples, newSamples...)

		if lastMarker != nil {
			newMarkers[*rdsLogFile.LogFileName] = *lastMarker
		}

		logFiles = append(logFiles, logFile)
	}
	psl.AwsMarkers = newMarkers

	return psl, logFiles, samples, err
}

var DescribeDBClustersErrorCache *util.TTLMap = util.NewTTLMap(10 * 60)

// getAwsDbInstanceID - Finds actual instance ID from Aurora cluster endpoint names in order to download logs
func getAwsDbInstanceID(config config.ServerConfig, sess *session.Session) (string, error) {
	if config.AwsDbInstanceID != "" {
		return config.AwsDbInstanceID, nil
	}

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

	return *instance.DBInstanceIdentifier, nil
}

func downloadRdsLogFilePortion(rdsSvc *rds.RDS, logger *util.Logger, identifier *string, logFileName *string, lastMarker *string) (content *string, newMarker *string, additionalDataPending bool, err error) {
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

	content = resp.LogFileData
	newMarker = resp.Marker
	additionalDataPending = *resp.AdditionalDataPending

	return
}

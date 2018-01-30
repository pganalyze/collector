package rds

import (
	"io/ioutil"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/input/system/logs"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	"github.com/pganalyze/collector/util/awsutil"
	uuid "github.com/satori/go.uuid"
)

// GetLogFiles - Gets log files for an Amazon RDS instance
func GetLogFiles(config config.ServerConfig, logger *util.Logger) (result []state.LogFile, samples []state.PostgresQuerySample) {
	sess := awsutil.GetAwsSession(config)

	rdsSvc := rds.New(sess)

	instance, err := awsutil.FindRdsInstance(config, sess)
	if err != nil {
		logger.PrintError("Could not find RDS instance: %s", err)
		return
	}

	// Retrieve all possibly matching logfiles in the last two minutes, assuming
	// a scheduler that runs once a minute
	// TODO: Use prevState here instead to get the last logline we saw
	linesNewerThan := time.Now().Add(-2 * time.Minute)
	lastWritten := linesNewerThan.Unix() * 1000

	params := &rds.DescribeDBLogFilesInput{
		DBInstanceIdentifier: instance.DBInstanceIdentifier,
		FileLastWritten:      &lastWritten,
	}

	resp, err := rdsSvc.DescribeDBLogFiles(params)
	if err != nil {
		logger.PrintError("Could not find RDS log files: %s", err)
		return
	}

	for _, rdsLogFile := range resp.DescribeDBLogFiles {
		var lastMarker *string

		var logFile state.LogFile
		logFile.UUID = uuid.NewV4()
		logFile.TmpFile, err = ioutil.TempFile("", "")
		if err != nil {
			logger.PrintError("Could not allocate tempfile for logs: %s", err)
			break
		}
		logFile.OriginalName = *rdsLogFile.LogFileName
		currentByteStart := int64(0)

		for {
			resp, err := rdsSvc.DownloadDBLogFilePortion(&rds.DownloadDBLogFilePortionInput{
				DBInstanceIdentifier: instance.DBInstanceIdentifier,
				LogFileName:          rdsLogFile.LogFileName,
				Marker:               lastMarker,
				NumberOfLines:        aws.Int64(100), // TODO: Temporary to fix problems
			})

			if err != nil {
				// TODO: Check for unauthorized error:
				// Error: AccessDenied: User: arn:aws:iam::XXX:user/pganalyze_collector is not authorized to perform: rds:DownloadDBLogFilePortion on resource: arn:aws:rds:us-east-1:XXX:db:XXX
				// status code: 403, request id: XXX
				logger.PrintError("%s", err)
				return
			}

			if resp.LogFileData == nil {
				logger.PrintVerbose("No log data in response, skipping")
				break
			}

			_, err = logFile.TmpFile.WriteString(*resp.LogFileData)
			if err != nil {
				logger.PrintError("%s", err)
				break
			}

			var newLogLines []state.LogLine
			var newSamples []state.PostgresQuerySample
			newLogLines, newSamples, currentByteStart = logs.ParseAndAnalyzeBuffer(*resp.LogFileData, currentByteStart, linesNewerThan)
			logFile.LogLines = append(logFile.LogLines, newLogLines...)
			samples = append(samples, newSamples...)

			lastMarker = resp.Marker
			if !*resp.AdditionalDataPending {
				break
			}
		}

		result = append(result, logFile)
	}

	return
}

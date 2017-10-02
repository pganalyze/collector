package rds

import (
	"bufio"
	"io"
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/input/system/logs"
	"github.com/pganalyze/collector/output/pganalyze_collector"
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

			var logLines []state.LogLine

			_, err = logFile.TmpFile.WriteString(*resp.LogFileData)
			if err != nil {
				logger.PrintError("%s", err)
				break
			}

			reader := bufio.NewReader(strings.NewReader(*resp.LogFileData))
			for {
				line, err := reader.ReadString('\n')
				if err == io.EOF {
					break
				}

				byteStart := currentByteStart
				currentByteStart += int64(len(line))

				if err != nil {
					logger.PrintError("%s", err)
					break
				}

				var logLine state.LogLine

				// log_line_prefix is always "%t:%r:%u@%d:[%p]:" on RDS
				parts := strings.SplitN(line, ":", 8)
				if len(parts) != 8 {
					if len(logLines) > 0 {
						logLines[len(logLines)-1].Content += line
						logLines[len(logLines)-1].ByteEnd += int64(len(line))
					}
					continue
				}

				timestamp, err := time.Parse("2006-01-02 15:04:05 MST", parts[0]+":"+parts[1]+":"+parts[2])
				if err != nil {
					if len(logLines) > 0 {
						logLines[len(logLines)-1].Content += line
						logLines[len(logLines)-1].ByteEnd += int64(len(line))
					}
					continue
				}

				// Ignore loglines which are outside our time window
				if timestamp.Before(linesNewerThan) {
					continue
				}

				userDbParts := strings.SplitN(parts[4], "@", 2)
				if len(userDbParts) == 2 {
					logLine.Username = userDbParts[0]
					logLine.Database = userDbParts[1]
				}
				if logLine.Username == "[unknown]" {
					logLine.Username = ""
				}
				if logLine.Database == "[unknown]" {
					logLine.Database = ""
				}

				logLine.OccurredAt = timestamp
				backendPid, _ := strconv.Atoi(parts[5][1 : len(parts[5])-1])
				logLine.BackendPid = int32(backendPid)
				logLine.LogLevel = pganalyze_collector.LogLineInformation_LogLevel(pganalyze_collector.LogLineInformation_LogLevel_value[parts[6]])
				logLine.Content = strings.TrimLeft(parts[7], " ")

				logLine.ByteStart = byteStart
				logLine.ByteContentStart = byteStart + int64(len(line)-len(logLine.Content))
				logLine.ByteEnd = byteStart + int64(len(line)) - 1

				// Generate unique ID that can be used to reference this line
				logLine.UUID = uuid.NewV4()

				logLines = append(logLines, logLine)
			}

			newLogLines, newSamples := logs.AnalyzeLogLines(logLines)
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

package logs

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/lfittl/pganalyze-collector-next/config"
	"github.com/lfittl/pganalyze-collector-next/util"
)

// http://docs.aws.amazon.com/AmazonRDS/latest/APIReference//API_DescribeDBLogFiles.html
// http://docs.aws.amazon.com/AmazonRDS/latest/APIReference//API_DownloadDBLogFilePortion.html
// Retain the marker across runs to only download new data

type rdsLogLine struct {
	timestamp           time.Time
	clientAndPort       string
	usernameAndDatabase string
	backendPid          int
	logLevel            string
	content             string
}

// GetFromAmazonRds - Gets log lines for an Amazon RDS instance
func getFromAmazonRds(config config.Config) (lines []Line) {
	// Get interesting files (last written to in the last 10 minutes)
	// Remember markers for each file

	creds := credentials.NewStaticCredentials(config.AwsAccessKeyId, config.AwsSecretAccessKey, "")

	sess := session.New(&aws.Config{Credentials: creds, Region: aws.String(config.AwsRegion)})
	//sess.Handlers.Send.PushFront(func(r *request.Request) {
	// Log every request made and its payload
	//  fmt.Printf("Request: %s/%s, Payload: %s\n", r.ClientInfo.ServiceName, r.Operation, r.Params)
	//})

	rdsSvc := rds.New(sess)

	instance, err := util.FindRdsInstance(config, sess)

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Retrieve all possibly matching logfiles in the last 10 minutes
	linesNewerThan := time.Now().Add(-10 * time.Minute)
	lastWritten := linesNewerThan.Unix() * 1000

	params := &rds.DescribeDBLogFilesInput{
		DBInstanceIdentifier: instance.DBInstanceIdentifier,
		FileLastWritten:      &lastWritten,
	}

	resp, err := rdsSvc.DescribeDBLogFiles(params)

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	for _, logFile := range resp.DescribeDBLogFiles {
		params := &rds.DownloadDBLogFilePortionInput{
			DBInstanceIdentifier: instance.DBInstanceIdentifier,
			LogFileName:          logFile.LogFileName,
			Marker:               aws.String("0"),
		}

		resp, err := rdsSvc.DownloadDBLogFilePortion(params)

		if err != nil {
			// TODO: Check for unauthorized error:
			// Error: AccessDenied: User: arn:aws:iam::793741702295:user/pganalyze_collector is not authorized to perform: rds:DownloadDBLogFilePortion on resource: arn:aws:rds:us-east-1:793741702295:db:pganalyze-production
			// status code: 403, request id: 8e8cb4e8-91a7-11e5-a24c-2bc3c220de32
			fmt.Printf("Error: %v\n", err)
			return
		}

		var logLines []rdsLogLine

		var incompleteLine = false

		reader := bufio.NewReader(strings.NewReader(*resp.LogFileData))
		for {
			line, isPrefix, err := reader.ReadLine()
			if err == io.EOF {
				break
			}

			if err != nil {
				fmt.Printf("Error: %v\n", err)
				break
			}

			if incompleteLine {
				if len(logLines) > 0 {
					logLines[len(logLines)-1].content += string(line)
				}
				incompleteLine = isPrefix
				continue
			}

			incompleteLine = isPrefix

			var logLine rdsLogLine

			parts := strings.SplitN(string(line), ":", 8)
			if len(parts) != 8 {
				if len(logLines) > 0 {
					logLines[len(logLines)-1].content += string(line)
				}
				continue
			}

			logLine.timestamp, err = time.Parse("2006-01-02 15:04:05 MST", parts[0]+":"+parts[1]+":"+parts[2])

			if err != nil {
				if len(logLines) > 0 {
					logLines[len(logLines)-1].content += string(line)
				}
				continue
			}

			logLine.clientAndPort = parts[3]
			logLine.usernameAndDatabase = parts[4]
			logLine.backendPid, _ = strconv.Atoi(parts[5][1 : len(parts[5])-1])
			logLine.logLevel = parts[6]
			logLine.content = strings.TrimLeft(parts[7], " ")

			logLines = append(logLines, logLine)
		}

		// Split log lines by backend to ensure we have the right context
		backendLogLines := make(map[int][]rdsLogLine)

		for _, logLine := range logLines {
			// Ignore loglines which are outside our time window
			if logLine.timestamp.Before(linesNewerThan) {
				continue
			}

			backendLogLines[logLine.backendPid] = append(backendLogLines[logLine.backendPid], logLine)
		}

		for _, logLines := range backendLogLines {
			for _, logLine := range logLines {
				if strings.HasPrefix(logLine.content, "duration: ") {
					// Do not include in final output, consider for auto explain
					break
				}

				fmt.Printf("%s\n", logLine.logLevel)
				fmt.Printf("%s\n", logLine.content)
				// Look ahead to include possibly useful information
				// DETAIL/STATEMENT (or both)
			}
		}

		// TODO: Handle resp.AdditionalDataPending / resp.Marker
	}

	return
}

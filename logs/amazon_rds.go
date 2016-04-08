package logs

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/explain"
	"github.com/pganalyze/collector/util"
)

// http://docs.aws.amazon.com/AmazonRDS/latest/APIReference//API_DescribeDBLogFiles.html
// http://docs.aws.amazon.com/AmazonRDS/latest/APIReference//API_DownloadDBLogFilePortion.html
// Retain the marker across runs to only download new data

// GetFromAmazonRds - Gets log lines for an Amazon RDS instance
func getFromAmazonRds(config config.DatabaseConfig) (result []Line, explains []explain.ExplainInput) {
	// Get interesting files (last written to in the last 10 minutes)
	// Remember markers for each file

	sess := util.GetAwsSession(config)

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
			// Error: AccessDenied: User: arn:aws:iam::XXX:user/pganalyze_collector is not authorized to perform: rds:DownloadDBLogFilePortion on resource: arn:aws:rds:us-east-1:XXX:db:XXX
			// status code: 403, request id: XXX
			fmt.Printf("Error: %v\n", err)
			return
		}

		var logLines []Line

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
					logLines[len(logLines)-1].Content += string(line)
				}
				incompleteLine = isPrefix
				continue
			}

			incompleteLine = isPrefix

			// TODO: What we should actually do is look at log_line_prefix and find the relevant
			// parts using that - otherwise we break in non-default RDS setups

			var logLine Line

			parts := strings.SplitN(string(line), ":", 8)
			if len(parts) != 8 {
				if len(logLines) > 0 {
					logLines[len(logLines)-1].Content += string(line)
				}
				continue
			}

			timestamp, err := time.Parse("2006-01-02 15:04:05 MST", parts[0]+":"+parts[1]+":"+parts[2])
			if err != nil {
				if len(logLines) > 0 {
					logLines[len(logLines)-1].Content += string(line)
				}
				continue
			}

			logLine.OccurredAt = util.TimestampFrom(timestamp)
			logLine.ClientIP = regexp.MustCompile(`[\d.]+`).FindString(parts[3])
			//logLine.usernameAndDatabase = parts[4] // TODO: We should probably filter out other databases (in our current monitoring model)
			logLine.BackendPid, _ = strconv.Atoi(parts[5][1 : len(parts[5])-1])
			logLine.LogLevel = parts[6]
			logLine.Content = strings.TrimLeft(parts[7], " ")

			logLines = append(logLines, logLine)
		}

		// Split log lines by backend to ensure we have the right context
		backendLogLines := make(map[int][]Line)

		for _, logLine := range logLines {
			// Ignore loglines which are outside our time window
			if logLine.OccurredAt.Ptr().Before(linesNewerThan) {
				continue
			}

			backendLogLines[logLine.BackendPid] = append(backendLogLines[logLine.BackendPid], logLine)
		}

		skipLines := 0

		for _, logLines := range backendLogLines {
			for idx, logLine := range logLines {
				if skipLines > 0 {
					skipLines--
					continue
				}

				// Look up to 2 lines in the future to find context for this line
				lowerBound := int(math.Min(float64(len(logLines)), float64(idx+1)))
				upperBound := int(math.Min(float64(len(logLines)), float64(idx+3)))
				for _, futureLine := range logLines[lowerBound:upperBound] {
					if futureLine.LogLevel == "STATEMENT" || futureLine.LogLevel == "DETAIL" {
						logLine.AdditionalLines = append(logLine.AdditionalLines, futureLine)
						skipLines++
					} else {
						break
					}
				}

				if strings.HasPrefix(logLine.Content, "duration: ") {
					parts := regexp.MustCompile(`duration: ([\d\.]+) ms([^:]+): (.+)`).FindStringSubmatch(logLine.Content)

					if len(parts) != 4 || strings.Contains(parts[2], "bind") || strings.Contains(parts[2], "parse") {
						fmt.Printf("ERR")
						continue
					}

					runtime, _ := strconv.ParseFloat(parts[1], 64)
					explains = append(explains, explain.ExplainInput{
						OccurredAt: logLine.OccurredAt,
						Query:      parts[3],
						Runtime:    runtime,
					})

					continue
				}

				// Need to clean STATEMENT
				// Need to remove DETAIL "parameters: "

				result = append(result, logLine)
			}
		}

		// TODO: Handle resp.AdditionalDataPending / resp.Marker
	}

	return
}

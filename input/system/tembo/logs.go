package tembo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

type StreamResult struct {
	Streams []StreamSet `json:"streams"`
}

type StreamSet struct {
	Stream StreamMetadata `json:"stream"`
	Values [][]string     `json:"values"`
}

type StreamMetadata struct {
	App                 string `json:"app"`
	Container           string `json:"container"`
	Pod                 string `json:"pod"`
	Stream              string `json:"stream"`
	TemboInstanceID     string `json:"tembo_instance_id"`
	TemboOrganizationID string `json:"tembo_organization_id"`
}

type JSONLogEvent struct {
	Record map[string]string `json:"record"`
}

func connectWebsocket(ctx context.Context, logger *util.Logger, server *state.Server) (*websocket.Conn, context.CancelFunc, error) {
	connCtx, cancelConn := context.WithCancel(ctx)

	// Construct query for Tembo Logs API
	// Note this does not yet handle multiple replicas in cases like HA
	query := "{tembo_instance_id=\"" + server.Config.TemboInstanceID + "\", pod=\"" + server.Config.TemboNamespace + "-1" + "\"}"

	// URI encode query
	encodedQuery := url.QueryEscape(query)

	// Construct URL for Tembo Logs API
	logsAPIURL := server.Config.TemboLogsAPIURL
	websocketUrl := "wss://" + logsAPIURL + "/loki/api/v1/tail?query=" + encodedQuery

	// Set headers
	headers := make(map[string][]string)
	headers["Authorization"] = []string{"Bearer " + server.Config.TemboAPIToken}
	headers["X-Scope-OrgId"] = []string{server.Config.TemboOrgID}

	conn, response, err := websocket.DefaultDialer.DialContext(connCtx, websocketUrl, headers)
	if err != nil && response != nil {
		cancelConn()
		return nil, nil, fmt.Errorf("%s (status %d)", err, response.StatusCode)
	} else if err != nil {
		cancelConn()
		return nil, nil, err
	}
	go func() {
		for {
			select {
			case <-connCtx.Done():
				err := conn.Close()
				if err != nil {
					logger.PrintError("Error closing websocket: %s", err)
					return
				}
				return
			}
		}
	}()
	return conn, cancelConn, nil
}

// SetupWebsocketHandlerLogs - Sets up a websocket handler for Tembo logs
func SetupWebsocketHandlerLogs(ctx context.Context, wg *sync.WaitGroup, logger *util.Logger, servers []*state.Server, globalCollectionOpts state.CollectionOpts, parsedLogStream chan state.ParsedLogStreamItem) {
	for _, server := range servers {
		prefixedLogger := logger.WithPrefix(server.Config.SectionName)

		if server.Config.TemboLogsAPIURL != "" {
			setupWebsocketForServer(ctx, wg, globalCollectionOpts, prefixedLogger, server, parsedLogStream)
		}
	}
}

func setupWebsocketForServer(ctx context.Context, wg *sync.WaitGroup, globalCollectionOpts state.CollectionOpts, logger *util.Logger, server *state.Server, parsedLogStream chan state.ParsedLogStreamItem) {
	// Only ingest log lines that were written in the last minute before startup
	linesNewerThan := time.Now().Add(-1 * time.Minute)
	logParser := server.GetLogParser()

	wg.Add(1)
	go func() {
		defer wg.Done()
		var conn *websocket.Conn
		var cancelConn context.CancelFunc
		for {
			var line []byte
			var err error

			if conn == nil {
				conn, cancelConn, err = connectWebsocket(ctx, logger, server)
				if err != nil {
					if globalCollectionOpts.TestRun {
						logger.PrintError("Error connecting to Tembo logs websocket: %s", err)
						return
					}
					if strings.Contains(err.Error(), "operation was canceled") {
						// We closed the connection since the context was cancelled
						return
					}
					logger.PrintError("Error connecting to Tembo logs websocket, sleeping 10 seconds: %s", err)
					time.Sleep(10 * time.Second)
					conn = nil
					continue
				}
			}

			// Attempt to read message from websocket and retry if it fails
			_, line, err = conn.ReadMessage()
			if err != nil {
				if strings.Contains(err.Error(), "use of closed network connection") {
					// We closed the connection since the context was cancelled
					return
				}
				if !(websocket.IsCloseError(err, websocket.CloseInternalServerErr) && strings.Contains(err.Error(), "reached tail max duration limit")) {
					logger.PrintError("Error reading from websocket, sleeping 1 second before reconnecting: %v", err)
					time.Sleep(1 * time.Second)
				}

				cancelConn()
				conn = nil
				continue
			}

			var result StreamResult
			err = json.Unmarshal(line, &result)
			if err != nil {
				logger.PrintError("Error unmarshalling JSON: %s", err)
			}
			for _, stream := range result.Streams {
				for _, values := range stream.Values {
					logLine, detailLine := logLineFromJsonlog(values[1], logParser, logger)
					// Ignore loglines which are outside our time window
					if !logLine.OccurredAt.IsZero() && logLine.OccurredAt.Before(linesNewerThan) {
						continue
					}
					parsedLogStream <- state.ParsedLogStreamItem{Identifier: server.Config.Identifier, LogLine: logLine}
					if detailLine != nil {
						parsedLogStream <- state.ParsedLogStreamItem{Identifier: server.Config.Identifier, LogLine: *detailLine}
					}
				}
			}
		}
	}()
}

func logLineFromJsonlog(recordIn string, logParser state.LogParser, logger *util.Logger) (state.LogLine, *state.LogLine) {
	var event JSONLogEvent
	var logLine state.LogLine

	err := json.Unmarshal([]byte(recordIn), &event)
	if err != nil {
		logger.PrintError("Error unmarshalling JSON: %s", err)
		return logLine, nil
	}

	// If a DETAIL line is set, we need to create an additional log line
	detailLineContent := ""

	for key, value := range event.Record {
		if key == "log_time" {
			logLine.OccurredAt = logParser.GetOccurredAt(value)
		}
		if key == "user_name" {
			logLine.Username = value
		}
		if key == "database_name" {
			logLine.Database = value
		}
		if key == "process_id" {
			backendPid, _ := strconv.ParseInt(value, 10, 32)
			logLine.BackendPid = int32(backendPid)
		}
		if key == "application_name" {
			logLine.Application = value
		}
		if key == "session_line_num" {
			logLineNumber, _ := strconv.ParseInt(value, 10, 32)
			logLine.LogLineNumber = int32(logLineNumber)
		}
		if key == "message" {
			logLine.Content = value
		}
		if key == "detail" {
			detailLineContent = value
		}
		if key == "error_severity" {
			logLine.LogLevel = pganalyze_collector.LogLineInformation_LogLevel(pganalyze_collector.LogLineInformation_LogLevel_value[value])
		}
	}
	if detailLineContent != "" {
		detailLine := logLine
		detailLine.Content = detailLineContent
		detailLine.LogLevel = pganalyze_collector.LogLineInformation_DETAIL
		return logLine, &detailLine
	}

	return logLine, nil
}

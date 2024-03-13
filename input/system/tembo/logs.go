package tembo

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pganalyze/collector/logs"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	uuid "github.com/satori/go.uuid"
)

type Result struct {
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
	query := "{tembo_instance_id=\"" + server.Config.TemboInstanceID + "\"}"

	// URI encode query
	encodedQuery := url.QueryEscape(query)

	// Construct URL for Tembo Logs API
	websocketUrl := "wss://api.data-1.use1.tembo.io/loki/api/v1/tail?query=" + encodedQuery

	// Set headers
	headers := make(map[string][]string)
	headers["Authorization"] = []string{"Bearer " + server.Config.TemboAPIToken}
	headers["X-Scope-OrgId"] = []string{server.Config.TemboOrgID}

	conn, response, err := websocket.DefaultDialer.DialContext(connCtx, websocketUrl, headers)
	if err != nil && response != nil {
		logger.PrintError("Error connecting to Tembo logs websocket: %s (status %d)", err, response.StatusCode)
		return nil, nil, err // Do we want this to return here? (or keep trying?)
	} else if err != nil {
		logger.PrintError("Error connecting to Tembo logs websocket: %s", err)
		return nil, nil, err // Do we want this to return here? (or keep trying?)
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
func SetupWebsocketHandlerLogs(ctx context.Context, wg *sync.WaitGroup, logger *util.Logger, server *state.Server, parsedLogStream chan state.ParsedLogStreamItem) {
	// Only ingest log lines that were written in the last minute before startup
	linesNewerThan := time.Now().Add(-1 * time.Minute)
	tz := server.GetLogTimezone()

	// Connect to websocket
	conn, cancelConn, err := connectWebsocket(ctx, logger, server)
	if err != nil {
		// Should we retry if we get an error the first time we're connecting?
		return
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			// Attempt to read message from websocket and retry if it fails
			var line []byte

			_, line, err = conn.ReadMessage()
			if err != nil {
				if !(websocket.IsCloseError(err, websocket.CloseInternalServerErr) && strings.Contains(err.Error(), "reached tail max duration limit")) {
					logger.PrintError("Error reading from websocket attempt, sleeping 1 second: %v", err)
					time.Sleep(1 * time.Second)
				}

				// Reconnect
				cancelConn()
				conn, cancelConn, err = connectWebsocket(ctx, logger, server)
				if err != nil {
					// Should we wait longer and keep retrying?
					return
				}
				continue
			}

			var result Result
			err = json.Unmarshal(line, &result)
			if err != nil {
				logger.PrintError("Error unmarshalling JSON: %s", err)
			}
			for _, stream := range result.Streams {
				for _, values := range stream.Values {
					logLine, detailLine := logLineFromJsonlog(values[1], tz, logger)
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

func logLineFromJsonlog(recordIn string, tz *time.Location, logger *util.Logger) (state.LogLine, *state.LogLine) {
	var event JSONLogEvent
	var logLine state.LogLine
	logLine.CollectedAt = time.Now()
	logLine.UUID = uuid.NewV4()

	err := json.Unmarshal([]byte(recordIn), &event)
	if err != nil {
		logger.PrintError("Error unmarshalling JSON: %s", err)
		return logLine, nil
	}

	// If a DETAIL line is set, we need to create an additional log line
	detailLineContent := ""

	for key, value := range event.Record {
		if key == "log_time" {
			logLine.OccurredAt = logs.GetOccurredAt(value, tz, false)
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

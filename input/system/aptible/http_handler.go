package aptible

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/pganalyze/collector/logs"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

type AptibleLog struct {
	Time     string `json:"time"`
	Source   string `json:"source"`
	Database string `json:"database"`
	Offset   int    `json:"offset"`
	Log      string `json:"log"`
}

func SetupHttpHandlerLogs(ctx context.Context, wg *sync.WaitGroup, globalCollectionOpts state.CollectionOpts, logger *util.Logger, servers []*state.Server, parsedLogStream chan state.ParsedLogStreamItem) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			resp := make(map[string]string)
			resp["message"] = "Status OK"
			jsonResp, err := json.Marshal(resp)
			if err != nil {
				log.Fatalf("Error happened in JSON marshal. Err: %s\n", err)
			}
			w.Write(jsonResp)

			switch r.Method {
			case http.MethodPost:
				decoder := json.NewDecoder(r.Body)

				var logMessage AptibleLog
				err := decoder.Decode(&logMessage)
				if err != nil {
					log.Fatalf("WARNING: Log message not parsed: %s\n", err)
					return
				}

				if logMessage.Source != "database" || logMessage.Database != "healthie-staging-14" {
					return
				}
				logLine, ok := logs.ParseLogLineWithPrefix(logs.LogPrefixCustom3, logMessage.Log+"\n", nil)
				if ok {
					log.Fatalf("Log line parsed: %v\n", logLine)
					occurredAt, err := time.Parse(time.RFC3339, logMessage.Time)
					if err != nil {
						log.Fatalf("Error happened time parsing. Err: %s", err)
					}
					logLine.OccurredAt = occurredAt
					logger.PrintVerbose("Log line content")
					logger.PrintVerbose(logLine.Content)
					logLine.LogLevel = pganalyze_collector.LogLineInformation_CONTEXT
					for _, server := range servers {
						if server.Config.SectionName == "healthie-staging-14" {
							logger.PrintVerbose("Server match, sending logs")
							parsedLogStream <- state.ParsedLogStreamItem{Identifier: server.Config.Identifier, LogLine: logLine}
						}
					}
				}
			}
		})
		http.ListenAndServe(":"+os.Getenv("PORT"), nil)
	}()
}

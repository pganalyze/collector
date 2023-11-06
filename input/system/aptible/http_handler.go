package aptible

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/pganalyze/collector/logs"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

type AptibleLog struct {
	Timestamp string `json:"@timestamp"`
	Log       string `json:"log"`
	Host      string `json:"host"`
	Service   string `json:"service"`
	App       string `json:"app"`
	AppId     string `json:"app_id"`
	Source    string `json:"source"`
	Container string `json:"container"`
}

func SetupHttpHandlerLogs(ctx context.Context, wg *sync.WaitGroup, globalCollectionOpts state.CollectionOpts, logger *util.Logger, servers []*state.Server, parsedLogStream chan state.ParsedLogStreamItem) {
	go func() {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				w.WriteHeader(http.StatusOK)
				w.Header().Set("Content-Type", "application/json")
				resp := make(map[string]string)
				resp["message"] = "Status OK"
				jsonResp, err := json.Marshal(resp)
				if err != nil {
					log.Fatalf("Error happened in JSON marshal. Err: %s", err)
				}
				w.Write(jsonResp)
				return
			case http.MethodPost:
				io.Copy(os.Stderr, r.Body)
				decoder := json.NewDecoder(r.Body)

				var logMessage AptibleLog
				err := decoder.Decode(&logMessage)
				if err != nil {
					log.Fatalln("WARNING: Log message not parsed")
				} else {
					logLine, _ := logs.ParseLogLineWithPrefix("", logMessage.Log+"\n", nil)
					//logLine.OccurredAt = log.Timestamp
					//logLine.LogLineNumber = int32(logLineNumber)
					//logLine.LogLineNumberChunk = int32(logLineNumberChunk)
					// somehow map back to a server identifier, which is the app identifier
					// Identifier is where it's going. LogLine is where it came from
					//parsedLogStream <- state.ParsedLogStreamItem{Identifier: server.Config.Identifier, LogLine: log.Log}
					fmt.Println(logLine)
				}
			}
		})
		http.ListenAndServe(":"+os.Getenv("PORT"), nil)
	}()
}

package aptible

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
		http.HandleFunc("/", util.HttpRedirectToApp)
		http.HandleFunc("/logs/", func(w http.ResponseWriter, r *http.Request) {
			io.Copy(os.Stdout, r.Body)
			decoder := json.NewDecoder(r.Body)

			var log AptibleLog
			err := decoder.Decode(&log)
			if err != nil {
				fmt.Printf("WARNING: Log message not parsed\n")
			} else {
				logLine, _ := logs.ParseLogLineWithPrefix("", log.Log+"\n", nil)
				//logLine.OccurredAt = log.Timestamp
				//logLine.LogLineNumber = int32(logLineNumber)
				//logLine.LogLineNumberChunk = int32(logLineNumberChunk)
				// somehow map back to a server identifier, which is the app identifier
				// Identifier is where it's going. LogLine is where it came from
				//parsedLogStream <- state.ParsedLogStreamItem{Identifier: server.Config.Identifier, LogLine: log.Log}
				fmt.Println(logLine)
			}
		})
		http.ListenAndServe(":"+os.Getenv("PORT"), nil)
	}()
}

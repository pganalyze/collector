package aptible

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"sync"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func httpOK(w http.ResponseWriter) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	resp := make(map[string]string)
	resp["message"] = "Status OK"
	jsonResp, _ := json.Marshal(resp)
	w.Write(jsonResp)
}

func SetupHttpHandler(ctx context.Context, wg *sync.WaitGroup, globalCollectionOpts state.CollectionOpts, logger *util.Logger, servers []*state.Server, parsedLogStream chan state.ParsedLogStreamItem) {
	logger.PrintVerbose(("Setting up aptible http handler"))

	wg.Add(1)
	server := http.NewServeMux()

	server.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		logger.PrintVerbose("Aptible http root handler: %s", r.URL.Path)
		httpOK(w)
	})

	server.HandleFunc("/logs", func(w http.ResponseWriter, r *http.Request) {
		logger.PrintVerbose("Aptible http log handler: %s", r.URL.Path)
		httpOK(w)

		switch r.Method {
		case http.MethodPost:
			decoder := json.NewDecoder(r.Body)

			var logMessage AptibleLog
			err := decoder.Decode(&logMessage)
			if err != nil {
				logger.PrintError("WARNING: Log message not parsed: %s\n", err)
			} else {
				HandleLogMessage(&logMessage, logger, servers, parsedLogStream)
			}

		}
	})

	// Mimic influxdb v2
	server.HandleFunc("/api/v2/write", func(w http.ResponseWriter, r *http.Request) {
		logger.PrintVerbose("Aptible http metric handler: %s", r.URL.Path)
		httpOK(w)

		switch r.Method {
		case http.MethodPost:
			decoder := json.NewDecoder(r.Body)

			var sample AptibleMetric
			err := decoder.Decode(&sample)
			if err != nil {
				logger.PrintWarning("WARNING: Metric message not parsed: %s\n", err)
			} else {
				HandleMetricMessage(ctx, &sample, globalCollectionOpts, logger, servers)
			}
		}
	})

	go func() {
		defer wg.Done()
		http.ListenAndServe(":"+os.Getenv("PORT"), server)
	}()
}

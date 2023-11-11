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

func httpOK(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	resp := make(map[string]string)
	resp["message"] = "Status OK"
	jsonResp, _ := json.Marshal(resp)
	w.Write(jsonResp)
}

func SetupHttpHandler(ctx context.Context, wg *sync.WaitGroup, globalCollectionOpts state.CollectionOpts, logger *util.Logger, servers []*state.Server, parsedLogStream chan state.ParsedLogStreamItem) {
	wg.Add(1)

	http.HandleFunc("/", httpOK)

	http.HandleFunc("/logs", func(w http.ResponseWriter, r *http.Request) {
		httpOK(w, r)

		switch r.Method {
		case http.MethodPost:
			decoder := json.NewDecoder(r.Body)

			var logMessage AptibleLog
			err := decoder.Decode(&logMessage)
			if err != nil {
				logger.PrintError("WARNING: Log message not parsed: %s\n", err)
				return
			}

			HandleLogMessage(&logMessage, logger, servers, parsedLogStream)
		}
	})

	// Mimic influxdb v2
	http.HandleFunc("/api/v2/write", func(w http.ResponseWriter, r *http.Request) {
		httpOK(w, r)

		switch r.Method {
		case http.MethodPost:
			decoder := json.NewDecoder(r.Body)

			var sample AptibleMetric
			err := decoder.Decode(&sample)
			if err != nil {
				logger.PrintWarning("WARNING: Metric message not parsed: %s\n", err)
				return
			}

			HandleMetricMessage(ctx, &sample, globalCollectionOpts, logger, servers)
		}
	})

	go func() {
		defer wg.Done()
		http.ListenAndServe(":"+os.Getenv("PORT"), nil)
	}()
}

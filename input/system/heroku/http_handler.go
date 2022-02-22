package heroku

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/bmizerany/lpx"
	"github.com/pganalyze/collector/logs"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func SetupHttpHandlerLogs(ctx context.Context, wg *sync.WaitGroup, globalCollectionOpts state.CollectionOpts, logger *util.Logger, servers []*state.Server, parsedLogStream chan state.ParsedLogStreamItem) {
	herokuLogStream := make(chan HerokuLogStreamItem, state.LogStreamBufferLen)
	setupLogTransformer(ctx, wg, servers, herokuLogStream, parsedLogStream, globalCollectionOpts, logger)

	go func() {
		http.HandleFunc("/", util.HttpRedirectToApp)
		http.HandleFunc("/logs/", func(w http.ResponseWriter, r *http.Request) {
			lp := lpx.NewReader(bufio.NewReader(r.Body))
			for lp.Next() {
				procID := string(lp.Header().Procid)
				if procID == "heroku-postgres" || strings.HasPrefix(procID, "postgres.") {
					select {
					case herokuLogStream <- HerokuLogStreamItem{Header: *lp.Header(), Content: lp.Bytes(), Path: r.URL.Path}:
						// Handed over successfully
					default:
						fmt.Printf("WARNING: Channel buffer exceeded, skipping message\n")
					}
				}
			}
		})
		http.ListenAndServe(":"+os.Getenv("PORT"), nil)
	}()

	for _, server := range servers {
		logs.EmitTestLogMsg(server, globalCollectionOpts, logger)
	}
}

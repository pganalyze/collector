package heroku

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func SetupHttpHandlerLogs(ctx context.Context, wg *sync.WaitGroup, globalCollectionOpts state.CollectionOpts, logger *util.Logger, servers []*state.Server, parsedLogStream chan state.ParsedLogStreamItem) {
	herokuLogStream := make(chan HttpSyslogMessage, state.LogStreamBufferLen)
	setupLogTransformer(ctx, wg, servers, herokuLogStream, parsedLogStream, globalCollectionOpts, logger)

	go func() {
		http.HandleFunc("/", util.HttpRedirectToApp)
		http.HandleFunc("/logs/", func(w http.ResponseWriter, r *http.Request) {
			for _, item := range ReadHerokuPostgresSyslogMessages(r.Body) {
				item.Path = r.URL.Path
				select {
				case herokuLogStream <- item:
					// Handed over successfully
				default:
					fmt.Printf("WARNING: Channel buffer exceeded, skipping message\n")
				}
			}
		})
		http.ListenAndServe(":"+os.Getenv("PORT"), nil)
	}()
}

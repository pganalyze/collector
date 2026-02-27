package heroku

import (
	"context"
	"net/http"
	"os"
	"sync"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func SetupHttpHandlerLogs(ctx context.Context, wg *sync.WaitGroup, opts state.CollectionOpts, logger *util.Logger, servers []*state.Server, parsedLogStream chan state.ParsedLogStreamItem) {
	herokuLogStream := make(chan HttpSyslogMessage, state.LogStreamBufferLen)
	setupLogTransformer(ctx, wg, servers, herokuLogStream, parsedLogStream, opts, logger)

	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/", util.HttpRedirectToApp)
	serveMux.HandleFunc("/logs/", func(w http.ResponseWriter, r *http.Request) {
		for _, item := range ReadHerokuPostgresSyslogMessages(r.Body) {
			item.Path = r.URL.Path
			select {
			case herokuLogStream <- item:
				// Handed over successfully
			default:
				logger.PrintInfo("WARNING: Channel buffer exceeded, skipping message\n")
			}
		}
	})

	util.GoServeHTTP(ctx, logger, ":"+os.Getenv("PORT"), serveMux)
}

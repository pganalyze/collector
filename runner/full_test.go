package runner

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

// Panics once collection first waits on the context (when connecting to the
// database), standing in for any panic during a full snapshot
type panickingContext struct{ context.Context }

func (panickingContext) Done() <-chan struct{} { panic("test panic") }

func TestProcessServerLogsPanicStackTrace(t *testing.T) {
	for _, withSentry := range []bool{false, true} {
		name := "without Sentry"
		if withSentry {
			name = "with Sentry"
		}
		t.Run(name, func(t *testing.T) {
			var sentryEvents atomic.Int32
			sentry := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				sentryEvents.Add(1)
			}))
			defer sentry.Close()

			server := state.MakeServer(config.ServerConfig{DbHost: "localhost", DbName: "postgres"}, false)
			if withSentry {
				dsn := strings.Replace(sentry.URL, "http://", "http://public:secret@", 1) + "/1"
				server.Grant.Store(&state.Grant{Config: pganalyze_collector.ServerMessage_Config{SentryDsn: dsn}})
			}
			var output bytes.Buffer
			logger := &util.Logger{Destination: log.New(&output, "", 0)}
			opts := state.CollectionOpts{ForceEmptyGrant: true}

			_, _, err := processServer(panickingContext{context.Background()}, server, opts, logger)

			if err == nil || err.Error() != "test panic" {
				t.Errorf("expected error \"test panic\", got %v", err)
			}
			if !strings.Contains(output.String(), "runner.panickingContext.Done") {
				t.Errorf("expected logged stack trace to include the panicking function, got:\n%s", output.String())
			}
			if withSentry && sentryEvents.Load() != 1 {
				t.Errorf("expected 1 Sentry event, got %d", sentryEvents.Load())
			}
		})
	}
}

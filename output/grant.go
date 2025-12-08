package output

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

// EnsureGrant - Ensures the server has a valid grant stored from either WebSocket or HTTP-based grant API
func EnsureGrant(ctx context.Context, server *state.Server, opts state.CollectionOpts, logger *util.Logger, refetchAlways bool) error {
	if opts.ForceEmptyGrant {
		return nil
	}

	// Accept existing grant data if we're using a WebSocket connection and it's
	// up and running and received at least one config message, or if the caller
	// allows reusing the last S3-based grant without a refetch, and its still valid.
	if server.Grant.Load().ValidConfig && (server.WebSocket.Connected() ||
		(!refetchAlways && server.Grant.Load().ValidForS3Until.After(time.Now()))) {
		return nil
	}

	err := server.WebSocket.Connect()
	if err != nil {
		server.SelfTest.MarkCollectionAspectError(state.CollectionAspectWebSocket, "error starting WebSocket: %s", err)
		if server.Config.APIRequireWebsocket {
			return fmt.Errorf("Error starting WebSocket: %w", err)
		}
	} else {
		// Wait for initial config so we don't incorrectly use an HTTP-based grant
		ok := waitWithTimeout(ctx, server.InitialConfigReceived, 1*time.Second)
		if ok && server.Grant.Load().ValidConfig {
			server.SelfTest.MarkCollectionAspectOk(state.CollectionAspectWebSocket)
			server.SelfTest.MarkCollectionAspectOk(state.CollectionAspectApiConnection)
			return nil
		} else {
			server.SelfTest.MarkCollectionAspectError(state.CollectionAspectWebSocket, "error starting WebSocket: initial configuration not received in time")
			if server.Config.APIRequireWebsocket {
				return fmt.Errorf("Error starting WebSocket: initial configuration not received in time")
			}
		}
	}

	newGrant, err := getGrant(ctx, server.Config, opts.TestRun)
	if err != nil {
		server.SelfTest.MarkCollectionAspectError(state.CollectionAspectApiConnection, "error contacting API: %s", err)
		if server.Grant.Load().ValidForS3Until.After(time.Now()) {
			logger.PrintVerbose("Could not acquire snapshot grant, reusing previous grant: %s", err)
			return nil
		}
		return err
	}

	server.SelfTest.MarkCollectionAspectOk(state.CollectionAspectApiConnection)
	server.Grant.Store(newGrant)

	return nil
}

func waitWithTimeout(ctx context.Context, c chan struct{}, timeout time.Duration) bool {
	tctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for {
		select {
		case <-tctx.Done():
			return false
		case <-c:
			return true
		}
	}
}

func getGrant(ctx context.Context, conf config.ServerConfig, testRun bool) (*state.Grant, error) {
	grant := &state.Grant{Config: pganalyze_collector.ServerMessage_Config{Features: &pganalyze_collector.ServerMessage_Features{}}}
	req, err := http.NewRequestWithContext(ctx, "GET", conf.APIBaseURL+"/v2/snapshots/grant", nil)
	if err != nil {
		return grant, err
	}

	req.Header = config.APIHeaders(conf, testRun)
	req.Header.Add("Accept", "application/json")

	resp, err := conf.HTTPClientWithRetry.Do(req)
	if err != nil {
		cleanErr := util.CleanHTTPError(err)
		return grant, cleanErr
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return grant, err
	}

	if resp.StatusCode != http.StatusOK || len(body) == 0 {
		return grant, fmt.Errorf("Error when getting grant: %s", body)
	}

	err = json.Unmarshal(body, &grant)
	if err != nil {
		return grant, err
	}
	grant.ValidConfig = true
	grant.ValidForS3Until = time.Now().Add(1 * time.Hour)

	return grant, nil
}

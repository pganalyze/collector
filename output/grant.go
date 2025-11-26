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
	if server.Grant.Load().ValidConfig && (server.WebSocket.Load() != nil ||
		(!refetchAlways && server.Grant.Load().ValidForS3Until.After(time.Now()))) {
		return nil
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

func getGrant(ctx context.Context, conf config.ServerConfig, testRun bool) (*state.Grant, error) {
	grant := &state.Grant{Config: pganalyze_collector.ServerMessage_Config{Features: &pganalyze_collector.ServerMessage_Features{}}}
	req, err := http.NewRequestWithContext(ctx, "GET", conf.APIBaseURL+"/v2/snapshots/grant", nil)
	if err != nil {
		return grant, err
	}

	req.Header.Set("Pganalyze-Api-Key", conf.APIKey)
	req.Header.Set("Pganalyze-System-Id", conf.SystemID)
	req.Header.Set("Pganalyze-System-Type", conf.SystemType)
	req.Header.Set("Pganalyze-System-Scope", conf.SystemScope)
	req.Header.Set("Pganalyze-System-Id-Fallback", conf.SystemIDFallback)
	req.Header.Set("Pganalyze-System-Type-Fallback", conf.SystemTypeFallback)
	req.Header.Set("Pganalyze-System-Scope-Fallback", conf.SystemScopeFallback)
	req.Header.Set("User-Agent", util.CollectorNameAndVersion)
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

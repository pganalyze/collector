package snapshot_api

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func getGrant(ctx context.Context, server *state.Server, opts state.CollectionOpts, logger *util.Logger) (state.Grant, error) {
	grant := state.Grant{Config: pganalyze_collector.ServerMessage_Config{Features: &pganalyze_collector.ServerMessage_Features{}}}
	req, err := http.NewRequestWithContext(ctx, "GET", server.Config.APIBaseURL+"/v2/snapshots/grant", nil)
	if err != nil {
		server.SelfTest.MarkCollectionAspectError(state.CollectionAspectApiConnection, "%s", err.Error())
		return grant, err
	}

	req.Header.Set("Pganalyze-Api-Key", server.Config.APIKey)
	req.Header.Set("Pganalyze-System-Id", server.Config.SystemID)
	req.Header.Set("Pganalyze-System-Type", server.Config.SystemType)
	req.Header.Set("Pganalyze-System-Scope", server.Config.SystemScope)
	req.Header.Set("Pganalyze-System-Id-Fallback", server.Config.SystemIDFallback)
	req.Header.Set("Pganalyze-System-Type-Fallback", server.Config.SystemTypeFallback)
	req.Header.Set("Pganalyze-System-Scope-Fallback", server.Config.SystemScopeFallback)
	req.Header.Set("User-Agent", util.CollectorNameAndVersion)
	req.Header.Add("Accept", "application/json")

	resp, err := server.Config.HTTPClientWithRetry.Do(req)
	if err != nil {
		cleanErr := util.CleanHTTPError(err)
		server.SelfTest.MarkCollectionAspectError(state.CollectionAspectApiConnection, "error contacting API: %s", cleanErr)
		return grant, cleanErr
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		server.SelfTest.MarkCollectionAspectError(state.CollectionAspectApiConnection, "error contacting API: %s", err)
		return grant, err
	}

	if resp.StatusCode != http.StatusOK || len(body) == 0 {
		server.SelfTest.MarkCollectionAspectError(state.CollectionAspectApiConnection, "error contacting API: %s", body)
		return grant, fmt.Errorf("Error when getting grant: %s", body)
	}

	err = json.Unmarshal(body, &grant)
	if err != nil {
		server.SelfTest.MarkCollectionAspectError(state.CollectionAspectApiConnection, "error deserializing API response: %s", err)
		return grant, err
	}
	grant.Valid = true

	server.SelfTest.MarkCollectionAspectOk(state.CollectionAspectApiConnection)

	return grant, nil
}

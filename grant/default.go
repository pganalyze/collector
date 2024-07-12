package grant

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func GetDefaultGrant(ctx context.Context, server *state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) (state.Grant, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", server.Config.APIBaseURL+"/v2/snapshots/grant", nil)
	if err != nil {
		server.SelfTest.MarkCollectionAspectError(state.CollectionAspectApiConnection, err.Error())
		return state.Grant{}, err
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
		return state.Grant{}, cleanErr
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		server.SelfTest.MarkCollectionAspectError(state.CollectionAspectApiConnection, "error contacting API: %s", err)
		return state.Grant{}, err
	}

	if resp.StatusCode != http.StatusOK || len(body) == 0 {
		server.SelfTest.MarkCollectionAspectError(state.CollectionAspectApiConnection, "error contacting API: %s", body)
		return state.Grant{}, fmt.Errorf("Error when getting grant: %s", body)
	}

	grant := state.Grant{}
	err = json.Unmarshal(body, &grant)
	if err != nil {
		server.SelfTest.MarkCollectionAspectError(state.CollectionAspectApiConnection, "error deserializing API response: %s", err)
		return state.Grant{}, err
	}
	grant.Valid = true

	server.SelfTest.MarkCollectionAspectOk(state.CollectionAspectApiConnection)

	return grant, nil
}

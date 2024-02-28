package grant

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func GetDefaultGrant(server *state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) (state.Grant, error) {
	req, err := http.NewRequest("GET", server.Config.APIBaseURL+"/v2/snapshots/grant", nil)
	if err != nil {
		return state.Grant{}, err
	}

	req.Header.Set("Pganalyze-Api-Key", server.Config.APIKey)
	req.Header.Set("Pganalyze-System-Id", server.Config.SystemID)
	// TODO(ianstanton) Placeholder for local dev
	req.Header.Set("Pganalyze-System-Type", "self_hosted")
	req.Header.Set("Pganalyze-System-Scope", server.Config.SystemScope)
	req.Header.Set("Pganalyze-System-Id-Fallback", server.Config.SystemIDFallback)
	req.Header.Set("Pganalyze-System-Type-Fallback", server.Config.SystemTypeFallback)
	req.Header.Set("Pganalyze-System-Scope-Fallback", server.Config.SystemScopeFallback)
	req.Header.Set("User-Agent", util.CollectorNameAndVersion)
	req.Header.Add("Accept", "application/json")

	resp, err := server.Config.HTTPClientWithRetry.Do(req)
	if err != nil {
		return state.Grant{}, util.CleanHTTPError(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return state.Grant{}, err
	}

	if resp.StatusCode != http.StatusOK || len(body) == 0 {
		return state.Grant{}, fmt.Errorf("Error when getting grant: %s", body)
	}

	grant := state.Grant{}
	err = json.Unmarshal(body, &grant)
	if err != nil {
		return state.Grant{}, err
	}
	grant.Valid = true

	return grant, nil
}

package grant

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func GetLogsGrant(server *state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) (state.GrantLogs, error) {
	req, err := http.NewRequest("GET", server.Config.APIBaseURL+"/v2/snapshots/grant_logs", nil)
	if err != nil {
		return state.GrantLogs{}, err
	}

	req.Header.Set("Pganalyze-Api-Key", server.Config.APIKey)
	req.Header.Set("Pganalyze-System-Id", server.Config.SystemID)
	req.Header.Set("Pganalyze-System-Type", server.Config.SystemType)
	req.Header.Set("Pganalyze-System-Scope", server.Config.SystemScope)
	req.Header.Set("Pganalyze-System-Scope-Fallback", server.Config.SystemScopeFallback)
	req.Header.Set("User-Agent", util.CollectorNameAndVersion)
	req.Header.Add("Accept", "application/json")

	resp, err := server.Config.HTTPClient.Do(req)
	if err != nil {
		return state.GrantLogs{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return state.GrantLogs{}, err
	}

	if resp.StatusCode == http.StatusForbidden {
		return state.GrantLogs{}, nil
	}

	if resp.StatusCode != http.StatusOK || len(body) == 0 {
		return state.GrantLogs{}, fmt.Errorf("Error when getting grant: %s", body)
	}

	grant := state.GrantLogs{}
	err = json.Unmarshal(body, &grant)
	if err != nil {
		return state.GrantLogs{}, err
	}
	grant.Valid = true

	return grant, nil
}

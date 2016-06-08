package runner

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/pganalyze/collector/input"
	"github.com/pganalyze/collector/output"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func processDatabase(db state.Database, globalCollectionOpts state.CollectionOpts, logger *util.Logger) (state.State, state.Grant, error) {
	// Note: In case of server errors, we should reuse the old grant if its still recent (i.e. less than 50 minutes ago)
	grant, err := getSnapshotGrant(db, globalCollectionOpts, logger)
	if err != nil {
		logger.PrintVerbose("Could not acquire snapshot grant, reusing previous grant: %s", err)
	} else {
		db.Grant = grant
	}

	newState, err := input.CollectFull(db, globalCollectionOpts, logger)
	if err != nil {
		return newState, grant, err
	}

	diffState := diffState(logger, db.PrevState, newState)

	output.SendFull(db, globalCollectionOpts, logger, newState, diffState)

	return newState, grant, nil
}

func getSnapshotGrant(db state.Database, globalCollectionOpts state.CollectionOpts, logger *util.Logger) (state.Grant, error) {
	var grant state.Grant

	req, err := http.NewRequest("GET", db.Config.APIBaseURL+"/v2/snapshots/grant", nil)
	if err != nil {
		return grant, err
	}

	req.Header.Set("Pganalyze-Api-Key", db.Config.APIKey)
	req.Header.Set("User-Agent", util.CollectorNameAndVersion)
	req.Header.Add("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return grant, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return grant, err
	}

	if resp.StatusCode != http.StatusOK || len(body) == 0 {
		return grant, fmt.Errorf("Error when getting grant: %s\n", body)
	}

	err = json.Unmarshal(body, &grant)
	if err != nil {
		return grant, err
	}

	return grant, nil
}

func CollectAllDatabases(databases []state.Database, globalCollectionOpts state.CollectionOpts, logger *util.Logger) {
	for idx, db := range databases {
		var err error

		prefixedLogger := logger.WithPrefix(db.Config.SectionName)

		db.Connection, err = establishConnection(db, logger, globalCollectionOpts)
		if err != nil {
			prefixedLogger.PrintError("Error: Failed to connect to database: %s", err)
			return
		}

		newState, grant, err := processDatabase(db, globalCollectionOpts, prefixedLogger)
		if err != nil {
			prefixedLogger.PrintError("Error: Could not process database: %s", err)
		} else {
			databases[idx].Grant = grant
			databases[idx].PrevState = newState
		}

		// This is the easiest way to avoid opening multiple connections to different databases on the same instance
		db.Connection.Close()
		db.Connection = nil
	}
}

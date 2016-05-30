package runner

import (
	"github.com/pganalyze/collector/input"
	"github.com/pganalyze/collector/output"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func processDatabase(db state.Database, globalCollectionOpts state.CollectionOpts, logger *util.Logger) (state.State, error) {
	newState, err := input.CollectFull(db, globalCollectionOpts, logger)
	if err != nil {
		return newState, err
	}

	diffState := diffState(logger, db.PrevState, newState)

	output.SendFull(db, globalCollectionOpts, logger, newState, diffState)

	return newState, nil
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

		newState, err := processDatabase(db, globalCollectionOpts, prefixedLogger)
		if err != nil {
			prefixedLogger.PrintError("Error: Could not process database: %s", err)
		} else {
			databases[idx].PrevState = newState
		}

		// This is the easiest way to avoid opening multiple connections to different databases on the same instance
		db.Connection.Close()
		db.Connection = nil
	}
}

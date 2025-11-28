package runner

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/pganalyze/collector/input"
	"github.com/pganalyze/collector/input/postgres"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	"github.com/pkg/errors"
)

func gather1minStatsForServer(ctx context.Context, server *state.Server, opts state.CollectionOpts, logger *util.Logger) (state.PersistedHighFreqState, error) {
	var err error
	var connection *sql.DB

	newState := server.HighFreqPrevState
	collectedAt := time.Now()

	connection, err = postgres.EstablishConnection(ctx, server, logger, opts, "")
	if err != nil {
		return newState, errors.Wrap(err, "failed to connect to database")
	}
	defer connection.Close()

	if server.Config.SkipIfReplica {
		err = checkReplicaCollectionDisabledWithConn(ctx, server, logger, connection)
		if err != nil {
			return newState, err
		}
	}

	c, err := postgres.NewCollection(ctx, logger, server, opts, connection)
	if err != nil {
		return newState, err
	}

	return input.CollectAndDiff1minStats(ctx, c, connection, collectedAt, server.HighFreqPrevState)
}

func Gather1minStatsFromAllServers(ctx context.Context, servers []*state.Server, opts state.CollectionOpts, logger *util.Logger) {
	var wg sync.WaitGroup

	for idx := range servers {
		if servers[idx].Config.QueryStatsInterval != 60 {
			continue
		}

		wg.Add(1)
		go func(server *state.Server) {
			prefixedLogger := logger.WithPrefixAndRememberErrors(server.Config.SectionName)

			server.HighFreqStateMutex.Lock()
			newState, err := gather1minStatsForServer(ctx, server, opts, prefixedLogger)

			if err != nil {
				server.HighFreqStateMutex.Unlock()

				if err == state.ErrReplicaCollectionDisabled {
					prefixedLogger.PrintVerbose("All monitoring suspended while server is replica")
				} else {
					prefixedLogger.PrintError("Could not collect high frequency statistics for server: %s", err)
					if server.Config.ErrorCallback != "" {
						go runCompletionCallback("error", server.Config.ErrorCallback, server.Config.SectionName, "query_stats", err, prefixedLogger)
					}
				}
			} else {
				server.HighFreqPrevState = newState
				server.HighFreqStateMutex.Unlock()
				prefixedLogger.PrintVerbose("Successfully collected high frequency statistics")
				if server.Config.SuccessCallback != "" {
					go runCompletionCallback("success", server.Config.SuccessCallback, server.Config.SectionName, "query_stats", nil, prefixedLogger)
				}
			}
			wg.Done()
		}(servers[idx])
	}

	// TODO: We currently do not write out the state file here, since we do not want to block
	// subsequent high-frequency stats collections in case the full snapshot is still running
	// (which holds the state mutex that the state file write also wants to acquire). That means
	// in case of collector crashes we may have an incorrect reference point on a subsequent start.
	//
	// We could potentially address this by using a read/write mutex that is held in read mode
	// during the collection, and only elevated to write (i.e. blocking other readers) once we swap
	// out the state stored on the server struct.

	wg.Wait()
}

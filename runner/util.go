package runner

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pganalyze/collector/input/postgres"
	"github.com/pganalyze/collector/runner/snapshot_api"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func checkReplicaCollectionDisabled(ctx context.Context, server *state.Server, opts state.CollectionOpts, logger *util.Logger) error {
	if !server.Config.SkipIfReplica {
		return nil
	}

	connection, err := postgres.EstablishConnection(ctx, server, logger, opts, "")
	if err != nil {
		return fmt.Errorf("Failed to connect to database: %s", err)
	}
	defer connection.Close()

	return checkReplicaCollectionDisabledWithConn(ctx, server, logger, connection)
}

func checkReplicaCollectionDisabledWithConn(ctx context.Context, server *state.Server, logger *util.Logger, connection *sql.DB) error {
	isReplica, err := postgres.GetIsReplica(ctx, logger, connection)
	if err != nil {
		return fmt.Errorf("Error checking replication status")
	}
	if isReplica {
		// Shut down websocket so another server that has become primary can collect without
		// being considered the older collector (which could cause the server to pause it)
		snapshot_api.ShutdownWebSocketIfNeeded(server)

		reason := state.ErrReplicaCollectionDisabled.Error()
		server.CollectionStatusMutex.Lock()
		server.CollectionStatus = state.CollectionStatus{
			CollectionDisabled:        true,
			CollectionDisabledReason:  reason,
			LogSnapshotDisabled:       true,
			LogSnapshotDisabledReason: reason,
		}
		server.CollectionStatusMutex.Unlock()

		return state.ErrReplicaCollectionDisabled
	} else {
		return nil
	}
}

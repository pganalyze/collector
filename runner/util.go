package runner

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pganalyze/collector/input/postgres"
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
		// Shut down WebSocket so another server that has become primary can collect without
		// being considered the older collector (which could cause the server to pause it)
		server.WebSocket.Disconnect()

		reason := state.ErrReplicaCollectionDisabled.Error()
		server.CollectionStatusMutex.Lock()
		if !server.CollectionStatus.CollectionDisabled {
			logger.PrintInfo("Server cluster role changed from primary to replica, turning off statistics collection (replica collection disabled via config)")
		}
		server.CollectionStatus.CollectionDisabled = true
		server.CollectionStatus.CollectionDisabledReason = reason
		server.CollectionStatusMutex.Unlock()

		return state.ErrReplicaCollectionDisabled
	} else {
		// WebSocket gets restarted by grant mechanism, so we don't do it here

		server.CollectionStatusMutex.Lock()
		if server.CollectionStatus.CollectionDisabled && server.CollectionStatus.CollectionDisabledReason == state.ErrReplicaCollectionDisabled.Error() {
			logger.PrintInfo("Server cluster role changed from replica to primary, re-enabling statistics collection")
			server.CollectionStatus.CollectionDisabled = false
			server.CollectionStatus.CollectionDisabledReason = ""
		}
		server.CollectionStatusMutex.Unlock()
		return nil
	}
}

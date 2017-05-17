package input

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/pganalyze/collector/input/postgres"
	"github.com/pganalyze/collector/input/system"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

// CollectFull - Collects a "full" snapshot of all data we need on a regular interval
func CollectFull(server state.Server, connection *sql.DB, collectionOpts state.CollectionOpts, logger *util.Logger) (ps state.PersistedState, ts state.TransientState, err error) {
	ps.CollectedAt = time.Now()

	ts.Version, err = postgres.GetPostgresVersion(logger, connection)
	if err != nil {
		logger.PrintError("Error collecting Postgres Version")
		return
	}

	if ts.Version.Numeric < state.MinRequiredPostgresVersion {
		err = fmt.Errorf("Error: Your PostgreSQL server version (%s) is too old, 9.2 or newer is required.", ts.Version.Short)
		return
	}

	ts.Roles, err = postgres.GetRoles(logger, connection, ts.Version)
	if err != nil {
		logger.PrintError("Error collecting pg_roles")
		return
	}

	ts.Databases, err = postgres.GetDatabases(logger, connection, ts.Version)
	if err != nil {
		logger.PrintError("Error collecting pg_databases")
		return
	}

	ts.Backends, err = postgres.GetBackends(logger, connection, ts.Version)
	if err != nil {
		logger.PrintError("Error collecting pg_stat_activity")
		return
	}

	ps.StatementTextCounter = server.PrevState.StatementTextCounter + 1
	if ps.StatementTextCounter >= server.Grant.Config.Features.StatementTextFrequency { // Stats and statements
		ps.StatementTextCounter = 0
		ts.HasStatementText = true
		ts.Statements, ps.StatementStats, err = postgres.GetStatements(logger, connection, ts.Version, true)
		if err != nil {
			logger.PrintError("Error collecting pg_stat_statements")
			return
		}
	} else { // Stats only
		logger.PrintVerbose("Collecting pg_stat_statements without statement text (%d of %d)", ps.StatementTextCounter, server.Grant.Config.Features.StatementTextFrequency)
		ts.HasStatementText = false
		_, ps.StatementStats, err = postgres.GetStatements(logger, connection, ts.Version, false)
		if err != nil {
			logger.PrintError("Error collecting pg_stat_statements")
			return
		}
	}

	ps.StatementResetCounter = server.PrevState.StatementResetCounter + 1
	if server.Grant.Config.Features.StatementResetFrequency != 0 && ps.StatementResetCounter >= server.Grant.Config.Features.StatementResetFrequency {
		ps.StatementResetCounter = 0
		err = postgres.ResetStatements(logger, connection)
		if err != nil {
			logger.PrintError("Error calling pg_stat_statements_reset() as requested: %s", err)
			return
		}
		_, ts.ResetStatementStats, err = postgres.GetStatements(logger, connection, ts.Version, false)
		if err != nil {
			logger.PrintError("Error collecting pg_stat_statements")
			return
		}
	}

	if collectionOpts.CollectPostgresSettings {
		ts.Settings, err = postgres.GetSettings(connection, ts.Version)
		if err != nil {
			logger.PrintError("Error collecting config settings")
			return
		}
	}

	ts.Replication, err = postgres.GetReplication(logger, connection)
	if err != nil {
		logger.PrintWarning("Error collecting replication statistics: %s", err)
		// We intentionally accept this as a non-fatal issue (at least for now)
		err = nil
	}

	ps, ts = postgres.CollectAllSchemas(server, collectionOpts, logger, ps, ts)

	if collectionOpts.CollectSystemInformation {
		ps.System = system.GetSystemState(server.Config, logger)
	}

	ps.CollectorStats = getCollectorStats()

	return
}

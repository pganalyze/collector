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
	var explainInputs []state.PostgresExplainInput

	ps.CollectedAt = time.Now()

	ps.Version, err = postgres.GetPostgresVersion(logger, connection)
	if err != nil {
		logger.PrintError("Error collecting Postgres Version")
		return
	}

	if ps.Version.Numeric < state.MinRequiredPostgresVersion {
		err = fmt.Errorf("Error: Your PostgreSQL server version (%s) is too old, 9.2 or newer is required.", ps.Version.Short)
		return
	}

	ps.Roles, err = postgres.GetRoles(logger, connection, ps.Version)
	if err != nil {
		logger.PrintError("Error collecting pg_roles")
		return
	}

	ps.Databases, err = postgres.GetDatabases(logger, connection, ps.Version)
	if err != nil {
		logger.PrintError("Error collecting pg_databases")
		return
	}

	ps.Backends, err = postgres.GetBackends(logger, connection, ps.Version)
	if err != nil {
		logger.PrintError("Error collecting pg_stat_activity")
		return
	}

	ps.StatementFrequencyCounter = server.PrevState.StatementFrequencyCounter + 1
	if ps.StatementFrequencyCounter >= server.Config.DbStatementFrequency { // Stats and statements
		ps.StatementFrequencyCounter = 0
		ts.HasStatementText = true
		ts.Statements, ps.StatementStats, err = postgres.GetStatements(logger, connection, ps.Version, true)
		if err != nil {
			logger.PrintError("Error collecting pg_stat_statements")
			return
		}
	} else { // Stats only
		logger.PrintVerbose("Collecting pg_stat_statements without statement text (%d of %d)", ps.StatementFrequencyCounter, server.Config.DbStatementFrequency)
		ts.HasStatementText = false
		_, ps.StatementStats, err = postgres.GetStatements(logger, connection, ps.Version, false)
		if err != nil {
			logger.PrintError("Error collecting pg_stat_statements")
			return
		}
	}

	if collectionOpts.CollectPostgresSettings {
		ps.Settings, err = postgres.GetSettings(connection, ps.Version)
		if err != nil {
			logger.PrintError("Error collecting config settings")
			return
		}
	}

	ps = postgres.CollectAllSchemas(server, collectionOpts, logger, ps)

	if collectionOpts.CollectSystemInformation {
		ps.System = system.GetSystemState(server.Config, logger)
	}

	if collectionOpts.CollectLogs {
		ps.Logs, explainInputs = system.GetLogLines(server.Config)

		if collectionOpts.CollectExplain {
			ps.Explains = postgres.RunExplain(connection, explainInputs)
		}
	}

	ps.CollectorStats = getCollectorStats()

	return
}

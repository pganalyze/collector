package input

import (
	"fmt"

	"github.com/pganalyze/collector/input/postgres"
	"github.com/pganalyze/collector/input/system"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func CollectFull(db state.Database, collectionOpts state.CollectionOpts, logger *util.Logger) (s state.State, err error) {
	var explainInputs []state.PostgresExplainInput

	postgresVersion, err := postgres.GetPostgresVersion(logger, db.Connection)
	if err != nil {
		logger.PrintError("Error collecting Postgres Version")
		return
	}

	/*stats.Postgres = &snapshot.SnapshotPostgres{}
	stats.Postgres.Version = &postgresVersion*/

	if postgresVersion.Numeric < state.MinRequiredPostgresVersion {
		err = fmt.Errorf("Error: Your PostgreSQL server version (%s) is too old, 9.2 or newer is required.", postgresVersion.Short)
		return
	}

	s.Backends, err = postgres.GetBackends(logger, db.Connection, postgresVersion)
	if err != nil {
		logger.PrintError("Error collecting pg_stat_activity")
		return
	}

	s.Statements, err = postgres.GetStatements(logger, db.Connection, postgresVersion)
	if err != nil {
		logger.PrintError("Error collecting pg_stat_statements")
		return
	}

	if collectionOpts.CollectPostgresRelations {
		s.Relations, err = postgres.GetRelations(db.Connection, postgresVersion)
		if err != nil {
			logger.PrintError("Error collecting relation/index information: %s", err)
			return
		}

		s.RelationStats, err = postgres.GetRelationStats(db.Connection, postgresVersion)
		if err != nil {
			logger.PrintError("Error collecting relation stats: %s", err)
			return
		}

		s.IndexStats, err = postgres.GetIndexStats(db.Connection, postgresVersion)
		if err != nil {
			logger.PrintError("Error collecting index stats: %s", err)
			return
		}

		// collectionOpts.CollectPostgresBloat
	}

	if collectionOpts.CollectPostgresSettings {
		s.Settings, err = postgres.GetSettings(db.Connection, postgresVersion)
		if err != nil {
			logger.PrintError("Error collecting config settings")
			return
		}
	}

	if collectionOpts.CollectPostgresFunctions {
		s.Functions, err = postgres.GetFunctions(db.Connection, postgresVersion)
		if err != nil {
			logger.PrintError("Error collecting stored procedures")
			return
		}
	}

	if collectionOpts.CollectSystemInformation {
		systemState := system.GetSystemState(db.Config, logger)
		s.System = &systemState
	}

	if collectionOpts.CollectLogs {
		s.Logs, explainInputs = system.GetLogLines(db.Config)

		if collectionOpts.CollectExplain {
			s.Explains = postgres.RunExplain(db.Connection, explainInputs)
		}
	}

	return
}

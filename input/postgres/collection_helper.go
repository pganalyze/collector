package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

type CollectionHelper struct {
	Config          config.ServerConfig
	Logger          *util.Logger
	SelfTest        *state.SelfTestResult
	GlobalOpts      state.CollectionOpts
	PostgresVersion state.PostgresVersion
}

func NewCollectionHelper(ctx context.Context, logger *util.Logger, server *state.Server, globalOpts state.CollectionOpts, db *sql.DB) (CollectionHelper, error) {
	version, err := GetPostgresVersion(ctx, db)
	if err != nil {
		server.SelfTest.MarkCollectionAspectError(state.CollectionAspectPgVersion, err.Error())
		return CollectionHelper{}, fmt.Errorf("failed collecting Postgres Version: %s", err)
	}
	logger.PrintVerbose("Detected PostgreSQL Version %d (%s)", version.Numeric, version.Full)
	if version.Numeric < state.MinRequiredPostgresVersion {
		server.SelfTest.MarkCollectionAspectError(state.CollectionAspectPgVersion, "PostgreSQL server version (%s) is too old, 10 or newer is required.", version.Short)
		return CollectionHelper{}, fmt.Errorf("your PostgreSQL server version (%s) is too old, 10 or newer is required", version.Short)
	}
	server.SelfTest.MarkCollectionAspect(state.CollectionAspectPgVersion, state.CollectionStateOkay, version.Short)

	return CollectionHelper{
		Config:          server.Config,
		Logger:          logger,
		SelfTest:        server.SelfTest,
		GlobalOpts:      globalOpts,
		PostgresVersion: version,
	}, nil
}

package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

type Collection struct {
	Config                    config.ServerConfig
	Logger                    *util.Logger
	SelfTest                  *state.SelfTestResult
	GlobalOpts                state.CollectionOpts
	PostgresVersion           state.PostgresVersion
	Roles                     []state.PostgresRole
	ConnectedAsSuperUser      bool
	ConnectedAsMonitoringRole bool
}

func NewCollection(ctx context.Context, logger *util.Logger, server *state.Server, globalOpts state.CollectionOpts, db *sql.DB) (*Collection, error) {
	version, err := getPostgresVersion(ctx, db)
	if err != nil {
		server.SelfTest.MarkCollectionAspectError(state.CollectionAspectPgVersion, err.Error())
		return &Collection{}, fmt.Errorf("failed collecting Postgres Version: %s", err)
	}
	logger.PrintVerbose("Detected PostgreSQL Version %d (%s)", version.Numeric, version.Full)
	if version.Numeric < state.MinRequiredPostgresVersion {
		server.SelfTest.MarkCollectionAspectError(state.CollectionAspectPgVersion, "PostgreSQL server version (%s) is too old, 10 or newer is required.", version.Short)
		return &Collection{}, fmt.Errorf("your PostgreSQL server version (%s) is too old, 10 or newer is required", version.Short)
	}
	server.SelfTest.MarkCollectionAspect(state.CollectionAspectPgVersion, state.CollectionStateOkay, version.Short)

	roles, err := getRoles(ctx, db, server.Config.SystemType)
	if err != nil {
		return &Collection{}, fmt.Errorf("failed collecting pg_roles: %s", err)
	}

	roleByName := make(map[string]state.PostgresRole)
	for _, role := range roles {
		roleByName[role.Name] = role
	}
	collectorRole := roleByName[server.Config.GetDbUsername()]
	connectedAsSuperUser := collectorRole.SuperUser || collectorRole.CloudSuperUser
	connectedAsMonitoringRole := collectorRole.MonitoringUser

	return &Collection{
		Config:                    server.Config,
		Logger:                    logger,
		SelfTest:                  server.SelfTest,
		GlobalOpts:                globalOpts,
		PostgresVersion:           version,
		Roles:                     roles,
		ConnectedAsSuperUser:      connectedAsSuperUser,
		ConnectedAsMonitoringRole: connectedAsMonitoringRole,
	}, nil
}

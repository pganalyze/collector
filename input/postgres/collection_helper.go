package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"slices"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

type CollectionHelper struct {
	Config                    config.ServerConfig
	Logger                    *util.Logger
	SelfTest                  *state.SelfTestResult
	GlobalOpts                state.CollectionOpts
	PostgresVersion           state.PostgresVersion
	Roles                     []state.PostgresRole
	ConnectedAsSuperUser      bool
	ConnectedAsMonitoringRole bool
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

	roles, err := GetRoles(ctx, db)
	if err != nil {
		return CollectionHelper{}, fmt.Errorf("failed collecting pg_roles: %s", err)
	}

	roleByName := make(map[string]state.PostgresRole)
	roleByOid := make(map[state.Oid]state.PostgresRole)
	for _, role := range roles {
		roleByName[role.Name] = role
		roleByOid[role.Oid] = role
	}

	collectorRole := roleByName[server.Config.GetDbUsername()]
	memberOf := collectorRole.MemberOf
	for _, m := range memberOf { // Allow one level of indirect role memberships
		memberOf = append(memberOf, roleByOid[m].MemberOf...)
	}
	connectedAsSuperUser := collectorRole.SuperUser ||
		slices.Contains(memberOf, roleByName["rds_superuser"].Oid) ||
		slices.Contains(memberOf, roleByName["azure_pg_admin"].Oid) ||
		slices.Contains(memberOf, roleByName["cloudsqlsuperuser"].Oid)
	connectedAsMonitoringRole := slices.Contains(memberOf, roleByName["pg_monitor"].Oid)

	return CollectionHelper{
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

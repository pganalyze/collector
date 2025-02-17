package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

type CollectionHelper struct {
	// Information that applies to all databases on a server
	Config                    config.ServerConfig
	Logger                    *util.Logger
	SelfTest                  *state.SelfTestResult
	GlobalOpts                state.CollectionOpts
	PostgresVersion           state.PostgresVersion
	Roles                     []state.PostgresRole
	ConnectedAsSuperUser      bool
	ConnectedAsMonitoringRole bool

	// Information that is specific to the current database we're connected to
	HelperFunctions map[string][]state.PostgresFunction
}

func helpersFromFunctions(functions []state.PostgresFunction) map[string][]state.PostgresFunction {
	helpers := make(map[string][]state.PostgresFunction)
	for _, f := range functions {
		if f.SchemaName != "pganalyze" {
			continue
		}
		funcs, ok := helpers[f.FunctionName]
		if ok {
			helpers[f.FunctionName] = append(funcs, f)
		} else {
			helpers[f.FunctionName] = []state.PostgresFunction{f}
		}
	}
	return helpers
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
	helperFunctions, err := GetFunctions(ctx, logger, db, version, 0, "", true)
	if err != nil {
		return CollectionHelper{}, fmt.Errorf("failed collecting pg_proc: %s", err)
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
		HelperFunctions:           helpersFromFunctions(helperFunctions),
	}, nil
}

func (s CollectionHelper) ForCurrentDatabase(functions []state.PostgresFunction) CollectionHelper {
	return CollectionHelper{
		Config:                    s.Config,
		Logger:                    s.Logger,
		SelfTest:                  s.SelfTest,
		GlobalOpts:                s.GlobalOpts,
		PostgresVersion:           s.PostgresVersion,
		Roles:                     s.Roles,
		ConnectedAsSuperUser:      s.ConnectedAsSuperUser,
		ConnectedAsMonitoringRole: s.ConnectedAsMonitoringRole,
		HelperFunctions:           helpersFromFunctions(functions),
	}
}

func (s CollectionHelper) findHelperFunction(name string, inputTypes []string) (state.PostgresFunction, bool) {
	funcs, exists := s.HelperFunctions[name]
	if !exists {
		return state.PostgresFunction{}, false
	}
	for _, f := range funcs {
		args := strings.Split(f.Arguments, ", ")
		if len(inputTypes) > len(args) {
			// We're expecting more arguments than the function has
			continue
		}
		mismatch := false
		for idx, arg := range args {
			// Split by this assumed pattern: "NAME TYPE DEFAULT DEFAULT_VALUE"
			//
			// Note this currently does not handle data types with spaces in
			// them, such as "double precision", or other cases not expected
			// with our known set of helper functions.
			parts := strings.Split(arg, " ")
			if idx >= len(inputTypes) && parts[2] != "DEFAULT" {
				// Function has more arguments required than we're expecting
				mismatch = true
				break
			}
			if parts[1] != inputTypes[idx] {
				mismatch = true
				break
			}
		}
		if !mismatch {
			return f, true
		}
	}
	return state.PostgresFunction{}, false
}

func (s CollectionHelper) HelperExists(name string, inputTypes []string) bool {
	_, ok := s.findHelperFunction(name, inputTypes)
	return ok
}

func (s CollectionHelper) HelperReturnType(name string, inputTypes []string) string {
	f, ok := s.findHelperFunction(name, inputTypes)
	if ok {
		return f.Result
	}
	return ""
}

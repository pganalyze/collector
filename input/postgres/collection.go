package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

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

	// Information that is specific to the current database we're connected to
	HelperFunctions map[string][]state.PostgresFunction

	Fingerprints *state.Fingerprints
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

func NewCollection(ctx context.Context, logger *util.Logger, server *state.Server, globalOpts state.CollectionOpts, db *sql.DB) (*Collection, error) {
	version, err := getPostgresVersion(ctx, db)
	if err != nil {
		server.SelfTest.MarkCollectionAspectError(state.CollectionAspectPgVersion, "%s", err.Error())
		return &Collection{}, fmt.Errorf("failed collecting Postgres Version: %s", err)
	}
	logger.PrintVerbose("Detected PostgreSQL Version %d (%s)", version.Numeric, version.Full)
	if version.Numeric < state.MinRequiredPostgresVersion {
		server.SelfTest.MarkCollectionAspectError(state.CollectionAspectPgVersion, "PostgreSQL server version (%s) is too old, 10 or newer is required.", version.Short)
		return &Collection{}, fmt.Errorf("your PostgreSQL server version (%s) is too old, 10 or newer is required", version.Short)
	}
	server.SelfTest.MarkCollectionAspect(state.CollectionAspectPgVersion, state.CollectionStateOkay, "%s", version.Short)

	roles, err := getRoles(ctx, db, server.Config.SystemType)
	if err != nil {
		return &Collection{}, fmt.Errorf("failed collecting pg_roles: %s", err)
	}
	helperFunctions, err := GetFunctions(ctx, logger, db, version, 0, "", true)
	if err != nil {
		return &Collection{}, fmt.Errorf("failed collecting pg_proc: %s", err)
	}

	roleByName := make(map[string]state.PostgresRole)
	for _, role := range roles {
		roleByName[role.Name] = role
	}
	collectorRole := roleByName[server.Config.GetEffectiveDbUsername()]
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
		HelperFunctions:           helpersFromFunctions(helperFunctions),
		Fingerprints:              server.Fingerprints,
	}, nil
}

func (c *Collection) ForCurrentDatabase(functions []state.PostgresFunction) *Collection {
	return &Collection{
		Config:                    c.Config,
		Logger:                    c.Logger,
		SelfTest:                  c.SelfTest,
		GlobalOpts:                c.GlobalOpts,
		PostgresVersion:           c.PostgresVersion,
		Roles:                     c.Roles,
		ConnectedAsSuperUser:      c.ConnectedAsSuperUser,
		ConnectedAsMonitoringRole: c.ConnectedAsMonitoringRole,
		HelperFunctions:           helpersFromFunctions(functions),
		Fingerprints:              c.Fingerprints,
	}
}

func (c *Collection) findHelperFunction(name string, inputTypes []string) (state.PostgresFunction, bool) {
	funcs, exists := c.HelperFunctions[name]
	if !exists {
		return state.PostgresFunction{}, false
	}
	for _, f := range funcs {
		var args []string
		if f.Arguments != "" {
			args = strings.Split(f.Arguments, ", ")
		}
		if len(inputTypes) > len(args) {
			// We're expecting more arguments than the function has
			continue
		}
		mismatch := false
		for idx, arg := range args {
			// Split by the assumed output pattern of pg_get_function_arguments:
			// "<name> <type> DEFAULT <default>"
			//
			// Note this currently does not handle data types with spaces in
			// them, such as "double precision", or other cases not expected
			// with our known set of helper functions.
			parts := strings.Split(arg, " ")

			// Check if function has more arguments required than we're expecting
			if idx >= len(inputTypes) {
				// Allow extra arguments if they have default values.
				if len(parts) >= 3 && parts[2] == "DEFAULT" {
					break
				}
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

func (c *Collection) HelperExists(name string, inputTypes []string) bool {
	_, ok := c.findHelperFunction(name, inputTypes)
	return ok
}

func (c *Collection) HelperReturnType(name string, inputTypes []string) string {
	f, ok := c.findHelperFunction(name, inputTypes)
	if ok {
		return f.Result
	}
	return ""
}

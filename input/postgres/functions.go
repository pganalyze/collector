package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/guregu/null"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

const functionsSQLDefaultKindFields = "CASE WHEN pp.proisagg THEN 'a' WHEN pp.proiswindow THEN 'w' ELSE 'f' END AS prokind"
const functionsSQLpg11KindFields = "pp.prokind"

const functionsSQLHelperFilter = "n.nspname = 'pganalyze'"

const functionsSQL string = `
SELECT pp.oid,
	   n.nspname,
	   pp.proname,
	   pl.lanname,
	   pp.prosrc,
	   pp.probin,
	   pp.proconfig,
	   pg_catalog.pg_get_function_arguments(pp.oid),
	   COALESCE(pg_catalog.pg_get_function_result(pp.oid), ''),
	   %s,
	   pp.prosecdef,
	   pp.proleakproof,
	   pp.proisstrict,
	   pp.proretset,
	   pp.provolatile
  FROM pg_catalog.pg_proc pp
	   INNER JOIN pg_catalog.pg_namespace n ON (pp.pronamespace = n.oid)
	   INNER JOIN pg_catalog.pg_language pl ON (pp.prolang = pl.oid)
 WHERE %s
	   AND pp.oid NOT IN (SELECT pd.objid FROM pg_catalog.pg_depend pd WHERE pd.deptype = 'e' AND pd.classid = 'pg_catalog.pg_proc'::regclass)
	   AND ($1 = '' OR (n.nspname || '.' || pp.proname) !~* $1)`

const functionStatsSQL string = `
SELECT funcid, calls, total_time, self_time
  FROM pg_stat_user_functions psuf
	   INNER JOIN pg_catalog.pg_proc pp ON (psuf.funcid = pp.oid)
	   INNER JOIN pg_catalog.pg_namespace n ON (pp.pronamespace = n.oid)
 WHERE %s
	   AND pp.oid NOT IN (SELECT pd.objid FROM pg_catalog.pg_depend pd WHERE pd.deptype = 'e' AND pd.classid = 'pg_catalog.pg_proc'::regclass)
	   AND ($1 = '' OR (n.nspname || '.' || pp.proname) !~* $1)`

type FunctionSignature struct {
	SchemaName   string
	FunctionName string
	Arguments    string
}

func GetFunctions(ctx context.Context, logger *util.Logger, db *sql.DB, postgresVersion state.PostgresVersion, currentDatabaseOid state.Oid, ignoreRegexp string, helpersOnly bool) ([]state.PostgresFunction, error) {
	var kindFields string
	var systemCatalogFilter string

	if postgresVersion.Numeric >= state.PostgresVersion11 {
		kindFields = functionsSQLpg11KindFields
	} else {
		kindFields = functionsSQLDefaultKindFields
	}

	if helpersOnly {
		systemCatalogFilter = functionsSQLHelperFilter
	} else if postgresVersion.IsEPAS {
		systemCatalogFilter = relationSQLEPASSystemCatalogFilter
	} else {
		systemCatalogFilter = relationSQLdefaultSystemCatalogFilter
	}

	stmt, err := db.PrepareContext(ctx, QueryMarkerSQL+fmt.Sprintf(functionsSQL, kindFields, systemCatalogFilter))
	if err != nil {
		return nil, err
	}

	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, ignoreRegexp)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var functions []state.PostgresFunction
	var functionSignatures map[FunctionSignature]struct{}

	if postgresVersion.IsEPAS {
		functionSignatures = make(map[FunctionSignature]struct{})
	}

	for rows.Next() {
		var row state.PostgresFunction
		var config null.String

		err := rows.Scan(&row.Oid, &row.SchemaName, &row.FunctionName, &row.Language, &row.Source,
			&row.SourceBin, &config, &row.Arguments, &row.Result, &row.Kind,
			&row.SecurityDefiner, &row.Leakproof, &row.Strict, &row.ReturnsSet, &row.Volatile)
		if err != nil {
			return nil, err
		}

		if postgresVersion.IsEPAS {
			// EDB Postgres Advanced Server allows creating functions and procedures that share a signature
			// (functions take precedence when calling), something which regular Postgres does not allow.
			//
			// For now we ignore them here to avoid server side errors, but we could revise this in the future.
			signature := FunctionSignature{SchemaName: row.SchemaName, FunctionName: row.FunctionName, Arguments: row.Arguments}
			if _, ok := functionSignatures[signature]; ok {
				logger.PrintVerbose("Ignoring duplicate function signature for %s.%s(%s)", signature.SchemaName, signature.FunctionName, signature.Arguments)
				continue
			} else {
				functionSignatures[signature] = struct{}{}
			}
		}

		row.DatabaseOid = currentDatabaseOid
		row.Config = unpackPostgresStringArray(config)

		functions = append(functions, row)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return functions, nil
}

func GetFunctionStats(ctx context.Context, db *sql.DB, postgresVersion state.PostgresVersion, ignoreRegexp string) (functionStats state.PostgresFunctionStatsMap, err error) {
	var systemCatalogFilter string

	if postgresVersion.IsEPAS {
		systemCatalogFilter = relationSQLEPASSystemCatalogFilter
	} else {
		systemCatalogFilter = relationSQLdefaultSystemCatalogFilter
	}

	stmt, err := db.PrepareContext(ctx, QueryMarkerSQL+fmt.Sprintf(functionStatsSQL, systemCatalogFilter))
	if err != nil {
		err = fmt.Errorf("FunctionStats/Prepare: %s", err)
		return
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, ignoreRegexp)
	if err != nil {
		err = fmt.Errorf("FunctionStats/Query: %s", err)
		return
	}
	defer rows.Close()

	functionStats = make(state.PostgresFunctionStatsMap)
	for rows.Next() {
		var oid state.Oid
		var stats state.PostgresFunctionStats

		err = rows.Scan(&oid, &stats.Calls, &stats.TotalTime, &stats.SelfTime)
		if err != nil {
			err = fmt.Errorf("FunctionStats/Scan: %s", err)
			return
		}

		functionStats[oid] = stats
	}

	if err = rows.Err(); err != nil {
		err = fmt.Errorf("FunctionStats/Rows: %s", err)
		return
	}

	return
}

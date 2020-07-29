package postgres

import (
	"database/sql"
	"fmt"

	"github.com/guregu/null"
	"github.com/pganalyze/collector/state"
)

const functionsSQLDefaultKindFields = "pp.proisagg, pp.proiswindow"
const functionsSQLpg11KindFields = "pp.prokind = 'a', pp.prokind = 'w'"

const functionsSQL string = `
SELECT pp.oid,
			 pn.nspname,
			 pp.proname,
			 pl.lanname,
			 pp.prosrc,
			 pp.probin,
			 pp.proconfig,
			 pg_catalog.pg_get_function_arguments(pp.oid),
			 pg_catalog.pg_get_function_result(pp.oid),
			 %s,
			 pp.prosecdef,
			 pp.proleakproof,
			 pp.proisstrict,
			 pp.proretset,
			 pp.provolatile
	FROM pg_catalog.pg_proc pp
 INNER JOIN pg_catalog.pg_namespace pn ON (pp.pronamespace = pn.oid)
 INNER JOIN pg_catalog.pg_language pl ON (pp.prolang = pl.oid)
 WHERE pl.lanname NOT IN ('internal', 'c')
			 AND pn.nspname NOT IN ('pg_catalog', 'information_schema')
			 AND pp.proname NOT IN ('pg_stat_statements', 'pg_stat_statements_reset')
			 AND ($1 = '' OR (pn.nspname || '.' || pp.proname) !~* $1)
			 `

const functionStatsSQL string = `
SELECT funcid, calls, total_time, self_time
	FROM pg_stat_user_functions psuf
 INNER JOIN pg_catalog.pg_proc pp ON (psuf.funcid = pp.oid)
 INNER JOIN pg_catalog.pg_namespace pn ON (pp.pronamespace = pn.oid)
 WHERE ($1 = '' OR (pn.nspname || '.' || pp.proname) !~* $1)`

func GetFunctions(db *sql.DB, postgresVersion state.PostgresVersion, currentDatabaseOid state.Oid, ignoreRegexp string) ([]state.PostgresFunction, error) {
	var kindFields string

	if postgresVersion.Numeric >= state.PostgresVersion11 {
		kindFields = functionsSQLpg11KindFields
	} else {
		kindFields = functionsSQLDefaultKindFields
	}

	stmt, err := db.Prepare(QueryMarkerSQL + fmt.Sprintf(functionsSQL, kindFields))
	if err != nil {
		return nil, err
	}

	defer stmt.Close()

	rows, err := stmt.Query(ignoreRegexp)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var functions []state.PostgresFunction

	for rows.Next() {
		var row state.PostgresFunction
		var config null.String

		err := rows.Scan(&row.Oid, &row.SchemaName, &row.FunctionName, &row.Language, &row.Source,
			&row.SourceBin, &config, &row.Arguments, &row.Result, &row.Aggregate,
			&row.Window, &row.SecurityDefiner, &row.Leakproof, &row.Strict, &row.ReturnsSet, &row.Volatile)
		if err != nil {
			return nil, err
		}

		row.DatabaseOid = currentDatabaseOid
		row.Config = unpackPostgresStringArray(config)

		functions = append(functions, row)
	}

	return functions, nil
}

func GetFunctionStats(db *sql.DB, postgresVersion state.PostgresVersion, ignoreRegexp string) (functionStats state.PostgresFunctionStatsMap, err error) {
	stmt, err := db.Prepare(QueryMarkerSQL + functionStatsSQL)
	if err != nil {
		err = fmt.Errorf("FunctionStats/Prepare: %s", err)
		return
	}
	defer stmt.Close()

	rows, err := stmt.Query(ignoreRegexp)
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

	return
}

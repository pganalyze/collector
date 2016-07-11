package postgres

import (
	"database/sql"
	"fmt"

	"github.com/guregu/null"
	"github.com/pganalyze/collector/state"
)

const functionsSQL string = `
SELECT pp.oid,
			 pn.nspname,
			 pp.proname,
			 pl.lanname,
			 pp.prosrc,
			 pp.probin,
			 pp.proconfig,
			 pg_get_function_arguments(pp.oid),
			 pg_get_function_result(pp.oid),
			 pp.proisagg,
			 pp.proiswindow,
			 pp.prosecdef,
			 pp.proleakproof,
			 pp.proisstrict,
			 pp.proretset,
			 pp.provolatile
	FROM pg_proc pp
 INNER JOIN pg_namespace pn ON (pp.pronamespace = pn.oid)
 INNER JOIN pg_language pl ON (pp.prolang = pl.oid)
 WHERE pl.lanname != 'internal'
			 AND pn.nspname NOT IN ('pg_catalog', 'information_schema')
			 AND pp.proname NOT IN ('pg_stat_statements', 'pg_stat_statements_reset')`

const functionStatsSQL string = `
SELECT funcid, calls, total_time, self_time
	FROM pg_stat_user_functions
`

func GetFunctions(db *sql.DB, postgresVersion state.PostgresVersion, currentDatabaseOid state.Oid) ([]state.PostgresFunction, error) {
	stmt, err := db.Prepare(QueryMarkerSQL + functionsSQL)
	if err != nil {
		return nil, err
	}

	defer stmt.Close()

	rows, err := stmt.Query()
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

func GetFunctionStats(db *sql.DB, postgresVersion state.PostgresVersion) (functionStats state.PostgresFunctionStatsMap, err error) {
	stmt, err := db.Prepare(QueryMarkerSQL + functionStatsSQL)
	if err != nil {
		err = fmt.Errorf("FunctionStats/Prepare: %s", err)
		return
	}
	defer stmt.Close()

	rows, err := stmt.Query()
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

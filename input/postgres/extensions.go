package postgres

import (
	"database/sql"

	"github.com/pganalyze/collector/state"
)

const extensionsSQL string = `
SELECT extname,
       nspname,
	   extversion
  FROM pg_extension
  JOIN pg_namespace ON (pg_extension.extnamespace = pg_namespace.oid)
	 `

func GetExtensions(db *sql.DB, currentDatabaseOid state.Oid) ([]state.PostgresExtension, error) {
	stmt, err := db.Prepare(QueryMarkerSQL + extensionsSQL)
	if err != nil {
		return nil, err
	}

	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var extensions []state.PostgresExtension

	for rows.Next() {
		var e state.PostgresExtension
		e.DatabaseOid = currentDatabaseOid

		err := rows.Scan(&e.ExtensionName, &e.SchemaName, &e.Version)
		if err != nil {
			return nil, err
		}

		extensions = append(extensions, e)
	}

	return extensions, nil
}

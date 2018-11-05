package postgres

import (
	"database/sql"
	"fmt"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

const databasesSQLDefaultOptionalFields = "1"
const databasesSQLpg93OptionalFields = "datminmxid"

// See also https://www.postgresql.org/docs/9.5/static/catalog-pg-database.html
const databasesSQL string = `
SELECT oid,
			 datname,
			 datdba,
			 pg_catalog.pg_encoding_to_char(encoding),
			 datcollate,
			 datctype,
			 datistemplate,
			 datallowconn,
			 datconnlimit,
			 datfrozenxid,
			 %s
	FROM pg_catalog.pg_database`

func GetDatabases(logger *util.Logger, db *sql.DB, postgresVersion state.PostgresVersion) ([]state.PostgresDatabase, error) {
	var optionalFields string

	if postgresVersion.Numeric >= state.PostgresVersion93 {
		optionalFields = databasesSQLpg93OptionalFields
	} else {
		optionalFields = databasesSQLDefaultOptionalFields
	}

	stmt, err := db.Prepare(QueryMarkerSQL + fmt.Sprintf(databasesSQL, optionalFields))
	if err != nil {
		return nil, err
	}

	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var databases []state.PostgresDatabase

	for rows.Next() {
		var d state.PostgresDatabase

		err := rows.Scan(&d.Oid, &d.Name, &d.OwnerRoleOid, &d.Encoding, &d.Collate, &d.CType,
			&d.IsTemplate, &d.AllowConnections, &d.ConnectionLimit, &d.FrozenXID, &d.MinimumMultixactXID)
		if err != nil {
			return nil, err
		}

		databases = append(databases, d)
	}

	return databases, nil
}

package postgres

import (
	"database/sql"
	"fmt"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

// See also https://www.postgresql.org/docs/9.5/static/catalog-pg-database.html
const databasesSQL string = `
SELECT
	d.oid,
	d.datname,
	d.datdba,
	pg_catalog.pg_encoding_to_char(d.encoding),
	d.datcollate,
	d.datctype,
	d.datistemplate,
	d.datallowconn,
	d.datconnlimit,
	d.datfrozenxid,
	d.datminmxid,
	CASE WHEN d.datfrozenxid <> '0' THEN age(d.datfrozenxid) ELSE 0 END,
	CASE WHEN d.datminmxid <> '0' THEN mxid_age(d.datminmxid) ELSE 0 END,
	sd.xact_commit,
	sd.xact_rollback
FROM pg_catalog.pg_database d
	LEFT JOIN pg_catalog.pg_stat_database sd
	ON d.oid = sd.datid`

func GetDatabases(logger *util.Logger, db *sql.DB, postgresVersion state.PostgresVersion) ([]state.PostgresDatabase, state.PostgresDatabaseStatsMap, error) {
	stmt, err := db.Prepare(QueryMarkerSQL + fmt.Sprintf(databasesSQL))
	if err != nil {
		return nil, nil, err
	}

	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		return nil, nil, err
	}

	defer rows.Close()

	var databases []state.PostgresDatabase
	var databaseStats = make(state.PostgresDatabaseStatsMap)

	for rows.Next() {
		var d state.PostgresDatabase
		var ds state.PostgresDatabaseStats

		err := rows.Scan(&d.Oid, &d.Name, &d.OwnerRoleOid, &d.Encoding, &d.Collate, &d.CType,
			&d.IsTemplate, &d.AllowConnections, &d.ConnectionLimit, &d.FrozenXID, &d.MinimumMultixactXID,
			&ds.FrozenXIDAge, &ds.MinMXIDAge, &ds.XactCommit, &ds.XactRollback)
		if err != nil {
			return nil, nil, err
		}

		databases = append(databases, d)
		databaseStats[d.Oid] = ds
	}

	if err = rows.Err(); err != nil {
		return nil, nil, err
	}

	return databases, databaseStats, nil
}

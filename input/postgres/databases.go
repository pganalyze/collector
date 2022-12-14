package postgres

import (
	"database/sql"
	"fmt"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

const databasesSQLDefaultMxidFields = "1"
const databasesSQLpg93MxidFields = "d.datminmxid"
const databasesSQLDefaultMxidAgeFields = "0"
const databasesSQLpg93MxidAgeFields = "CASE WHEN d.datminmxid <> '0' THEN mxid_age(d.datminmxid) ELSE 0 END"

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
	%s,
	CASE WHEN d.datfrozenxid <> '0' THEN age(d.datfrozenxid) ELSE 0 END,
	%s,
	(sd.xact_commit + sd.xact_rollback) AS transaction_count
FROM pg_catalog.pg_database d
	LEFT JOIN pg_catalog.pg_stat_database sd
	ON d.oid = sd.datid`

func GetDatabases(logger *util.Logger, db *sql.DB, postgresVersion state.PostgresVersion) ([]state.PostgresDatabase, state.PostgresDatabaseStatsMap, error) {
	var mxidFields string
	var mxidAgeFields string

	if postgresVersion.Numeric >= state.PostgresVersion93 {
		mxidFields = databasesSQLpg93MxidFields
		mxidAgeFields = databasesSQLpg93MxidAgeFields
	} else {
		mxidFields = databasesSQLDefaultMxidFields
		mxidAgeFields = databasesSQLDefaultMxidAgeFields
	}

	stmt, err := db.Prepare(QueryMarkerSQL + fmt.Sprintf(databasesSQL, mxidFields, mxidAgeFields))
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
			&d.XIDAge, &d.MXIDAge, &ds.TransactionCount)
		if err != nil {
			return nil, nil, err
		}

		databases = append(databases, d)
		databaseStats[d.Oid] = ds
	}

	return databases, databaseStats, nil
}

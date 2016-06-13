package postgres

import (
	"database/sql"
	"fmt"

	"gopkg.in/guregu/null.v2"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

const rolesSQLDefaultOptionalFields = "rolsuper"
const rolesSQLpg95OptionalFields = "rolbypassrls"

// See also https://www.postgresql.org/docs/9.5/static/catalog-pg-database.html
const rolesSQL string = `
SELECT oid,
			 rolname,
			 rolinherit,
			 rolcanlogin,
			 rolcreaterole,
			 rolcreatedb,
			 rolsuper,
			 rolreplication,
			 rolconnlimit,
			 rolvaliduntil,
			 rolconfig,
			 (SELECT array_agg(roleid) FROM pg_auth_members WHERE pg_roles.oid = pg_auth_members.member) AS member_of,
			 %s
	FROM pg_roles
	 `

func GetRoles(logger *util.Logger, db *sql.DB, postgresVersion state.PostgresVersion) ([]state.PostgresRole, error) {
	var optionalFields string

	if postgresVersion.Numeric >= state.PostgresVersion95 {
		optionalFields = rolesSQLpg95OptionalFields
	} else {
		optionalFields = rolesSQLDefaultOptionalFields
	}

	stmt, err := db.Prepare(QueryMarkerSQL + fmt.Sprintf(rolesSQL, optionalFields))
	if err != nil {
		return nil, err
	}

	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var roles []state.PostgresRole

	for rows.Next() {
		var r state.PostgresRole
		var config, memberOf null.String

		err := rows.Scan(&r.Oid, &r.Name, &r.Inherit, &r.Login, &r.CreateRole, &r.CreateDb, &r.SuperUser,
			&r.Replication, &r.ConnectionLimit, &r.PasswordValidUntil, &config, &memberOf, &r.BypassRLS)
		if err != nil {
			return nil, err
		}

		r.Config = unpackPostgresStringArray(config)
		r.MemberOf = unpackPostgresOidArray(memberOf)

		roles = append(roles, r)
	}

	return roles, nil
}

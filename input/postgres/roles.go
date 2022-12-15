package postgres

import (
	"context"
	"database/sql"

	"github.com/guregu/null"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

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
			 CASE WHEN rolvaliduntil = 'infinity' THEN NULL ELSE rolvaliduntil END,
			 rolconfig,
			 (SELECT pg_catalog.array_agg(roleid) FROM pg_auth_members am WHERE r.oid = am.member) AS member_of,
			 rolbypassrls
	FROM pg_roles r
	 `

func GetRoles(ctx context.Context, logger *util.Logger, db *sql.DB, postgresVersion state.PostgresVersion) ([]state.PostgresRole, error) {
	stmt, err := db.PrepareContext(ctx, QueryMarkerSQL+rolesSQL)
	if err != nil {
		return nil, err
	}

	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx)
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

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return roles, nil
}

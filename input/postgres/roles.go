package postgres

import (
	"context"
	"database/sql"

	"github.com/guregu/null"
	"github.com/pganalyze/collector/state"
)

// See also https://www.postgresql.org/docs/current/static/catalog-pg-database.html
const rolesSQL string = `
SELECT oid,
			 rolname,
			 rolinherit,
			 rolcanlogin,
			 rolcreaterole,
			 rolcreatedb,
			 rolsuper,
			 COALESCE((SELECT pg_has_role(r.oid, r2.oid, 'MEMBER') FROM pg_roles r2 WHERE rolname = $1), false) AS cloud_superuser,
			 COALESCE((SELECT pg_has_role(r.oid, r2.oid, 'MEMBER') FROM pg_roles r2 WHERE rolname = 'pg_monitor'), false) AS monitoring_user,
			 rolreplication,
			 rolconnlimit,
			 CASE WHEN rolvaliduntil = 'infinity' THEN NULL ELSE rolvaliduntil END,
			 rolconfig,
			 (SELECT pg_catalog.array_agg(roleid) FROM pg_auth_members am WHERE r.oid = am.member) AS member_of,
			 rolbypassrls
	FROM pg_roles r
	 `

func getRoles(ctx context.Context, db *sql.DB, systemType string) ([]state.PostgresRole, error) {
	cloudSuperuserName := ""
	if systemType == "amazon_rds" {
		cloudSuperuserName = "rds_superuser"
	} else if systemType == "azure_database" {
		cloudSuperuserName = "azure_pg_admin"
	} else if systemType == "google_cloudsql" {
		cloudSuperuserName = "cloudsqlsuperuser"
	}

	rows, err := db.QueryContext(ctx, QueryMarkerSQL+rolesSQL, cloudSuperuserName)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var roles []state.PostgresRole

	for rows.Next() {
		var r state.PostgresRole
		var config, memberOf null.String

		err := rows.Scan(&r.Oid, &r.Name, &r.Inherit, &r.Login, &r.CreateRole, &r.CreateDb, &r.SuperUser,
			&r.CloudSuperUser, &r.MonitoringUser, &r.Replication, &r.ConnectionLimit, &r.PasswordValidUntil,
			&config, &memberOf, &r.BypassRLS)
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

package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/pganalyze/collector/state"
)

const typesSQL string = `
SELECT t.oid,
       t.typarray AS arrayoid,
       n.nspname AS schema,
       t.typname AS name,
       t.typtype AS type,
       CASE WHEN t.typtype = 'd' THEN pg_catalog.format_type(t.typbasetype, t.typtypmod) END AS domain_type,
       t.typnotnull AS domain_not_null,
       t.typdefault AS domain_default,
       COALESCE(
         CASE t.typtype
           WHEN 'd' THEN
             (SELECT pg_catalog.json_agg(pg_catalog.pg_get_constraintdef(oid, FALSE)) FROM pg_catalog.pg_constraint WHERE contypid = t.oid)::text
           WHEN 'e' THEN
             (SELECT pg_catalog.json_agg(enumlabel ORDER BY enumsortorder) FROM pg_catalog.pg_enum WHERE enumtypid = t.oid)::text
           WHEN 'c' THEN
             (SELECT pg_catalog.json_agg(ARRAY[attname, pg_catalog.format_type(atttypid, atttypmod)]) FROM pg_catalog.pg_attribute WHERE attrelid = t.typrelid)::text
         END
       , '[]') AS json
  FROM pg_catalog.pg_type t
 INNER JOIN pg_catalog.pg_namespace n ON n.oid = t.typnamespace
 WHERE t.typtype <> 'b'
    AND (t.typrelid = 0 OR (SELECT c.relkind = 'c' FROM pg_catalog.pg_class c WHERE c.oid = t.typrelid))
    AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_type el WHERE el.oid = t.typelem AND el.typarray = t.oid)
	AND t.oid NOT IN (SELECT pd.objid FROM pg_catalog.pg_depend pd WHERE pd.deptype = 'e' AND pd.classid = 'pg_catalog.pg_type'::regclass)
    AND %s
`

func GetTypes(ctx context.Context, db *sql.DB, postgresVersion state.PostgresVersion, currentDatabaseOid state.Oid) ([]state.PostgresType, error) {
	var systemCatalogFilter string
	if postgresVersion.IsEPAS {
		systemCatalogFilter = relationSQLepasSystemCatalogFilter
	} else {
		systemCatalogFilter = relationSQLdefaultSystemCatalogFilter
	}

	stmt, err := db.PrepareContext(ctx, QueryMarkerSQL+fmt.Sprintf(typesSQL, systemCatalogFilter))
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var types []state.PostgresType

	for rows.Next() {
		var t state.PostgresType
		var arrayString string
		t.DatabaseOid = currentDatabaseOid

		err := rows.Scan(
			&t.Oid, &t.ArrayOid, &t.SchemaName, &t.Name, &t.Type, &t.DomainType, &t.DomainNotNull, &t.DomainDefault, &arrayString)

		if err != nil {
			return nil, err
		}

		if t.Type == "d" {
			json.Unmarshal([]byte(arrayString), &t.DomainConstraints)
		}
		if t.Type == "e" {
			json.Unmarshal([]byte(arrayString), &t.EnumValues)
		}
		if t.Type == "c" {
			json.Unmarshal([]byte(arrayString), &t.CompositeAttrs)
		}

		types = append(types, t)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return types, nil
}

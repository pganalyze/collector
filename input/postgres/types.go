package postgres

import (
  "database/sql"

  "github.com/pganalyze/collector/state"
)

const typesSQL string = `
SELECT t.oid,
       n.nspname AS schema,
       pg_catalog.format_type(t.oid, null) AS name,
       t.typtype AS type,
       CASE WHEN t.typtype = 'd' THEN pg_catalog.format_type(t.typbasetype, t.typtypmod) ELSE null END AS underlying_type,
       t.typnotnull AS not_null,
       t.typdefault AS default,
       (
           SELECT pg_get_constraintdef(oid)
           FROM pg_constraint WHERE contypid = t.oid
       ) AS constraint
      -- (
      --     SELECT array_agg(enumlabel ORDER BY enumsortorder)
      --     FROM pg_enum WHERE enumtypid = t.oid
      -- ) AS values,
      -- (
      --     SELECT array_agg(array[attname, pg_catalog.format_type(atttypid, atttypmod)])
      --     FROM pg_attribute WHERE attrelid = t.typrelid
      -- ) AS attrs
  FROM pg_catalog.pg_type t
 INNER JOIN pg_catalog.pg_namespace n ON n.oid = t.typnamespace
 WHERE t.typtype <> 'b'
    AND (t.typrelid = 0 OR (SELECT c.relkind = 'c' FROM pg_catalog.pg_class c WHERE c.oid = t.typrelid))
    AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_type el WHERE el.oid = t.typelem AND el.typarray = t.oid)
    AND n.nspname <> 'pg_catalog'
    AND n.nspname <> 'information_schema'
    AND n.nspname !~ '^pg_toast'
`

func GetTypes(db *sql.DB, postgresVersion state.PostgresVersion, currentDatabaseOid state.Oid) ([]state.PostgresType, error) {
  stmt, err := db.Prepare(QueryMarkerSQL + typesSQL)
  if err != nil {
    return nil, err
  }
  defer stmt.Close()

  rows, err := stmt.Query()
  if err != nil {
    return nil, err
  }
  defer rows.Close()

  var types []state.PostgresType

  for rows.Next() {
    var t state.PostgresType
    t.DatabaseOid = currentDatabaseOid

    // TODO: unpackPostgresStringArray

    err := rows.Scan(
      &t.Oid, &t.SchemaName, &t.Name, &t.Type, &t.UnderlyingType, &t.NotNull, &t.Default, &t.Constraint/*, &t.EnumValues, &t.CompositeAttrs*/)

    if err != nil {
      return nil, err
    }

    types = append(types, t)
  }

  return types, nil
}

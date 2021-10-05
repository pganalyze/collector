package postgres

import (
  "database/sql"
  "encoding/json"

  "github.com/pganalyze/collector/state"
)

const typesSQL string = `
SELECT t.oid,
       n.nspname AS schema,
       t.typname AS name,
       t.typtype AS type,
       CASE WHEN t.typtype = 'd' THEN pg_catalog.format_type(t.typbasetype, t.typtypmod) END AS domain_type,
       t.typnotnull AS domain_not_null,
       t.typdefault AS domain_default,
       CASE WHEN t.typtype = 'd' THEN
           (SELECT pg_get_constraintdef(oid) FROM pg_constraint WHERE contypid = t.oid)
       END AS domain_constraint,
       CASE t.typtype
       WHEN 'e' THEN
           (SELECT json_agg(enumlabel ORDER BY enumsortorder) FROM pg_enum WHERE enumtypid = t.oid)
       WHEN 'c' THEN
           (SELECT json_agg(array[attname, pg_catalog.format_type(atttypid, atttypmod)]) FROM pg_attribute WHERE attrelid = t.typrelid)
       END AS json
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
    var arrayString string
    t.DatabaseOid = currentDatabaseOid

    err := rows.Scan(
      &t.Oid, &t.SchemaName, &t.Name, &t.Type, &t.DomainType, &t.DomainNotNull, &t.DomainDefault, &t.DomainConstraint, &arrayString)

    if err != nil {
      return nil, err
    }

    if t.Type == "e" {
      json.Unmarshal([]byte(arrayString), &t.EnumValues)
    }
    if t.Type == "c" {
      json.Unmarshal([]byte(arrayString), &t.CompositeAttrs)
    }

    types = append(types, t)
  }

  return types, nil
}

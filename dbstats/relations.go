package dbstats

import (
  "database/sql"
)

type Relation struct {
  Oid int64 `json:"oid"`
  SchemaName string `json:"schema_name"`
  TableName string `json:"table_name"`
  SizeBytes int64 `json:"size_bytes"`
  RelationType string `json:"relation_type"`
}

const relationsSQL string =
`SELECT c.oid,
        n.nspname AS schema_name,
        c.relname AS table_name,
        pg_catalog.pg_table_size(c.oid) AS size_bytes,
        c.relkind AS relation_type
   FROM pg_catalog.pg_class c
   LEFT JOIN pg_catalog.pg_namespace n ON (n.oid = c.relnamespace)
   LEFT JOIN pg_catalog.pg_stat_user_tables s ON (s.relid = c.oid)
   LEFT JOIN pg_catalog.pg_statio_user_tables sio ON (sio.relid = c.oid)
  WHERE c.relkind IN ('r','v','m')
        AND c.relpersistence <> 't'
        AND c.relname NOT IN ('pg_stat_statements')
        AND n.nspname NOT IN ('pg_catalog', 'information_schema')`

// s.*, sio.*

const columnsSQL string =
`SELECT c.oid,
        a.attname AS name,
        pg_catalog.format_type(a.atttypid, a.atttypmod) AS data_type,
   (SELECT pg_catalog.pg_get_expr(d.adbin, d.adrelid)
    FROM pg_catalog.pg_attrdef d
    WHERE d.adrelid = a.attrelid
      AND d.adnum = a.attnum
      AND a.atthasdef) AS default_value,
        a.attnotnull AS not_null,
        a.attnum AS position
 FROM pg_catalog.pg_class c
 LEFT JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
 LEFT JOIN pg_catalog.pg_attribute a ON c.oid = a.attrelid
 WHERE c.relkind IN ('r','v','m')
       AND c.relpersistence <> 't'
       AND c.relname NOT IN ('pg_stat_statements')
       AND n.nspname NOT IN ('pg_catalog', 'information_schema')
       AND a.attnum > 0
       AND NOT a.attisdropped
 ORDER BY a.attnum`

func GetRelations(db *sql.DB) []Relation {
  stmt, err := db.Prepare(relationsSQL)
  checkErr(err)

  defer stmt.Close()

  rows, err := stmt.Query()

  var relations []Relation

  defer rows.Close()
  for rows.Next() {
    var row Relation

    err := rows.Scan(&row.Oid, &row.SchemaName, &row.TableName, &row.SizeBytes,
                     &row.RelationType)
    checkErr(err)

    relations = append(relations, row)
  }

  return relations
}

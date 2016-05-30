package postgres

import (
	"database/sql"
	"fmt"

	"github.com/pganalyze/collector/state"
)

const relationsSQL string = `SELECT c.oid,
				n.nspname AS schema_name,
				c.relname AS table_name,
				c.relkind AS relation_type
	 FROM pg_catalog.pg_class c
	 LEFT JOIN pg_catalog.pg_namespace n ON (n.oid = c.relnamespace)
	WHERE c.relkind IN ('r','v','m')
				AND c.relpersistence <> 't'
				AND c.relname NOT IN ('pg_stat_statements')
				AND n.nspname NOT IN ('pg_catalog','pg_toast','information_schema')`

const columnsSQL string = `SELECT c.oid,
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
			 AND n.nspname NOT IN ('pg_catalog','pg_toast','information_schema')
			 AND a.attnum > 0
			 AND NOT a.attisdropped
 ORDER BY a.attnum`

const indicesSQL string = `
SELECT c.oid,
			 c2.oid AS index_oid,
			 i.indkey::text AS columns,
			 c2.relname AS name,
			 i.indisprimary AS is_primary,
			 i.indisunique AS is_unique,
			 i.indisvalid AS is_valid,
			 pg_catalog.pg_get_indexdef(i.indexrelid, 0, TRUE) AS index_def,
			 pg_catalog.pg_get_constraintdef(con.oid, TRUE) AS constraint_def
	FROM pg_catalog.pg_class c
	JOIN pg_catalog.pg_namespace n ON (n.oid = c.relnamespace)
	JOIN pg_catalog.pg_index i ON (c.oid = i.indrelid)
	JOIN pg_catalog.pg_class c2 ON (i.indexrelid = c2.oid)
	LEFT JOIN pg_catalog.pg_constraint con ON (conrelid = i.indrelid
																						 AND conindid = i.indexrelid
																						 AND contype IN ('p', 'u', 'x'))
 WHERE c.relkind IN ('r','v','m')
			 AND c.relpersistence <> 't'
			 AND n.nspname NOT IN ('pg_catalog','pg_toast','information_schema')`

// FIXME: This misses check constraints and others
const constraintsSQL string = `
SELECT c.oid,
			 conname AS name,
			 pg_catalog.pg_get_constraintdef(r.oid, TRUE) AS constraint_def,
			 r.conkey AS columns,
			 n2.nspname AS foreign_schema,
			 c2.relname AS foreign_table,
			 r.confkey AS foreign_columns
	FROM pg_catalog.pg_class c
			 LEFT JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
			 LEFT JOIN pg_catalog.pg_constraint r ON r.conrelid = c.oid
			 LEFT JOIN pg_catalog.pg_class c2 ON r.confrelid = c2.oid
			 LEFT JOIN pg_catalog.pg_namespace n2 ON n2.oid = c2.relnamespace
WHERE r.contype = 'f'
			 AND n.nspname NOT IN ('pg_catalog','pg_toast','information_schema')`

const viewDefinitionSQL string = `
SELECT c.oid,
			 pg_catalog.pg_get_viewdef(c.oid) AS view_definition
	FROM pg_catalog.pg_class c
	LEFT JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
	WHERE c.relkind IN ('v','m')
			 AND c.relpersistence <> 't'
			 AND c.relname NOT IN ('pg_stat_statements')
			 AND n.nspname NOT IN ('pg_catalog','pg_toast','information_schema')`

func GetRelations(db *sql.DB, postgresVersion state.PostgresVersion) ([]state.PostgresRelation, error) {
	relations := make(map[state.Oid]state.PostgresRelation, 0)

	// Relations
	stmt, err := db.Prepare(QueryMarkerSQL + relationsSQL)
	if err != nil {
		err = fmt.Errorf("Relations/Prepare: %s", err)
		return nil, err
	}

	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		err = fmt.Errorf("Relations/Query: %s", err)
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var row state.PostgresRelation

		err = rows.Scan(&row.Oid, &row.SchemaName, &row.RelationName, &row.RelationType)
		if err != nil {
			err = fmt.Errorf("Relations/Scan: %s", err)
			return nil, err
		}

		relations[row.Oid] = row
	}

	// Columns
	stmt, err = db.Prepare(QueryMarkerSQL + columnsSQL)
	if err != nil {
		err = fmt.Errorf("Columns/Prepare: %s", err)
		return nil, err
	}

	defer stmt.Close()

	rows, err = stmt.Query()
	if err != nil {
		err = fmt.Errorf("Columns/Query: %s", err)
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var row state.PostgresColumn

		err = rows.Scan(&row.RelationOid, &row.Name, &row.DataType, &row.DefaultValue,
			&row.NotNull, &row.Position)
		if err != nil {
			err = fmt.Errorf("Columns/Scan: %s", err)
			return nil, err
		}

		relation := relations[row.RelationOid]
		relation.Columns = append(relation.Columns, row)
		relations[row.RelationOid] = relation
	}

	// Indices
	stmt, err = db.Prepare(QueryMarkerSQL + indicesSQL)
	if err != nil {
		err = fmt.Errorf("Indices/Prepare: %s", err)
		return nil, err
	}

	defer stmt.Close()

	rows, err = stmt.Query()
	if err != nil {
		err = fmt.Errorf("Indices/Query: %s", err)
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var row state.PostgresIndex

		err = rows.Scan(&row.RelationOid, &row.IndexOid, &row.Columns, &row.Name,
			&row.IsPrimary, &row.IsUnique, &row.IsValid, &row.IndexDef, &row.ConstraintDef)
		if err != nil {
			err = fmt.Errorf("Indices/Scan: %s", err)
			return nil, err
		}

		relation := relations[row.RelationOid]
		relation.Indices = append(relation.Indices, row)
		relations[row.RelationOid] = relation
	}

	// Constraints
	stmt, err = db.Prepare(QueryMarkerSQL + constraintsSQL)
	if err != nil {
		err = fmt.Errorf("Constraints/Prepare: %s", err)
		return nil, err
	}

	defer stmt.Close()

	rows, err = stmt.Query()
	if err != nil {
		err = fmt.Errorf("Constraints/Query: %s", err)
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var row state.PostgresConstraint

		err = rows.Scan(&row.RelationOid, &row.Name, &row.ConstraintDef, &row.Columns,
			&row.ForeignSchema, &row.ForeignTable, &row.ForeignColumns)
		if err != nil {
			err = fmt.Errorf("Constraints/Scan: %s", err)
			return nil, err
		}

		relation := relations[row.RelationOid]
		relation.Constraints = append(relation.Constraints, row)
		relations[row.RelationOid] = relation
	}

	// View definitions
	stmt, err = db.Prepare(QueryMarkerSQL + viewDefinitionSQL)
	if err != nil {
		err = fmt.Errorf("Views/Prepare: %s", err)
		return nil, err
	}

	defer stmt.Close()

	rows, err = stmt.Query()
	if err != nil {
		err = fmt.Errorf("Views/Query: %s", err)
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var relationOid state.Oid
		var viewDefinition string

		err := rows.Scan(&relationOid, &viewDefinition)
		if err != nil {
			err = fmt.Errorf("Views/Scan: %s", err)
			return nil, err
		}

		relation := relations[relationOid]
		relation.ViewDefinition = viewDefinition
		relations[relationOid] = relation
	}

	// Flip the oid-based map into an array
	v := make([]state.PostgresRelation, 0, len(relations))
	for _, value := range relations {
		v = append(v, value)
	}

	return v, nil
}

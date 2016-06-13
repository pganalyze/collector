package postgres

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/guregu/null.v2"

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
			 c2.oid,
			 i.indkey::text,
			 c2.relname,
			 i.indisprimary,
			 i.indisunique,
			 i.indisvalid,
			 pg_catalog.pg_get_indexdef(i.indexrelid, 0, TRUE),
			 pg_catalog.pg_get_constraintdef(con.oid, TRUE)
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

const constraintsSQL string = `
SELECT c.oid,
			 conname,
			 contype,
			 pg_catalog.pg_get_constraintdef(r.oid, TRUE),
			 conkey,
			 confrelid,
			 confkey,
			 confupdtype,
			 confdeltype,
			 confmatchtype
	FROM pg_catalog.pg_constraint r
			 JOIN pg_catalog.pg_class c ON r.conrelid = c.oid
			 JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname NOT IN ('pg_catalog','pg_toast','information_schema')`

const viewDefinitionSQL string = `
SELECT c.oid,
			 pg_catalog.pg_get_viewdef(c.oid) AS view_definition
	FROM pg_catalog.pg_class c
	LEFT JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
	WHERE c.relkind IN ('v','m')
			 AND c.relpersistence <> 't'
			 AND c.relname NOT IN ('pg_stat_statements')
			 AND n.nspname NOT IN ('pg_catalog','pg_toast','information_schema')`

func GetRelations(db *sql.DB, postgresVersion state.PostgresVersion, currentDatabaseOid state.Oid) ([]state.PostgresRelation, error) {
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

		row.DatabaseOid = currentDatabaseOid

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
		var columns string

		err = rows.Scan(&row.RelationOid, &row.IndexOid, &columns, &row.Name,
			&row.IsPrimary, &row.IsUnique, &row.IsValid, &row.IndexDef, &row.ConstraintDef)
		if err != nil {
			err = fmt.Errorf("Indices/Scan: %s", err)
			return nil, err
		}

		for _, cstr := range strings.Split(columns, " ") {
			cint, _ := strconv.Atoi(cstr)
			row.Columns = append(row.Columns, int32(cint))
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
		var columns, foreignColumns null.String
		var foreignUpdateType, foreignDeleteType, foreignMatchType string

		err = rows.Scan(&row.RelationOid, &row.Name, &row.Type, &row.ConstraintDef,
			&columns, &row.ForeignOid, &foreignColumns, &foreignUpdateType,
			&foreignDeleteType, &foreignMatchType)
		if err != nil {
			err = fmt.Errorf("Constraints/Scan: %s", err)
			return nil, err
		}

		if foreignUpdateType != " " {
			row.ForeignUpdateType = foreignUpdateType
		}
		if foreignDeleteType != " " {
			row.ForeignDeleteType = foreignDeleteType
		}
		if foreignMatchType != " " {
			row.ForeignMatchType = foreignMatchType
		}
		if columns.Valid {
			for _, cstr := range strings.Split(strings.Trim(columns.String, "{}"), ",") {
				cint, _ := strconv.Atoi(cstr)
				row.Columns = append(row.Columns, int32(cint))
			}
		}
		if foreignColumns.Valid {
			for _, cstr := range strings.Split(strings.Trim(foreignColumns.String, "{}"), ",") {
				cint, _ := strconv.Atoi(cstr)
				row.ForeignColumns = append(row.ForeignColumns, int32(cint))
			}
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

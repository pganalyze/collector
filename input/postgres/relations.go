package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/guregu/null"
	"github.com/pganalyze/collector/state"
)

const relationsSQLOidField = "c.relhasoids AS relation_has_oids"
const relationsSQLpg12OidField = "false AS relation_has_oids"

const relationsSQL string = `
	 WITH locked_relids AS (SELECT DISTINCT relation relid FROM pg_catalog.pg_locks WHERE mode = 'AccessExclusiveLock' AND relation IS NOT NULL)
 SELECT c.oid,
				n.nspname AS schema_name,
				c.relname AS table_name,
				c.relkind AS relation_type,
				c.reloptions AS relation_options,
				%s,
				c.relpersistence AS relation_persistence,
				c.relhassubclass AS relation_has_inheritance_children,
				c.reltoastrelid IS NOT NULL AS relation_has_toast,
				c.relfrozenxid AS relation_frozen_xid,
				c.relminmxid AS relation_min_mxid,
				COALESCE((SELECT inhparent FROM pg_inherits WHERE inhrelid = c.oid ORDER BY inhseqno LIMIT 1), 0) AS parent_relid,
				COALESCE(pg_get_expr(c.relpartbound, c.oid, true), '') AS partition_boundary,
				COALESCE((SELECT p.partstrat FROM pg_partitioned_table p WHERE p.partrelid = c.oid), '') AS partition_strategy,
				(SELECT p.partattrs FROM pg_partitioned_table p WHERE p.partrelid = c.oid) AS partition_columns,
				COALESCE(pg_catalog.pg_get_partkeydef(c.oid), '') AS partition_expr,
				locked_relids.relid IS NOT NULL AS exclusively_locked
	 FROM pg_catalog.pg_class c
	 LEFT JOIN pg_catalog.pg_namespace n ON (n.oid = c.relnamespace)
	 LEFT JOIN locked_relids ON (c.oid = locked_relids.relid)
	WHERE c.relkind IN ('r','v','m','p')
				AND c.relpersistence <> 't'
				AND c.oid NOT IN (SELECT pd.objid FROM pg_catalog.pg_depend pd WHERE pd.deptype = 'e' AND pd.classid = 'pg_catalog.pg_class'::regclass)
				AND n.nspname NOT IN ('pg_catalog','pg_toast','information_schema')
				AND ($1 = '' OR (n.nspname || '.' || c.relname) !~* $1)`

const columnsSQL string = `
	 WITH locked_relids AS (SELECT DISTINCT relation relid FROM pg_catalog.pg_locks WHERE mode = 'AccessExclusiveLock' AND relation IS NOT NULL)
 SELECT c.oid,
				a.attname AS name,
				pg_catalog.format_type(a.atttypid, a.atttypmod) AS data_type,
	 (SELECT pg_catalog.pg_get_expr(d.adbin, d.adrelid)
		FROM pg_catalog.pg_attrdef d
		WHERE d.adrelid = a.attrelid
			AND d.adnum = a.attnum
			AND a.atthasdef) AS default_value,
				a.attnotnull AS not_null,
				a.attnum AS position,
				a.atttypid as type_oid,
				false AS exclusively_locked
 FROM pg_catalog.pg_class c
 LEFT JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
 LEFT JOIN pg_catalog.pg_attribute a ON c.oid = a.attrelid
 WHERE c.relkind IN ('r','v','m','p')
			 AND c.relpersistence <> 't'
			 AND c.oid NOT IN (SELECT pd.objid FROM pg_catalog.pg_depend pd WHERE pd.deptype = 'e' AND pd.classid = 'pg_catalog.pg_class'::regclass)
			 AND n.nspname NOT IN ('pg_catalog','pg_toast','information_schema')
			 AND a.attnum > 0
			 AND NOT a.attisdropped
			 AND c.oid NOT IN (SELECT relid FROM locked_relids)
			 AND ($1 = '' OR (n.nspname || '.' || c.relname) !~* $1)
 UNION ALL
 SELECT relid,
		'',
		'',
		NULL,
		false,
		0,
		0,
		true AS exclusively_locked
   FROM locked_relids`

const indicesSQL string = `
	WITH locked_relids AS (SELECT DISTINCT relation relid FROM pg_catalog.pg_locks WHERE mode = 'AccessExclusiveLock' AND relation IS NOT NULL)
SELECT c.oid,
			 c2.oid,
			 i.indkey::text,
			 c2.relname,
			 i.indisprimary,
			 i.indisunique,
			 i.indisvalid,
			 pg_catalog.pg_get_indexdef(i.indexrelid, 0, FALSE),
			 pg_catalog.pg_get_constraintdef(con.oid, FALSE),
			 c2.reloptions,
			 (SELECT a.amname FROM pg_catalog.pg_am a JOIN pg_catalog.pg_opclass o ON (a.oid = o.opcmethod) WHERE o.oid = i.indclass[0]),
			 false AS exclusively_locked
	FROM pg_catalog.pg_class c
	JOIN pg_catalog.pg_namespace n ON (n.oid = c.relnamespace)
	JOIN pg_catalog.pg_index i ON (c.oid = i.indrelid)
	JOIN pg_catalog.pg_class c2 ON (i.indexrelid = c2.oid)
	LEFT JOIN pg_catalog.pg_constraint con ON (conrelid = i.indrelid
																						 AND conindid = i.indexrelid
																						 AND contype IN ('p', 'u', 'x'))
 WHERE c.relkind IN ('r','v','m','p')
			 AND c.relpersistence <> 't'
			 AND c.oid NOT IN (SELECT pd.objid FROM pg_catalog.pg_depend pd WHERE pd.deptype = 'e' AND pd.classid = 'pg_catalog.pg_class'::regclass)
			 AND n.nspname NOT IN ('pg_catalog','pg_toast','information_schema')
			 AND c.oid NOT IN (SELECT relid FROM locked_relids)
			 AND c2.oid NOT IN (SELECT relid FROM locked_relids)
			 AND ($1 = '' OR (n.nspname || '.' || c.relname) !~* $1)
 UNION ALL
 SELECT relid,
		0,
		'',
		'',
		false,
		false,
		false,
		'',
		NULL,
		NULL,
		'',
		true AS exclusively_locked
  FROM locked_relids
`

const constraintsSQL string = `
	WITH locked_relids AS (SELECT DISTINCT relation relid FROM pg_catalog.pg_locks WHERE mode = 'AccessExclusiveLock' AND relation IS NOT NULL)
SELECT c.oid,
			 conname,
			 contype,
			 pg_catalog.pg_get_constraintdef(r.oid, FALSE),
			 conkey,
			 confrelid,
			 confkey,
			 confupdtype,
			 confdeltype,
			 confmatchtype,
			 false AS exclusively_locked
	FROM pg_catalog.pg_constraint r
			 JOIN pg_catalog.pg_class c ON (r.conrelid = c.oid)
			 JOIN pg_catalog.pg_namespace n ON (n.oid = c.relnamespace)
WHERE c.relkind IN ('r','v','m','p')
			AND c.relpersistence <> 't'
			AND c.oid NOT IN (SELECT pd.objid FROM pg_catalog.pg_depend pd WHERE pd.deptype = 'e' AND pd.classid = 'pg_catalog.pg_class'::regclass)
			AND n.nspname NOT IN ('pg_catalog','pg_toast','information_schema')
			AND c.oid NOT IN (SELECT relid FROM locked_relids)
			AND ($1 = '' OR (n.nspname || '.' || c.relname) !~* $1)
UNION ALL
SELECT relid,
	   '',
	   '',
	   '',
	   NULL,
	   0,
	   NULL,
	   '',
	   '',
	   '',
	   true AS exclusively_locked
  FROM locked_relids
`

const viewDefinitionSQL string = `
	WITH locked_relids AS (SELECT DISTINCT relation relid FROM pg_catalog.pg_locks WHERE mode = 'AccessExclusiveLock' AND relation IS NOT NULL)
SELECT c.oid,
			 pg_catalog.pg_get_viewdef(c.oid) AS view_definition,
			 false AS exclusively_locked
	FROM pg_catalog.pg_class c
	LEFT JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
	WHERE c.relkind IN ('v','m')
			 AND c.relpersistence <> 't'
			 AND c.oid NOT IN (SELECT pd.objid FROM pg_catalog.pg_depend pd WHERE pd.deptype = 'e' AND pd.classid = 'pg_catalog.pg_class'::regclass)
			 AND n.nspname NOT IN ('pg_catalog','pg_toast','information_schema')
			 AND c.oid NOT IN (SELECT relid FROM locked_relids)
			 AND ($1 = '' OR (n.nspname || '.' || c.relname) !~* $1)
UNION ALL
SELECT relid,
	   '',
	   true AS exclusively_locked
  FROM locked_relids
`

func GetRelations(ctx context.Context, db *sql.DB, postgresVersion state.PostgresVersion, currentDatabaseOid state.Oid, ignoreRegexp string) ([]state.PostgresRelation, error) {
	relations := make(map[state.Oid]state.PostgresRelation, 0)

	// Relations
	var oidField string

	if postgresVersion.Numeric >= state.PostgresVersion12 {
		oidField = relationsSQLpg12OidField
	} else {
		oidField = relationsSQLOidField
	}

	rows, err := db.QueryContext(ctx, QueryMarkerSQL+fmt.Sprintf(relationsSQL, oidField), ignoreRegexp)
	if err != nil {
		err = fmt.Errorf("Relations/Query: %s", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var row state.PostgresRelation
		var options null.String
		var partCols null.String

		err = rows.Scan(&row.Oid, &row.SchemaName, &row.RelationName, &row.RelationType,
			&options, &row.HasOids, &row.PersistenceType, &row.HasInheritanceChildren,
			&row.HasToast, &row.FrozenXID, &row.MinimumMultixactXID, &row.ParentTableOid,
			&row.PartitionBoundary, &row.PartitionStrategy, &partCols, &row.PartitionedBy,
			&row.ExclusivelyLocked)
		if err != nil {
			err = fmt.Errorf("Relations/Scan: %s", err)
			return nil, err
		}

		row.Options = make(map[string]string)
		if options.Valid {
			for _, cstr := range strings.Split(strings.Trim(options.String, "{}"), ",") {
				parts := strings.Split(cstr, "=")
				row.Options[parts[0]] = parts[1]
			}
		}

		if partCols.Valid {
			for _, cstr := range strings.Split(partCols.String, " ") {
				cint, _ := strconv.ParseInt(cstr, 10, 32)
				row.PartitionColumns = append(row.PartitionColumns, int32(cint))
			}
		}

		row.DatabaseOid = currentDatabaseOid

		relations[row.Oid] = row
	}

	if err := rows.Err(); err != nil {
		err = fmt.Errorf("Relations/Rows: %s", err)
		return nil, err
	}

	// Columns
	rows, err = db.QueryContext(ctx, QueryMarkerSQL+columnsSQL, ignoreRegexp)
	if err != nil {
		err = fmt.Errorf("Columns/Query: %s", err)
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var row state.PostgresColumn
		var exclusivelyLocked bool

		err = rows.Scan(&row.RelationOid, &row.Name, &row.DataType, &row.DefaultValue,
			&row.NotNull, &row.Position, &row.TypeOid, &exclusivelyLocked)
		if err != nil {
			err = fmt.Errorf("Columns/Scan: %s", err)
			return nil, err
		}

		relation, ok := relations[row.RelationOid]
		if !ok {
			continue
		}

		if exclusivelyLocked {
			relation.ExclusivelyLocked = true
		} else {
			relation.Columns = append(relation.Columns, row)
		}

		relations[row.RelationOid] = relation
	}

	if err := rows.Err(); err != nil {
		err = fmt.Errorf("Columns/Rows: %s", err)
		return nil, err
	}

	// Indices
	rows, err = db.QueryContext(ctx, QueryMarkerSQL+indicesSQL, ignoreRegexp)
	if err != nil {
		err = fmt.Errorf("Indices/Query: %s", err)
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var row state.PostgresIndex
		var columns string
		var options null.String
		var exclusivelyLocked bool

		err = rows.Scan(&row.RelationOid, &row.IndexOid, &columns, &row.Name, &row.IsPrimary,
			&row.IsUnique, &row.IsValid, &row.IndexDef, &row.ConstraintDef, &options, &row.IndexType,
			&exclusivelyLocked)
		if err != nil {
			err = fmt.Errorf("Indices/Scan: %s", err)
			return nil, err
		}

		for _, cstr := range strings.Split(columns, " ") {
			cint, _ := strconv.ParseInt(cstr, 10, 32)
			row.Columns = append(row.Columns, int32(cint))
		}

		row.Options = make(map[string]string)
		if options.Valid {
			for _, cstr := range strings.Split(strings.Trim(options.String, "{}"), ",") {
				parts := strings.Split(cstr, "=")
				row.Options[parts[0]] = parts[1]
			}
		}

		relation, ok := relations[row.RelationOid]
		if !ok {
			continue
		}

		if exclusivelyLocked {
			relation.ExclusivelyLocked = true
		} else {
			relation.Indices = append(relation.Indices, row)
		}

		relations[row.RelationOid] = relation
	}

	if err := rows.Err(); err != nil {
		err = fmt.Errorf("Indices/Rows: %s", err)
		return nil, err
	}

	// Constraints
	rows, err = db.QueryContext(ctx, QueryMarkerSQL+constraintsSQL, ignoreRegexp)
	if err != nil {
		err = fmt.Errorf("Constraints/Query: %s", err)
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var row state.PostgresConstraint
		var columns, foreignColumns null.String
		var foreignUpdateType, foreignDeleteType, foreignMatchType string
		var exclusivelyLocked bool

		err = rows.Scan(&row.RelationOid, &row.Name, &row.Type, &row.ConstraintDef,
			&columns, &row.ForeignOid, &foreignColumns, &foreignUpdateType,
			&foreignDeleteType, &foreignMatchType, &exclusivelyLocked)
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
				cint, _ := strconv.ParseInt(cstr, 10, 32)
				row.Columns = append(row.Columns, int32(cint))
			}
		}
		if foreignColumns.Valid {
			for _, cstr := range strings.Split(strings.Trim(foreignColumns.String, "{}"), ",") {
				cint, _ := strconv.ParseInt(cstr, 10, 32)
				row.ForeignColumns = append(row.ForeignColumns, int32(cint))
			}
		}

		relation, ok := relations[row.RelationOid]
		if !ok {
			continue
		}

		if exclusivelyLocked {
			relation.ExclusivelyLocked = true
		} else {
			relation.Constraints = append(relation.Constraints, row)
		}

		relations[row.RelationOid] = relation
	}

	if err := rows.Err(); err != nil {
		err = fmt.Errorf("Constraints/Rows: %s", err)
		return nil, err
	}

	// View definitions
	rows, err = db.QueryContext(ctx, QueryMarkerSQL+viewDefinitionSQL, ignoreRegexp)
	if err != nil {
		err = fmt.Errorf("Views/Prepare: %s", err)
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var relationOid state.Oid
		var viewDefinition string
		var exclusivelyLocked bool

		err := rows.Scan(&relationOid, &viewDefinition, &exclusivelyLocked)
		if err != nil {
			err = fmt.Errorf("Views/Scan: %s", err)
			return nil, err
		}

		relation, ok := relations[relationOid]
		if !ok {
			continue
		}

		if exclusivelyLocked {
			relation.ExclusivelyLocked = true
		} else {
			relation.ViewDefinition = viewDefinition
		}

		relations[relationOid] = relation
	}

	if err := rows.Err(); err != nil {
		err = fmt.Errorf("Views/Rows: %s", err)
		return nil, err
	}

	// Flip the oid-based map into an array
	v := make([]state.PostgresRelation, 0, len(relations))
	for _, value := range relations {
		v = append(v, value)
	}

	return v, nil
}

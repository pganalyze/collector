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

func unpackPostgresInt32Array(input null.String) (result []int32) {
	if !input.Valid {
		return
	}

	for _, cstr := range strings.Split(strings.Trim(input.String, "{}"), ",") {
		cint, _ := strconv.ParseInt(cstr, 10, 32)
		result = append(result, int32(cint))
	}

	return
}

func unpackPostgresOidArray(input null.String) (result []state.Oid) {
	if !input.Valid {
		return
	}

	for _, cstr := range strings.Split(strings.Trim(input.String, "{}"), ",") {
		cint, _ := strconv.ParseInt(cstr, 10, 32)
		result = append(result, state.Oid(cint))
	}

	return
}

func unpackPostgresStringArray(input null.String) (result []string) {
	if !input.Valid {
		return
	}

	result = strings.Split(strings.Trim(input.String, "{}"), ",")

	return
}

const resolveToastSQL string = `
SELECT n.nspname, c.relname
  FROM pg_catalog.pg_class c
	JOIN pg_catalog.pg_namespace n ON (n.oid = c.relnamespace)
 WHERE reltoastrelid = (SELECT c2.oid
	                        FROM pg_catalog.pg_class c2
                          JOIN pg_catalog.pg_namespace n2 ON (n2.oid = c2.relnamespace)
                         WHERE n2.nspname = 'pg_toast' AND c2.relname = $1)
`

func resolveToastTable(ctx context.Context, db *sql.DB, toastName string) (string, string, error) {
	var schemaName, relationName string
	err := db.QueryRowContext(ctx, QueryMarkerSQL+resolveToastSQL, toastName).Scan(&schemaName, &relationName)
	if err != nil {
		return "", "", err
	}
	return schemaName, relationName, nil
}

const settingValueSQL string = `
SELECT setting
	FROM pg_settings
 WHERE name = '%s'`

func GetPostgresSetting(ctx context.Context, db *sql.DB, settingName string) (string, error) {
	var value string

	err := db.QueryRow(QueryMarkerSQL + fmt.Sprintf(settingValueSQL, settingName)).Scan(&value)
	if err != nil {
		return "", fmt.Errorf("Could not read \"%s\" setting: %s", settingName, err)
	}

	return value, nil
}

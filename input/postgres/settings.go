package postgres

import (
	"database/sql"
	"fmt"

	"github.com/pganalyze/collector/state"
)

// Retrieve config settings, with at most one row per setting name.
//
// Some variants of Postgres, most notably Aurora, seem to sometimes output
// duplicate values in pg_settings (listing the default and the config file),
// confusing later logic. Thus we ensure that we get at most one row per name
// here, and sort by sourceline (putting defaults which don't have it set last).
const settingsSQL string = `
SELECT DISTINCT ON (name)
	   name,
	   CASE name
		 WHEN 'primary_conninfo' THEN regexp_replace(setting, '.', 'X', 'g')
		 ELSE setting
	   END AS current_value,
	   unit,
	   boot_val AS boot_value,
	   reset_val AS reset_value,
	   source,
	   sourcefile,
	   sourceline
  FROM pg_catalog.pg_settings
 ORDER BY name, CASE source WHEN 'default' THEN 1 ELSE 0 END`

func GetSettings(db *sql.DB) ([]state.PostgresSetting, error) {
	stmt, err := db.Prepare(QueryMarkerSQL + settingsSQL)
	if err != nil {
		err = fmt.Errorf("Settings/Prepare: %s", err)
		return nil, err
	}

	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		err = fmt.Errorf("Settings/Query: %s", err)
		return nil, err
	}

	defer rows.Close()

	var settings []state.PostgresSetting

	for rows.Next() {
		var row state.PostgresSetting

		err := rows.Scan(&row.Name, &row.CurrentValue, &row.Unit, &row.BootValue,
			&row.ResetValue, &row.Source, &row.SourceFile, &row.SourceLine)
		if err != nil {
			err = fmt.Errorf("Settings/Scan: %s", err)
			return nil, err
		}

		settings = append(settings, row)
	}

	if err = rows.Err(); err != nil {
		err = fmt.Errorf("Settings/Rows: %s", err)
		return nil, err
	}

	return settings, nil
}

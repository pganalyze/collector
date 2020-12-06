package postgres

import (
	"database/sql"
	"fmt"

	"github.com/pganalyze/collector/state"
)

const settingsSQL string = `
SELECT name,
			 setting AS current_value,
			 unit,
			 boot_val AS boot_value,
			 reset_val AS reset_value,
			 source,
			 sourcefile,
			 sourceline
	FROM pg_catalog.pg_settings`

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

	return settings, nil
}

package dbstats

import (
	"database/sql"

	null "gopkg.in/guregu/null.v2"
)

type Setting struct {
	Name         string      `json:"name"`
	CurrentValue null.String `json:"current_value"`
	Unit         null.String `json:"unit"`
	BootValue    null.String `json:"boot_value"`
	ResetValue   null.String `json:"reset_value"`
	Source       null.String `json:"source"`
	SourceFile   null.String `json:"sourcefile"`
	SourceLine   null.String `json:"sourceline"`
}

const settingsSQL string = `
SELECT name,
			 setting AS current_value,
			 unit,
			 boot_val AS boot_value,
			 reset_val AS reset_value,
			 source,
			 sourcefile,
			 sourceline
	FROM pg_settings`

func GetSettings(db *sql.DB, postgresVersionNum int) ([]Setting, error) {
	stmt, err := db.Prepare(QueryMarkerSQL + settingsSQL)
	if err != nil {
		return nil, err
	}

	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var settings []Setting

	for rows.Next() {
		var row Setting

		err := rows.Scan(&row.Name, &row.CurrentValue, &row.Unit, &row.BootValue,
			&row.ResetValue, &row.Source, &row.SourceFile, &row.SourceLine)
		if err != nil {
			return nil, err
		}

		settings = append(settings, row)
	}

	return settings, nil
}

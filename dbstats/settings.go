package dbstats

import (
	"database/sql"

	"github.com/pganalyze/collector/snapshot"
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
	FROM pg_settings`

func GetSettings(db *sql.DB, postgresVersion snapshot.PostgresVersion) ([]snapshot.Setting, error) {
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

	var settings []snapshot.Setting

	for rows.Next() {
		var row snapshot.Setting

		err := rows.Scan(&row.Name, &row.CurrentValue, &row.Unit, &row.BootValue,
			&row.ResetValue, &row.Source, &row.SourceFile, &row.SourceLine)
		if err != nil {
			return nil, err
		}

		settings = append(settings, row)
	}

	return settings, nil
}

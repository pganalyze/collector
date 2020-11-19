package steps

import (
	"errors"
	"fmt"
	"strings"

	survey "github.com/AlecAivazis/survey/v2"
	s "github.com/pganalyze/collector/setup/state"
)

var SelectDatabases = &s.Step{
	Description: "Select database(s) to monitor",
	Check: func(state *s.SetupState) (bool, error) {
		hasDb := state.CurrentSection.HasKey("db_name")
		if !hasDb {
			return false, nil
		}
		key, err := state.CurrentSection.GetKey("db_name")
		if err != nil {
			return false, err
		}
		dbs := key.Strings(",")
		if len(dbs) == 0 || dbs[0] == "" {
			return false, nil
		}
		db := dbs[0]
		// Now that we know the database, connect to the right one for setup:
		// this is important for extensions and helper functions. Note that we
		// need to do this in Check, rather than the Run, since a subsequent
		// execution, resuming an incomplete setup, will not run Run again
		state.QueryRunner.Database = db
		return true, nil
	},
	Run: func(state *s.SetupState) error {
		rows, err := state.QueryRunner.Query("SELECT datname FROM pg_database WHERE datallowconn AND NOT datistemplate")
		if err != nil {
			return err
		}
		var dbOpts []string
		for _, row := range rows {
			dbOpts = append(dbOpts, row.GetString(0))
		}

		var dbNames []string
		if state.Inputs.Scripted {
			if !state.Inputs.Settings.DBName.Valid {
				return errors.New("no db_name setting specified")
			}
			dbNameInputs := strings.Split(state.Inputs.Settings.DBName.String, ",")
			for i, dbNameInput := range dbNameInputs {
				trimmed := strings.TrimSpace(dbNameInput)
				if trimmed == "*" {
					dbNames = append(dbNames, trimmed)
				} else {
					for _, opt := range dbOpts {
						if trimmed == opt {
							dbNames = append(dbNames, trimmed)
							break
						}
					}
				}

				if len(dbNames) != i+1 {
					return fmt.Errorf("database %s not found", trimmed)
				}
			}
		} else {
			var primaryDb string
			err = survey.AskOne(&survey.Select{
				Message: "Choose a primary database to monitor (will be saved to collector config):",
				Options: dbOpts,
				Help:    "The collector will connect to this database for monitoring; others can be added next",
			}, &primaryDb)
			if err != nil {
				return err
			}

			dbNames = append(dbNames, primaryDb)
			if len(dbOpts) > 0 {
				var otherDbs []string
				for _, db := range dbOpts {
					if db == primaryDb {
						continue
					}
					otherDbs = append(otherDbs, db)
				}
				var othersOpt int
				err = survey.AskOne(&survey.Select{
					Message: "Monitor other databases? (will be saved to collector config):",
					Help:    "The 'all' option will also automatically monitor all future databases created on this server",
					Options: []string{"no other databases", "all other databases", "select databases..."},
				}, &othersOpt)
				if err != nil {
					return err
				}
				if othersOpt == 1 {
					dbNames = append(dbNames, "*")
				} else if othersOpt == 2 {
					var otherDbsSelected []string
					err = survey.AskOne(&survey.MultiSelect{
						Message: "Select other databases to monitor (will be saved to collector config):",
						Options: otherDbs,
					}, &otherDbsSelected)
					if err != nil {
						return err
					}
					dbNames = append(dbNames, otherDbsSelected...)
				}
			}
		}

		dbNamesStr := strings.Join(dbNames, ",")
		_, err = state.CurrentSection.NewKey("db_name", dbNamesStr)
		if err != nil {
			return err
		}

		return state.SaveConfig()
	},
}

package steps

import (
	"fmt"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/pganalyze/collector/setup/service"
	s "github.com/pganalyze/collector/setup/state"
	"github.com/pganalyze/collector/setup/util"
)

var ConfirmRestartPostgres = &s.Step{
	Description: "Confirm whether Postgres should be restarted to have pending configuration changes take effect",
	Check: func(state *s.SetupState) (bool, error) {
		row, err := state.QueryRunner.QueryRow("SELECT COUNT(*) FROM pg_settings WHERE pending_restart;")
		if err != nil {
			return false, err
		}
		return row.GetInt(0) == 0, nil
	},
	Run: func(state *s.SetupState) error {
		rows, err := state.QueryRunner.Query("SELECT name FROM pg_settings WHERE pending_restart")
		if err != nil {
			return err
		}
		var pendingSettings []string
		for _, row := range rows {
			pendingSettings = append(pendingSettings, row.GetString(0))
		}

		pendingList := util.JoinWithAnd(pendingSettings)
		var restartNow bool
		if state.Inputs.Scripted {
			if !state.Inputs.ConfirmPostgresRestart.Valid || !state.Inputs.ConfirmPostgresRestart.Bool {
				return fmt.Errorf("confirm_postgres_restart flag not set but Postgres restart required for settings %s", pendingList)
			}
			restartNow = state.Inputs.ConfirmPostgresRestart.Bool
		} else {
			err = survey.AskOne(&survey.Confirm{
				Message: fmt.Sprintf("WARNING: Postgres must be restarted for changes to %s to take effect; restart Postgres now?", pendingList),
				Default: false,
			}, &restartNow)
			if err != nil {
				return err
			}

			if !restartNow {
				return nil
			}

			err = survey.AskOne(&survey.Confirm{
				Message: "WARNING: Your database will be restarted. Are you sure?",
				Default: false,
			}, &restartNow)
			if err != nil {
				return err
			}
		}

		if !restartNow {
			return nil
		}

		return service.RestartPostgres(state)
	},
}

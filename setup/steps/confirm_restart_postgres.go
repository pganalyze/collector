package steps

import (
	"fmt"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/pganalyze/collector/setup/service"
	"github.com/pganalyze/collector/setup/state"
	"github.com/pganalyze/collector/setup/util"
)

var ConfirmRestartPostgres = &state.Step{
	ID:          "confirm_restart_postgres",
	Description: "Confirm whether Postgres should be restarted to have pending configuration changes take effect",
	Check: func(s *state.SetupState) (bool, error) {
		row, err := s.QueryRunner.QueryRow("SELECT COUNT(*) FROM pg_settings WHERE pending_restart;")
		if err != nil {
			return false, err
		}
		return row.GetInt(0) == 0, nil
	},
	Run: func(s *state.SetupState) error {
		rows, err := s.QueryRunner.Query("SELECT name FROM pg_settings WHERE pending_restart")
		if err != nil {
			return err
		}
		var pendingSettings []string
		for _, row := range rows {
			pendingSettings = append(pendingSettings, row.GetString(0))
		}

		pendingList := util.JoinWithAnd(pendingSettings)
		var restartNow bool
		if s.Inputs.Scripted {
			if !s.Inputs.ConfirmPostgresRestart.Valid || !s.Inputs.ConfirmPostgresRestart.Bool {
				return fmt.Errorf("confirm_postgres_restart flag not set but Postgres restart required for settings %s", pendingList)
			}
			restartNow = s.Inputs.ConfirmPostgresRestart.Bool
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

		return service.RestartPostgres(s)
	},
}

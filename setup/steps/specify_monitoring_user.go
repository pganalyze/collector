package steps

import (
	"errors"
	"fmt"

	survey "github.com/AlecAivazis/survey/v2"
	s "github.com/pganalyze/collector/setup/state"
)

var SpecifyMonitoringUser = &s.Step{
	Description: "Check config for monitoring user",
	Check: func(state *s.SetupState) (bool, error) {
		hasUser := state.CurrentSection.HasKey("db_username")
		return hasUser, nil
	},
	Run: func(state *s.SetupState) error {
		var pgaUser string

		if state.Inputs.Scripted {
			if !state.Inputs.Settings.DBUsername.Valid {
				return errors.New("no db_username setting specified")
			}
			pgaUser = state.Inputs.Settings.DBUsername.String
		} else {
			var monitoringUserIdx int
			err := survey.AskOne(&survey.Select{
				Message: "Select Postgres user for the collector to use (will be saved to collector config):",
				Help:    "If the user does not exist, it can be created in a later step",
				Options: []string{"pganalyze (recommended)", "a different user"},
			}, &monitoringUserIdx)
			if err != nil {
				return err
			}

			if monitoringUserIdx == 0 {
				pgaUser = "pganalyze"
			} else if monitoringUserIdx == 1 {
				err := survey.AskOne(&survey.Input{
					Message: "Enter Postgres user for the collector to use (will be saved to collector config):",
					Help:    "If the user does not exist, it can be created in a later step",
				}, &pgaUser, survey.WithValidator(survey.Required))
				if err != nil {
					return err
				}
			} else {
				panic(fmt.Sprintf("unexpected user selection: %d", monitoringUserIdx))
			}
		}

		_, err := state.CurrentSection.NewKey("db_username", pgaUser)
		if err != nil {
			return err
		}
		return state.SaveConfig()
	},
}

package steps

import (
	"errors"
	"fmt"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/pganalyze/collector/setup/state"
)

var SpecifyMonitoringUser = &state.Step{
	ID:          "specify_monitoring_user",
	Description: "Specify the monitoring user to connect as (db_username) in the collector config file",
	Check: func(s *state.SetupState) (bool, error) {
		hasUser := s.CurrentSection.HasKey("db_username")
		return hasUser, nil
	},
	Run: func(s *state.SetupState) error {
		var pgaUser string

		if s.Inputs.Scripted {
			if !s.Inputs.Settings.DBUsername.Valid {
				return errors.New("no db_username setting specified")
			}
			pgaUser = s.Inputs.Settings.DBUsername.String
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

		_, err := s.CurrentSection.NewKey("db_username", pgaUser)
		if err != nil {
			return err
		}
		return s.SaveConfig()
	},
}

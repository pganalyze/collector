package steps

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

	survey "github.com/AlecAivazis/survey/v2"
	s "github.com/pganalyze/collector/setup/state"
)

var ConfigureMonitoringUserPasswd = &s.Step{
	Description: "Configure monitoring user password",
	Check: func(state *s.SetupState) (bool, error) {
		hasPassword := state.CurrentSection.HasKey("db_password")
		return hasPassword, nil
	},
	Run: func(state *s.SetupState) error {
		var passwordStrategy int
		if state.Inputs.Scripted {
			if state.Inputs.GenerateMonitoringPassword.Valid && state.Inputs.GenerateMonitoringPassword.Bool {
				if state.Inputs.Settings.DBPassword.Valid && state.Inputs.Settings.DBPassword.String != "" {
					return errors.New("cannot specify both generate password and set explicit password")
				}
				passwordStrategy = 0
			} else if state.Inputs.Settings.DBPassword.Valid && state.Inputs.Settings.DBPassword.String != "" {
				passwordStrategy = 1
			} else {
				return errors.New("no db_password specified and generate_monitoring_password flag not set")
			}
		} else {
			err := survey.AskOne(&survey.Select{
				Message: "Select how to set up the collector user password (will be saved to collector config):",
				Options: []string{"generate random password (recommended)", "enter existing password"},
			}, &passwordStrategy)
			if err != nil {
				return err
			}
		}

		var pgaPasswd string
		if passwordStrategy == 0 {
			passwdBytes := make([]byte, 16)
			rand.Read(passwdBytes)
			pgaPasswd = hex.EncodeToString(passwdBytes)
		} else if passwordStrategy == 1 {
			if state.Inputs.Scripted {
				pgaPasswd = state.Inputs.Settings.DBPassword.String
			} else {
				err := survey.AskOne(&survey.Input{
					Message: "Enter password for the collector to use (will be saved to collector config):",
				}, &pgaPasswd, survey.WithValidator(survey.Required))
				if err != nil {
					return err
				}
			}
		} else {
			panic(fmt.Sprintf("unexpected password option selection: %d", passwordStrategy))
		}

		_, err := state.CurrentSection.NewKey("db_password", pgaPasswd)
		if err != nil {
			return err
		}

		return state.SaveConfig()
	},
}

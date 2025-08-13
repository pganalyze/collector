package steps

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/pganalyze/collector/setup/state"
)

var SpecifyMonitoringUserPasswd = &state.Step{
	ID:          "specify_monitoring_user_password",
	Description: "Specify monitoring user password (db_password) in the collector config file",
	Check: func(s *state.SetupState) (bool, error) {
		return s.CurrentSection.HasKey("db_password"), nil
	},
	Run: func(s *state.SetupState) error {
		var passwordStrategy int
		if s.Inputs.Scripted {
			if s.Inputs.GenerateMonitoringPassword.Valid && s.Inputs.GenerateMonitoringPassword.Bool {
				if s.Inputs.Settings.DBPassword.Valid && s.Inputs.Settings.DBPassword.String != "" {
					return errors.New("cannot specify both generate password and set explicit password")
				}
				passwordStrategy = 0
			} else if s.Inputs.Settings.DBPassword.Valid && s.Inputs.Settings.DBPassword.String != "" {
				passwordStrategy = 1
			} else {
				return errors.New("no db_password specified and generate_monitoring_password flag not set")
			}
		} else {
			err := survey.AskOne(&survey.Select{
				Message: "Select how to set up the collector user password (will be saved to collector config):",
				Options: []string{"generate random password (recommended)", "enter password"},
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
			if s.Inputs.Scripted {
				pgaPasswd = s.Inputs.Settings.DBPassword.String
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

		_, err := s.CurrentSection.NewKey("db_password", pgaPasswd)
		if err != nil {
			return err
		}

		return s.SaveConfig()
	},
}

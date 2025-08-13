package steps

import (
	"errors"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/pganalyze/collector/setup/state"
)

var SpecifyAPIKey = &state.Step{
	ID:          "specify_api_key",
	Description: "Specify the pganalyze API key (api_key) in the collector config file",
	Check: func(s *state.SetupState) (bool, error) {
		return s.PGAnalyzeSection.HasKey("api_key"), nil
	},
	Run: func(s *state.SetupState) error {
		var apiKey string
		var apiBaseURL string

		if s.Inputs.Settings.APIKey.Valid {
			apiKey = s.Inputs.Settings.APIKey.String
		}
		if s.Inputs.Settings.APIBaseURL.Valid {
			apiBaseURL = s.Inputs.Settings.APIBaseURL.String
		}

		var configWriteConfirmed bool

		if s.Inputs.Scripted {
			if apiKey != "" {
				configWriteConfirmed = true
			} else {
				return errors.New("no api_key setting specified")
			}
		} else if apiKey == "" {
			err := survey.AskOne(&survey.Input{
				Message: "Please enter API key (will be saved to collector config):",
				Help:    "The key can be found on the API keys page for your organization in the pganalyze app",
			}, &apiKey, survey.WithValidator(survey.Required))
			if err != nil {
				return err
			}
			configWriteConfirmed = true
		} else {
			err := survey.AskOne(&survey.Confirm{
				Message: "Save pganalyze API key to collector config?",
				Default: false,
			}, &configWriteConfirmed)
			if err != nil {
				return err
			}
		}
		if !configWriteConfirmed {
			return nil
		}
		_, err := s.PGAnalyzeSection.NewKey("api_key", apiKey)
		if err != nil {
			return err
		}
		if apiBaseURL != "" {
			_, err := s.PGAnalyzeSection.NewKey("api_base_url", apiBaseURL)
			if err != nil {
				return err
			}
		}
		return s.SaveConfig()
	},
}

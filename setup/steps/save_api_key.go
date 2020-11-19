package steps

import (
	"errors"
	"os"

	survey "github.com/AlecAivazis/survey/v2"
	s "github.com/pganalyze/collector/setup/state"
)

var SaveAPIKey = &s.Step{
	Description: "Add pganalyze API key to collector config",
	Check: func(state *s.SetupState) (bool, error) {
		return state.PGAnalyzeSection.HasKey("api_key"), nil
	},
	Run: func(state *s.SetupState) error {
		apiKey := os.Getenv("PGA_API_KEY")
		var configWriteConfirmed bool

		if state.Inputs.Scripted {
			if state.Inputs.Settings.APIKey.Valid {
				inputsAPIKey := state.Inputs.Settings.APIKey.String
				if apiKey != "" && inputsAPIKey != apiKey {
					state.Log("WARNING: overriding API key from env with API key from inputs file")
				}
				apiKey = inputsAPIKey
				configWriteConfirmed = true
			} else if apiKey != "" {
				configWriteConfirmed = true
			} else {
				return errors.New("no api_key setting specified and PGA_API_KEY not found in env")
			}
		} else if apiKey == "" {
			err := survey.AskOne(&survey.Input{
				Message: "PGA_API_KEY environment variable not found; please enter API key (will be saved to collector config):",
				Help:    "The key can be found on the API keys page for your organization in the pganalyze app",
			}, &apiKey, survey.WithValidator(survey.Required))
			if err != nil {
				return err
			}
			configWriteConfirmed = true
		} else {
			err := survey.AskOne(&survey.Confirm{
				Message: "PGA_API_KEY found in environment; save to config file?",
				Default: false,
			}, &configWriteConfirmed)
			if err != nil {
				return err
			}
		}
		if !configWriteConfirmed {
			return nil
		}
		_, err := state.PGAnalyzeSection.NewKey("api_key", apiKey)
		if err != nil {
			return err
		}
		return state.SaveConfig()
	},
}

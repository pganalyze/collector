package steps

import (
	"errors"
	"fmt"
	"strings"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/lib/pq"
	"github.com/pganalyze/collector/logs"
	s "github.com/pganalyze/collector/setup/state"
	"github.com/pganalyze/collector/setup/util"
)

var EnsureSupportedLogLinePrefix = &s.Step{
	ID:          "li_ensure_supported_log_line_prefix",
	Kind:        s.LogInsightsStep,
	Description: "Ensure the log_line_prefix setting in Postgres is supported by the collector",
	Check: func(state *s.SetupState) (bool, error) {
		row, err := state.QueryRunner.QueryRow(`SELECT setting FROM pg_settings WHERE name = 'log_line_prefix'`)
		if err != nil {
			return false, err
		}

		currValue := row.GetString(0)
		hasDb := strings.Contains(currValue, "%d")
		hasUser := strings.Contains(currValue, "%u")
		hasTs := strings.Contains(currValue, "%m") || strings.Contains(currValue, "%n") || strings.Contains(currValue, "%t")
		supported := hasDb && hasUser && hasTs

		needsUpdate := !supported ||
			(state.Inputs.Scripted &&
				state.Inputs.GUCS.LogLinePrefix.Valid &&
				currValue != state.Inputs.GUCS.LogLinePrefix.String)

		return !needsUpdate, nil
	},
	Run: func(state *s.SetupState) error {
		var selectedPrefix string
		if state.Inputs.Scripted {
			if !state.Inputs.GUCS.LogLinePrefix.Valid {
				return errors.New("log_line_prefix not provided and current setting is not supported")
			}
			selectedPrefix = state.Inputs.GUCS.LogLinePrefix.String
		} else {
			row, err := state.QueryRunner.QueryRow(`SELECT setting FROM pg_settings WHERE name = 'log_line_prefix'`)
			if err != nil {
				return err
			}
			oldVal := row.GetString(0)
			err = survey.AskOne(&survey.Input{
				Message: fmt.Sprintf("Setting 'log_line_prefix' (%s) is missing user (%%u), database (%%d), or timestamp (%%n, %%m, or %%t); set to (will be saved to Postgres):", oldVal),
				Suggest: func(toComplete string) []string {
					if toComplete == "" {
						return []string{logs.LogPrefixRecommended}
					}
					return []string{}
				},
				Help: "Check format specifier reference in Postgres documentation: https://www.postgresql.org/docs/current/runtime-config-logging.html#GUC-LOG-LINE-PREFIX",
			}, &selectedPrefix)
			if err != nil {
				return err
			}
		}
		return util.ApplyConfigSetting("log_line_prefix", pq.QuoteLiteral(selectedPrefix), state.QueryRunner)
	},
}

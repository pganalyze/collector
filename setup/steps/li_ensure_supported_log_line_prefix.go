package steps

import (
	"errors"
	"fmt"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/lib/pq"
	s "github.com/pganalyze/collector/setup/state"
	"github.com/pganalyze/collector/setup/util"
)

var EnsureSupportedLogLinePrefix = &s.Step{
	Kind:        s.LogInsightsStep,
	Description: "Ensure the log_line_prefix setting in Postgres is supported by the collector",
	Check: func(state *s.SetupState) (bool, error) {
		row, err := state.QueryRunner.QueryRow(`SELECT setting FROM pg_settings WHERE name = 'log_line_prefix'`)
		if err != nil {
			return false, err
		}

		currValue := row.GetString(0)
		needsUpdate := !util.Includes(s.SupportedLogLinePrefixes, currValue) ||
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
			if !util.Includes(s.SupportedLogLinePrefixes, selectedPrefix) {
				return fmt.Errorf("log_line_prefix provided as unsupported value '%s'", selectedPrefix)
			}
		} else {
			row, err := state.QueryRunner.QueryRow(`SELECT setting FROM pg_settings WHERE name = 'log_line_prefix'`)
			if err != nil {
				return err
			}
			oldVal := row.GetString(0)
			var opts []string
			for i, llp := range s.SupportedLogLinePrefixes {
				// N.B.: we quote the options because many prefixes end in whitespace; we need to make that clear
				var opt string
				if i == 0 {
					opt = fmt.Sprintf("'%s' (recommended)", llp)
				} else {
					opt = fmt.Sprintf("'%s'", llp)
				}
				opts = append(opts, opt)
			}
			var prefixIdx int
			err = survey.AskOne(&survey.Select{
				Message: fmt.Sprintf("Setting 'log_line_prefix' is set to unsupported value '%s'; set to (will be saved to Postgres):", oldVal),
				Help:    "Check format specifier reference in Postgres documentation: https://www.postgresql.org/docs/current/runtime-config-logging.html#GUC-LOG-LINE-PREFIX",
				Options: opts,
			}, &prefixIdx)
			if err != nil {
				return err
			}
			selectedPrefix = s.SupportedLogLinePrefixes[prefixIdx]
		}
		return util.ApplyConfigSetting("log_line_prefix", pq.QuoteLiteral(selectedPrefix), state.QueryRunner)
	},
}

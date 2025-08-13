package steps

import (
	"fmt"
	"strconv"
	"strings"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/guregu/null"
	"github.com/lib/pq"

	"github.com/pganalyze/collector/setup/state"
	"github.com/pganalyze/collector/setup/util"
)

// N.B.: this needs to happen *after* the Postgres restart so that ALTER SYSTEM
// recognizes these as valid configuration settings
var EnsureRecommendedAutoExplainSettings = &state.Step{
	Kind:        state.AutomatedExplainStep,
	ID:          "aemod_ensure_recommended_settings",
	Description: "Ensure auto_explain settings in Postgres are configured as recommended, if desired",
	Check: func(s *state.SetupState) (bool, error) {
		if s.DidAutoExplainRecommendedSettings ||
			(s.Inputs.EnsureAutoExplainRecommendedSettings.Valid && !s.Inputs.EnsureAutoExplainRecommendedSettings.Bool) {
			return true, nil
		}
		logExplain, err := util.UsingLogExplain(s.CurrentSection)
		if err != nil || logExplain {
			return logExplain, err
		}

		autoExplainGucsQuery := getAutoExplainGUCSQuery(s)
		rows, err := s.QueryRunner.Query(
			autoExplainGucsQuery,
		)
		if err != nil {
			return false, fmt.Errorf("error checking existing settings: %s", err)
		}

		return len(rows) == 0, nil
	},
	Run: func(s *state.SetupState) error {
		var doReview bool
		if s.Inputs.Scripted {
			if s.Inputs.EnsureAutoExplainRecommendedSettings.Valid {
				doReview = s.Inputs.EnsureAutoExplainRecommendedSettings.Bool
			}
		} else {
			err := survey.AskOne(&survey.Confirm{
				Message: "Review auto_explain configuration settings?",
				Default: false,
				Help:    "Optional, but will ensure best balance of monitoring visibility and performance; review these settings at https://pganalyze.com/docs/explain/setup/auto_explain",
			}, &doReview)
			if err != nil {
				return err
			}
			s.Inputs.EnsureAutoExplainRecommendedSettings = null.BoolFrom(doReview)
		}

		if !doReview {
			return nil
		}

		autoExplainGucsQuery := getAutoExplainGUCSQuery(s)
		rows, err := s.QueryRunner.Query(
			autoExplainGucsQuery,
		)
		if err != nil {
			return fmt.Errorf("error checking existing settings: %s", err)
		}
		if len(rows) == 0 {
			s.Log("all auto_explain configuration settings using recommended values")
			s.DidAutoExplainRecommendedSettings = true
			return nil
		}
		settingsToReview := make(map[string]string)
		for _, row := range rows {
			settingsToReview[row.GetString(0)] = row.GetString(1)
		}

		// N.B.: we ask about log_timing first since this is typically most impactful
		if currValue, ok := settingsToReview["auto_explain.log_timing"]; ok {
			logTiming, err := getLogTimingValue(s, currValue)
			if err != nil {
				return err
			}
			if logTiming != currValue {
				err = util.ApplyConfigSetting("auto_explain.log_timing", logTiming, s.QueryRunner)
				if err != nil {
					return err
				}
			}
		}

		if currValue, ok := settingsToReview["auto_explain.log_analyze"]; ok {
			logAnalyze, err := getLogAnalyzeValue(s, currValue)
			if err != nil {
				return err
			}

			if logAnalyze != currValue {
				err = util.ApplyConfigSetting("auto_explain.log_analyze", logAnalyze, s.QueryRunner)
				if err != nil {
					return err
				}
			}
		}

		// we could reason based on the above, but it's safe to just re-query
		row, err := s.QueryRunner.QueryRow("SHOW auto_explain.log_analyze")
		if err != nil {
			return err
		}
		isLogAnalyzeOn := row.GetString(0) == "on"

		if isLogAnalyzeOn {
			if currValue, ok := settingsToReview["auto_explain.log_buffers"]; ok {
				logBuffers, err := getLogBuffersValue(s, currValue)
				if err != nil {
					return err
				}

				if logBuffers != currValue {
					err = util.ApplyConfigSetting("auto_explain.log_buffers", logBuffers, s.QueryRunner)
					if err != nil {
						return err
					}
				}
			}

			if currValue, ok := settingsToReview["auto_explain.log_triggers"]; ok {
				logTriggers, err := getLogTriggersValue(s, currValue)
				if err != nil {
					return err
				}

				if logTriggers != currValue {
					err = util.ApplyConfigSetting("auto_explain.log_triggers", logTriggers, s.QueryRunner)
					if err != nil {
						return err
					}
				}
			}

			if currValue, ok := settingsToReview["auto_explain.log_verbose"]; ok {
				logVerbose, err := getLogVerboseValue(s, currValue)
				if err != nil {
					return err
				}
				if logVerbose != currValue {
					err = util.ApplyConfigSetting("auto_explain.log_verbose", logVerbose, s.QueryRunner)
					if err != nil {
						return err
					}
				}
			}
		}

		if currValue, ok := settingsToReview["auto_explain.log_format"]; ok {
			logFormat, err := getLogFormatValue(s, currValue)
			if err != nil {
				return err
			}

			if logFormat != currValue {
				err = util.ApplyConfigSetting("auto_explain.log_format", logFormat, s.QueryRunner)
				if err != nil {
					return err
				}
			}
		}

		if currValue, ok := settingsToReview["auto_explain.log_min_duration"]; ok {
			logMinDuration, err := getLogMinDurationValue(s, currValue)
			if err != nil {
				return err
			}

			if logMinDuration != currValue {
				err = util.ApplyConfigSetting("auto_explain.log_min_duration", logMinDuration, s.QueryRunner)
				if err != nil {
					return err
				}
			}
		}

		if currValue, ok := settingsToReview["auto_explain.log_nested_statements"]; ok {
			logNested, err := getLogNestedStatements(s, currValue)
			if err != nil {
				return err
			}

			if logNested != currValue {
				err = util.ApplyConfigSetting("auto_explain.log_nested_statements", logNested, s.QueryRunner)
				if err != nil {
					return err
				}
			}
		}
		s.DidAutoExplainRecommendedSettings = true
		return nil
	},
}

func getAutoExplainGUCSQuery(s *state.SetupState) string {
	query := `SELECT name, setting
FROM pg_settings
WHERE `
	var predicate string

	if s.Inputs.Scripted {
		var predParts []string
		appendPredicate := func(name, value string) []string {
			return append(predParts,
				fmt.Sprintf(
					"(name = %s AND setting <> %s)",
					pq.QuoteLiteral(name),
					pq.QuoteLiteral(value),
				),
			)
		}
		if s.Inputs.GUCS.AutoExplainLogAnalyze.Valid {
			predParts = appendPredicate(
				"auto_explain.log_analyze",
				s.Inputs.GUCS.AutoExplainLogAnalyze.String,
			)
		}
		if s.Inputs.GUCS.AutoExplainLogBuffers.Valid {
			predParts = appendPredicate(
				"auto_explain.log_buffers",
				s.Inputs.GUCS.AutoExplainLogBuffers.String,
			)
		}
		if s.Inputs.GUCS.AutoExplainLogTiming.Valid {
			predParts = appendPredicate(
				"auto_explain.log_timing",
				s.Inputs.GUCS.AutoExplainLogTiming.String,
			)
		}
		if s.Inputs.GUCS.AutoExplainLogTriggers.Valid {
			predParts = appendPredicate(
				"auto_explain.log_triggers",
				s.Inputs.GUCS.AutoExplainLogTriggers.String,
			)
		}
		if s.Inputs.GUCS.AutoExplainLogVerbose.Valid {
			predParts = appendPredicate(
				"auto_explain.log_verbose",
				s.Inputs.GUCS.AutoExplainLogVerbose.String,
			)
		}
		if s.Inputs.GUCS.AutoExplainLogFormat.Valid {
			predParts = appendPredicate(
				"auto_explain.log_format",
				s.Inputs.GUCS.AutoExplainLogFormat.String,
			)
		}
		if s.Inputs.GUCS.AutoExplainLogMinDuration.Valid {
			// N.B.: here we check for exact equality with the setting, rather than
			// under the threshold, since the semantics of that behavior are more
			// straightforward when providing a setting from an inputs file
			predParts = append(
				predParts,
				fmt.Sprintf(
					"(name = 'auto_explain.log_min_duration' AND setting::integer <> %d)",
					s.Inputs.GUCS.AutoExplainLogMinDuration.Int64,
				),
			)
		}
		if s.Inputs.GUCS.AutoExplainLogNestedStatements.Valid {
			predParts = appendPredicate(
				"auto_explain.log_nested_statements",
				s.Inputs.GUCS.AutoExplainLogNestedStatements.String,
			)
		}

		predicate = strings.Join(predParts, " OR ")

	} else {
		predicate = fmt.Sprintf(
			`(name = 'auto_explain.log_analyze' AND setting <> %s) OR
(name = 'auto_explain.log_buffers' AND setting <> %s) OR
(name = 'auto_explain.log_timing' AND setting <> %s) OR
(name = 'auto_explain.log_triggers' AND setting <> %s) OR
(name = 'auto_explain.log_verbose' AND setting <> %s) OR
(name = 'auto_explain.log_format' AND setting <> %s) OR
(name = 'auto_explain.log_min_duration' AND setting::integer < %d) OR
(name = 'auto_explain.log_nested_statements' AND setting <> %s)`,
			pq.QuoteLiteral(state.RecommendedGUCS.AutoExplainLogAnalyze.String),
			pq.QuoteLiteral(state.RecommendedGUCS.AutoExplainLogBuffers.String),
			pq.QuoteLiteral(state.RecommendedGUCS.AutoExplainLogTiming.String),
			pq.QuoteLiteral(state.RecommendedGUCS.AutoExplainLogTriggers.String),
			pq.QuoteLiteral(state.RecommendedGUCS.AutoExplainLogVerbose.String),
			pq.QuoteLiteral(state.RecommendedGUCS.AutoExplainLogFormat.String),
			state.RecommendedGUCS.AutoExplainLogMinDuration.Int64,
			pq.QuoteLiteral(state.RecommendedGUCS.AutoExplainLogNestedStatements.String),
		)
	}

	return query + predicate
}

func getLogAnalyzeValue(s *state.SetupState, currValue string) (string, error) {
	if s.Inputs.Scripted {
		if !s.Inputs.GUCS.AutoExplainLogAnalyze.Valid {
			panic("auto_explain.log_analyze setting needs review but was not provided")
		}
		return s.Inputs.GUCS.AutoExplainLogAnalyze.String, nil
	}
	var logAnalyzeIdx int
	opts, optLabels := getBooleanOpts(currValue, state.RecommendedGUCS.AutoExplainLogAnalyze.String)
	err := survey.AskOne(&survey.Select{
		Message: fmt.Sprintf("Setting auto_explain.log_analyze is currently set to '%s':", currValue),
		Help:    "Include EXPLAIN ANALYZE output rather than just EXPLAIN output when a plan is logged; required for several other settings",
		Options: optLabels,
	}, &logAnalyzeIdx)
	if err != nil {
		return "", err
	}
	return opts[logAnalyzeIdx], nil
}

func getLogBuffersValue(s *state.SetupState, currValue string) (string, error) {
	if s.Inputs.Scripted {
		if !s.Inputs.GUCS.AutoExplainLogBuffers.Valid {
			panic("auto_explain.log_buffers setting needs review but was not provided")
		}
		return s.Inputs.GUCS.AutoExplainLogBuffers.String, nil
	}

	var logBuffersIdx int
	var opts, optLabels = getBooleanOpts(currValue, state.RecommendedGUCS.AutoExplainLogBuffers.String)
	err := survey.AskOne(&survey.Select{
		Message: fmt.Sprintf("Setting auto_explain.log_buffers is currently set to '%s'", currValue),
		Help:    "Include BUFFERS usage information when a plan is logged",
		Options: optLabels,
	}, &logBuffersIdx)
	if err != nil {
		return "", err
	}
	return opts[logBuffersIdx], nil
}

func getLogTimingValue(s *state.SetupState, currValue string) (string, error) {
	if s.Inputs.Scripted {
		if !s.Inputs.GUCS.AutoExplainLogTiming.Valid {
			panic("auto_explain.log_timing setting needs review but was not provided")
		}
		return s.Inputs.GUCS.AutoExplainLogTiming.String, nil
	}
	var logTimingIdx int
	opts, optLabels := getBooleanOpts(currValue, state.RecommendedGUCS.AutoExplainLogTiming.String)
	err := survey.AskOne(&survey.Select{
		Message: fmt.Sprintf("Setting auto_explain.log_timing is currently set to '%s'", currValue),
		Help:    "Include timing information for each plan node when a plan is logged; can have high performance impact",
		Options: optLabels,
	}, &logTimingIdx)
	if err != nil {
		return "", err
	}
	return opts[logTimingIdx], nil
}

func getLogTriggersValue(s *state.SetupState, currValue string) (string, error) {
	if s.Inputs.Scripted {
		if !s.Inputs.GUCS.AutoExplainLogTriggers.Valid {
			panic("auto_explain.log_triggers setting needs review but was not provided")
		}
		return s.Inputs.GUCS.AutoExplainLogTriggers.String, nil
	}

	var logTriggersIdx int
	opts, optLabels := getBooleanOpts(currValue, state.RecommendedGUCS.AutoExplainLogTriggers.String)
	err := survey.AskOne(&survey.Select{
		Message: fmt.Sprintf("Setting auto_explain.log_triggers is currently set to '%s'", currValue),
		Help:    "Include trigger execution statistics when a plan is logged",
		Options: optLabels,
	}, &logTriggersIdx)
	if err != nil {
		return "", err
	}
	return opts[logTriggersIdx], nil
}

func getLogVerboseValue(s *state.SetupState, currValue string) (string, error) {
	if s.Inputs.Scripted {
		if !s.Inputs.GUCS.AutoExplainLogVerbose.Valid {
			panic("auto_explain.log_verbose setting needs review but was not provided")
		}
		return s.Inputs.GUCS.AutoExplainLogVerbose.String, nil
	}

	var logVerboseIdx int
	opts, optLabels := getBooleanOpts(currValue, state.RecommendedGUCS.AutoExplainLogVerbose.String)
	err := survey.AskOne(&survey.Select{
		Message: fmt.Sprintf("Setting auto_explain.log_verbose is currently set to '%s'", currValue),
		Help:    "Include VERBOSE EXPLAIN details when a plan is logged",
		Options: optLabels,
	}, &logVerboseIdx)
	if err != nil {
		return "", err
	}
	return opts[logVerboseIdx], nil
}

func getLogFormatValue(s *state.SetupState, currValue string) (string, error) {
	if s.Inputs.Scripted {
		if !s.Inputs.GUCS.AutoExplainLogFormat.Valid {
			panic("auto_explain.log_format setting needs review but was not provided")
		}
		logFormat := s.Inputs.GUCS.AutoExplainLogFormat.String
		if logFormat != "text" && logFormat != "json" {
			return "", fmt.Errorf("unsupported auto_explain.log_format: %s", logFormat)
		}
		return logFormat, nil
	}

	var logFormatIdx int
	var logFormatOpts = []string{"json", "text"}
	var optLabels = []string{"set to 'json' (recommended; will be saved to Postgres)"}
	if currValue == "text" {
		optLabels = append(optLabels, "leave as 'text' (text format support is experimental)")
	} else {
		optLabels = append(
			optLabels,
			"set to 'text' (text format support is experimental; will be saved to Postgres)",
			fmt.Sprintf("leave as '%s' (unsupported)", currValue),
		)
	}

	err := survey.AskOne(&survey.Select{
		Message: fmt.Sprintf("Setting auto_explain.log_format is currently set to '%s'", currValue),
		Help:    "Select EXPLAIN output format to be used (only 'text' and 'json' are supported)",
		Options: optLabels,
	}, &logFormatIdx)
	if err != nil {
		return "", err
	}

	if logFormatIdx == 0 || logFormatIdx == 1 {
		return logFormatOpts[logFormatIdx], nil
	} else if logFormatIdx == 2 {
		return currValue, nil
	} else {
		panic(fmt.Sprintf("unexpected auto_explain.log_format selection: %d", logFormatIdx))
	}
}

func getLogMinDurationValue(s *state.SetupState, currValue string) (string, error) {
	if s.Inputs.Scripted {
		if !s.Inputs.GUCS.AutoExplainLogMinDuration.Valid {
			panic("auto_explain.log_min_duration setting needs review but was not provided")
		}
		return strconv.Itoa(int(s.Inputs.GUCS.AutoExplainLogMinDuration.Int64)), nil
	}

	var durationOpts = []string{
		fmt.Sprintf("set to %dms (recommended inital value; will be saved to Postgres)", state.RecommendedGUCS.AutoExplainLogMinDuration.Int64),
		"set to other value...",
		fmt.Sprintf("leave at %sms", currValue),
	}
	var durationOptIdx int
	err := survey.AskOne(&survey.Select{
		Message: fmt.Sprintf("Setting auto_explain.log_min_duration is currently set to '%s ms'", currValue),
		Help:    fmt.Sprintf("Threshold to log EXPLAIN plans, in ms; recommend %d, must be at least 10", state.RecommendedGUCS.AutoExplainLogMinDuration.Int64),
		Options: durationOpts,
	}, &durationOptIdx)
	if err != nil {
		return "", err
	}

	if durationOptIdx == 0 {
		return strconv.Itoa(int(state.RecommendedGUCS.AutoExplainLogMinDuration.Int64)), nil
	} else if durationOptIdx == 1 {
		var logMinDuration string
		err = survey.AskOne(&survey.Input{
			Message: "Set auto_explain.log_min_duration, in milliseconds, to (will be saved to Postgres):",
			Help:    "Threshold to log EXPLAIN plans, in ms; recommend 1000, must be at least 10",
		}, &logMinDuration, survey.WithValidator(util.ValidateLogMinDurationStatement))
		if err != nil {
			return "", err
		}
		return logMinDuration, nil
	} else if durationOptIdx == 2 {
		return currValue, nil
	} else {
		panic(fmt.Sprintf("unexpected auto_explain.log_min_duration selection: %d", durationOptIdx))
	}
}

func getLogNestedStatements(s *state.SetupState, currValue string) (string, error) {
	if s.Inputs.Scripted {
		if !s.Inputs.GUCS.AutoExplainLogNestedStatements.Valid {
			panic("auto_explain.log_nested_statements setting needs review but was not provided")
		}
		return s.Inputs.GUCS.AutoExplainLogNestedStatements.String, nil
	}

	var logNestedIdx int
	opts, optLabels := getBooleanOpts(currValue, state.RecommendedGUCS.AutoExplainLogNestedStatements.String)
	err := survey.AskOne(&survey.Select{
		Message: fmt.Sprintf("Setting auto_explain.log_nested_statements is currently set to '%s'", currValue),
		Help:    "Causes nested statements (statements executed inside a function) to be considered for logging",
		Options: optLabels,
	}, &logNestedIdx)
	if err != nil {
		return "", err
	}
	return opts[logNestedIdx], nil
}

func getBooleanOpts(currValue, recommendedValue string) (opts []string, optLabels []string) {
	if recommendedValue == "on" {
		opts = []string{"on", "off"}
		if currValue == "on" {
			return opts, []string{"leave as 'on' (recommended)", "set to 'off' (will be saved to Postgres)"}
		} else {
			return opts, []string{"set to 'on' (recommended; will be saved to Postgres)", "leave as 'off'"}
		}
	} else {
		opts = []string{"off", "on"}
		if currValue == "on" {
			return opts, []string{"set to 'off' (recommended; will be saved to Postgres)", "leave as 'on'"}
		} else {
			return opts, []string{"leave as 'off' (recommended)", "set to 'on' (will be saved to Postgres)"}
		}
	}
}

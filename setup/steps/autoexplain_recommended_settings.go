package steps

import (
	"fmt"
	"strconv"
	"strings"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/guregu/null"
	"github.com/lib/pq"

	s "github.com/pganalyze/collector/setup/state"
	"github.com/pganalyze/collector/setup/util"
)

// N.B.: this needs to happen *after* the Postgres restart so that ALTER SYSTEM
// recognizes these as valid configuration settings
var ConfigureAutoExplain = &s.Step{
	Kind:        s.AutomatedExplainStep,
	Description: "Review auto_explain settings",
	Check: func(state *s.SetupState) (bool, error) {
		if state.DidAutoExplainRecommendedSettings ||
			(state.Inputs.SkipAutoExplainRecommended.Valid && state.Inputs.SkipAutoExplainRecommended.Bool) {
			return true, nil
		}
		logExplain, err := util.UsingLogExplain(state.CurrentSection)
		if err != nil || logExplain {
			return logExplain, err
		}

		return false, nil
	},
	Run: func(state *s.SetupState) error {
		var doReview bool
		if state.Inputs.Scripted {
			if state.Inputs.SkipAutoExplainRecommended.Valid {
				doReview = !state.Inputs.SkipAutoExplainRecommended.Bool
			}
		} else {
			err := survey.AskOne(&survey.Confirm{
				Message: "Review auto_explain configuration settings?",
				Default: false,
				Help:    "Optional, but will ensure best balance of monitoring visibility and performance; review these settings at https://www.postgresql.org/docs/current/auto-explain.html#id-1.11.7.13.5",
			}, &doReview)
			if err != nil {
				return err
			}
			state.Inputs.SkipAutoExplainRecommended = null.BoolFrom(!doReview)
		}

		if !doReview {
			return nil
		}

		settingsToReview := make(map[string]string)
		autoExplainGucsQuery := getAutoExplainGUCSQuery(state)
		rows, err := state.QueryRunner.Query(
			autoExplainGucsQuery,
		)
		if err != nil {
			return fmt.Errorf("error checking existing settings: %s", err)
		}
		if len(rows) == 0 {
			state.Log("all auto_explain configuration settings using recommended values")
			state.DidAutoExplainRecommendedSettings = true
			return nil
		}
		for _, row := range rows {
			settingsToReview[row.GetString(0)] = row.GetString(1)
		}

		if currValue, ok := settingsToReview["auto_explain.log_analyze"]; ok {
			logAnalyze, err := getLogAnalyzeValue(state, currValue)
			if err != nil {
				return err
			}

			if logAnalyze != currValue {
				err = util.ApplyConfigSetting("auto_explain.log_analyze", logAnalyze, state.QueryRunner)
				if err != nil {
					return err
				}
			}
		}

		// we could reason based on the above, but it's safe to just re-query
		row, err := state.QueryRunner.QueryRow("SHOW auto_explain.log_analyze")
		if err != nil {
			return err
		}
		isLogAnalyzeOn := row.GetString(0) == "on"

		if isLogAnalyzeOn {
			if currValue, ok := settingsToReview["auto_explain.log_buffers"]; ok {
				logBuffers, err := getLogBuffersValue(state, currValue)
				if err != nil {
					return err
				}

				if logBuffers != currValue {
					err = util.ApplyConfigSetting("auto_explain.log_buffers", logBuffers, state.QueryRunner)
					if err != nil {
						return err
					}
				}
			}

			if currValue, ok := settingsToReview["auto_explain.log_timing"]; ok {
				logTiming, err := getLogTimingValue(state, currValue)
				if err != nil {
					return err
				}
				if logTiming != currValue {
					err = util.ApplyConfigSetting("auto_explain.log_timing", logTiming, state.QueryRunner)
					if err != nil {
						return err
					}
				}
			}

			if currValue, ok := settingsToReview["auto_explain.log_triggers"]; ok {
				logTriggers, err := getLogTriggersValue(state, currValue)
				if err != nil {
					return err
				}

				if logTriggers != currValue {
					err = util.ApplyConfigSetting("auto_explain.log_triggers", logTriggers, state.QueryRunner)
					if err != nil {
						return err
					}
				}
			}

			if currValue, ok := settingsToReview["auto_explain.log_verbose"]; ok {
				logVerbose, err := getLogVerboseValue(state, currValue)
				if err != nil {
					return err
				}
				if logVerbose != currValue {
					err = util.ApplyConfigSetting("auto_explain.log_verbose", logVerbose, state.QueryRunner)
					if err != nil {
						return err
					}
				}
			}
		}

		if currValue, ok := settingsToReview["auto_explain.log_format"]; ok {
			logFormat, err := getLogFormatValue(state, currValue)
			if err != nil {
				return err
			}

			if logFormat != currValue {
				err = util.ApplyConfigSetting("auto_explain.log_format", logFormat, state.QueryRunner)
				if err != nil {
					return err
				}
			}
		}

		if currValue, ok := settingsToReview["auto_explain.log_min_duration"]; ok {
			logMinDuration, err := getLogMinDurationValue(state, currValue)
			if err != nil {
				return err
			}

			if logMinDuration != currValue {
				err = util.ApplyConfigSetting("auto_explain.log_min_duration", logMinDuration, state.QueryRunner)
				if err != nil {
					return err
				}
			}
		}

		if currValue, ok := settingsToReview["auto_explain.log_nested_statements"]; ok {
			logNested, err := getLogNestedStatements(state, currValue)
			if err != nil {
				return err
			}

			if logNested != currValue {
				err = util.ApplyConfigSetting("auto_explain.log_nested_statements", logNested, state.QueryRunner)
				if err != nil {
					return err
				}
			}
		}
		state.DidAutoExplainRecommendedSettings = true
		return nil
	},
}

func getAutoExplainGUCSQuery(state *s.SetupState) string {
	query := `SELECT name, setting
FROM pg_settings
WHERE `
	var predicate string

	if state.Inputs.Scripted {
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
		if state.Inputs.GUCS.AutoExplainLogAnalyze.Valid {
			predParts = appendPredicate(
				"auto_explain.log_analyze",
				state.Inputs.GUCS.AutoExplainLogAnalyze.String,
			)
		}
		if state.Inputs.GUCS.AutoExplainLogBuffers.Valid {
			predParts = appendPredicate(
				"auto_explain.log_buffers",
				state.Inputs.GUCS.AutoExplainLogBuffers.String,
			)
		}
		if state.Inputs.GUCS.AutoExplainLogTiming.Valid {
			predParts = appendPredicate(
				"auto_explain.log_timing",
				state.Inputs.GUCS.AutoExplainLogTiming.String,
			)
		}
		if state.Inputs.GUCS.AutoExplainLogTriggers.Valid {
			predParts = appendPredicate(
				"auto_explain.log_triggers",
				state.Inputs.GUCS.AutoExplainLogTriggers.String,
			)
		}
		if state.Inputs.GUCS.AutoExplainLogVerbose.Valid {
			predParts = appendPredicate(
				"auto_explain.log_verbose",
				state.Inputs.GUCS.AutoExplainLogVerbose.String,
			)
		}
		if state.Inputs.GUCS.AutoExplainLogFormat.Valid {
			predParts = appendPredicate(
				"auto_explain.log_format",
				state.Inputs.GUCS.AutoExplainLogFormat.String,
			)
		}
		if state.Inputs.GUCS.AutoExplainLogMinDuration.Valid {
			// N.B.: here we check for exact equality with the setting, rather than
			// under the threshold, since the semantics of that behavior are more
			// straightforward when providing a setting from an inputs file
			predParts = append(
				predParts,
				fmt.Sprintf(
					"(name = 'auto_explain.log_min_duration' AND setting::float <> %d)",
					state.Inputs.GUCS.AutoExplainLogMinDuration.Int64,
				),
			)
		}
		if state.Inputs.GUCS.AutoExplainLogNestedStatements.Valid {
			predParts = appendPredicate(
				"auto_explain.log_nested_statements",
				state.Inputs.GUCS.AutoExplainLogNestedStatements.String,
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
(name = 'auto_explain.log_min_duration' AND setting::float < %d) OR
(name = 'auto_explain.log_nested_statements' AND setting <> %s)`,
			pq.QuoteLiteral(s.RecommendedGUCS.AutoExplainLogAnalyze.String),
			pq.QuoteLiteral(s.RecommendedGUCS.AutoExplainLogBuffers.String),
			pq.QuoteLiteral(s.RecommendedGUCS.AutoExplainLogTiming.String),
			pq.QuoteLiteral(s.RecommendedGUCS.AutoExplainLogTriggers.String),
			pq.QuoteLiteral(s.RecommendedGUCS.AutoExplainLogVerbose.String),
			pq.QuoteLiteral(s.RecommendedGUCS.AutoExplainLogFormat.String),
			s.RecommendedGUCS.AutoExplainLogMinDuration.Int64,
			pq.QuoteLiteral(s.RecommendedGUCS.AutoExplainLogNestedStatements.String),
		)
	}

	return query + predicate
}

func getLogAnalyzeValue(state *s.SetupState, currValue string) (string, error) {
	if state.Inputs.Scripted {
		if !state.Inputs.GUCS.AutoExplainLogAnalyze.Valid {
			panic("auto_explain.log_analyze setting needs review but was not provided")
		}
		return state.Inputs.GUCS.AutoExplainLogAnalyze.String, nil
	}
	var logAnalyzeIdx int
	opts, optLabels := getBooleanOpts(currValue, s.RecommendedGUCS.AutoExplainLogAnalyze.String)
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

func getLogBuffersValue(state *s.SetupState, currValue string) (string, error) {
	if state.Inputs.Scripted {
		if !state.Inputs.GUCS.AutoExplainLogBuffers.Valid {
			panic("auto_explain.log_buffers setting needs review but was not provided")
		}
		return state.Inputs.GUCS.AutoExplainLogBuffers.String, nil
	}

	var logBuffersIdx int
	var opts, optLabels = getBooleanOpts(currValue, s.RecommendedGUCS.AutoExplainLogBuffers.String)
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

func getLogTimingValue(state *s.SetupState, currValue string) (string, error) {
	if state.Inputs.Scripted {
		if !state.Inputs.GUCS.AutoExplainLogTiming.Valid {
			panic("auto_explain.log_timing setting needs review but was not provided")
		}
		return state.Inputs.GUCS.AutoExplainLogTiming.String, nil
	}
	var logTimingIdx int
	opts, optLabels := getBooleanOpts(currValue, s.RecommendedGUCS.AutoExplainLogTiming.String)
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

func getLogTriggersValue(state *s.SetupState, currValue string) (string, error) {
	if state.Inputs.Scripted {
		if !state.Inputs.GUCS.AutoExplainLogTriggers.Valid {
			panic("auto_explain.log_triggers setting needs review but was not provided")
		}
		return state.Inputs.GUCS.AutoExplainLogTriggers.String, nil
	}

	var logTriggersIdx int
	opts, optLabels := getBooleanOpts(currValue, s.RecommendedGUCS.AutoExplainLogTriggers.String)
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

func getLogVerboseValue(state *s.SetupState, currValue string) (string, error) {
	if state.Inputs.Scripted {
		if !state.Inputs.GUCS.AutoExplainLogVerbose.Valid {
			panic("auto_explain.log_verbose setting needs review but was not provided")
		}
		return state.Inputs.GUCS.AutoExplainLogVerbose.String, nil
	}

	var logVerboseIdx int
	opts, optLabels := getBooleanOpts(currValue, s.RecommendedGUCS.AutoExplainLogVerbose.String)
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

func getLogFormatValue(state *s.SetupState, currValue string) (string, error) {
	if state.Inputs.Scripted {
		if !state.Inputs.GUCS.AutoExplainLogFormat.Valid {
			panic("auto_explain.log_format setting needs review but was not provided")
		}
		logFormat := state.Inputs.GUCS.AutoExplainLogFormat.String
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

func getLogMinDurationValue(state *s.SetupState, currValue string) (string, error) {
	if state.Inputs.Scripted {
		if !state.Inputs.GUCS.AutoExplainLogMinDuration.Valid {
			panic("auto_explain.log_min_duration setting needs review but was not provided")
		}
		return strconv.Itoa(int(state.Inputs.GUCS.AutoExplainLogMinDuration.Int64)), nil
	}

	var durationOpts = []string{
		fmt.Sprintf("set to %dms (recommended inital value; will be saved to Postgres)", s.RecommendedGUCS.AutoExplainLogMinDuration.Int64),
		"set to other value...",
		fmt.Sprintf("leave at %sms", currValue),
	}
	var durationOptIdx int
	err := survey.AskOne(&survey.Select{
		Message: fmt.Sprintf("Setting auto_explain.log_min_duration is currently set to '%s ms'", currValue),
		Help:    fmt.Sprintf("Threshold to log EXPLAIN plans, in ms; recommend %d, must be at least 10", s.RecommendedGUCS.AutoExplainLogMinDuration.Int64),
		Options: durationOpts,
	}, &durationOptIdx)
	if err != nil {
		return "", err
	}

	if durationOptIdx == 0 {
		return strconv.Itoa(int(s.RecommendedGUCS.AutoExplainLogMinDuration.Int64)), nil
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

func getLogNestedStatements(state *s.SetupState, currValue string) (string, error) {
	if state.Inputs.Scripted {
		if !state.Inputs.GUCS.AutoExplainLogNestedStatements.Valid {
			panic("auto_explain.log_nested_statements setting needs review but was not provided")
		}
		return state.Inputs.GUCS.AutoExplainLogNestedStatements.String, nil
	}

	var logNestedIdx int
	opts, optLabels := getBooleanOpts(currValue, s.RecommendedGUCS.AutoExplainLogNestedStatements.String)
	err := survey.AskOne(&survey.Select{
		Message: fmt.Sprintf("Setting auto_explain.log_nested_statements is currently set to '%s'", currValue),
		Help:    "Consider statements executed inside functions for logging",
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

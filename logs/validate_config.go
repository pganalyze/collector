package logs

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pganalyze/collector/state"
)

const MinSupportedLogMinDurationStatement = 100

func ValidateLogCollectionConfig(server *state.Server, settings []state.PostgresSetting) (bool, string) {
	var disabled = false
	var disabledReasons []string
	if server.Config.DisableLogs {
		disabled = true
		disabledReasons = append(disabledReasons, "the collector setting disable_logs or environment variable PGA_DISABLE_LOGS is set")
	}

	if !disabled {
		for _, setting := range settings {
			if setting.Name == "log_min_duration_statement" && setting.CurrentValue.Valid {
				numVal, err := strconv.Atoi(setting.CurrentValue.String)
				if err != nil {
					continue
				}
				if numVal < MinSupportedLogMinDurationStatement {
					disabled = true
					disabledReasons = append(disabledReasons,
						fmt.Sprintf("log_min_duration_statement is set to '%d', below minimum supported threshold '%d'", numVal, MinSupportedLogMinDurationStatement),
					)
				}
			} else if setting.Name == "log_duration" && setting.CurrentValue.Valid {
				if setting.CurrentValue.String == "on" {
					disabled = true
					disabledReasons = append(disabledReasons, "log_duration is set to unsupported value 'on'")
				}
			} else if setting.Name == "log_statement" && setting.CurrentValue.Valid {
				if setting.CurrentValue.String == "all" {
					disabled = true
					disabledReasons = append(disabledReasons, "log_statement is set to unsupported value 'all'")
				}
			}
		}
	}

	return disabled, strings.Join(disabledReasons, "; ")
}

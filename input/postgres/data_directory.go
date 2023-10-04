package postgres

import (
	"github.com/pganalyze/collector/state"
)

// GetDataDirectory - Finds the data_directory in the list of settings and returns
// its current value. Returns an empty string if not found (e.g., the setting is
// not present due to permissions issues)
func GetDataDirectory(server *state.Server, settings []state.PostgresSetting) string {
	for _, setting := range settings {
		if setting.Name == "data_directory" && setting.CurrentValue.Valid {
			return setting.CurrentValue.String
		}
	}
	return ""
}

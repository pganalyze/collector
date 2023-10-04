package postgres

import (
	"github.com/pganalyze/collector/state"
)

// GetDataDirectory - Finds the data_directory in the list of settings and returns
// its current value. Returns an empty string if not found (e.g., the setting is
// not present due to permissions issues)
func GetDataDirectory(server *state.Server, settings []state.PostgresSetting) string {
	// N.B.: we're not taking a mutex here, because the data directory will not
	// change location, so we only run this once at collector startup
	for _, setting := range settings {
		if setting.Name == "data_directory" && setting.CurrentValue.Valid {
			return setting.CurrentValue.String
		}
	}
	return ""
}

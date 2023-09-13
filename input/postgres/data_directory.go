package postgres

import (
	"github.com/pganalyze/collector/state"
)

// SetDataDirectory - Finds the data_directory in the list of settings and sets
// the Server.Config.DataDirectory field to its value. Does nothing if the
// setting is not found (e.g., due to permission issues).
func SetDataDirectory(server *state.Server, settings []state.PostgresSetting) {
	// N.B.: we're not taking a mutex here, because the data directory will not change location,
	// so we only run this once at collector startup
	for _, setting := range settings {
		if setting.Name == "data_directory" && setting.CurrentValue.Valid {
			server.Config.DataDirectory = setting.CurrentValue.String
			return
		}
	}
}

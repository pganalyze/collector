package steps

import (
	"fmt"

	s "github.com/pganalyze/collector/setup/state"
)

var CheckPostgresVersion = &s.Step{
	Description: "Check Postgres version",
	Check: func(state *s.SetupState) (bool, error) {
		row, err := state.QueryRunner.QueryRow("SELECT current_setting('server_version'), current_setting('server_version_num')::integer")
		if err != nil {
			return false, err
		}
		state.PGVersionStr = row.GetString(0)
		state.PGVersionNum = row.GetInt(1)

		if state.PGVersionNum < 100000 {
			return false, fmt.Errorf("not supported for Postgres versions older than 10; found %s", state.PGVersionStr)
		}

		if state.PGVersionNum >= 120000 {
			state.QueryRunner.EnableCSV()
		}

		return true, nil
	},
}

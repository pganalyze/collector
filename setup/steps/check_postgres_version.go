package steps

import (
	"fmt"

	"github.com/pganalyze/collector/setup/state"
)

var CheckPostgresVersion = &state.Step{
	ID:          "check_postgres_version",
	Description: "Check whether this Postgres version is supported by pganalyze guided setup",
	Check: func(s *state.SetupState) (bool, error) {
		row, err := s.QueryRunner.QueryRow("SELECT current_setting('server_version'), current_setting('server_version_num')::integer")
		if err != nil {
			return false, err
		}
		s.PGVersionStr = row.GetString(0)
		s.PGVersionNum = row.GetInt(1)

		if s.PGVersionNum < 100000 {
			return false, fmt.Errorf("not supported for Postgres versions older than 10; found %s", s.PGVersionStr)
		}
		return true, nil
	},
}

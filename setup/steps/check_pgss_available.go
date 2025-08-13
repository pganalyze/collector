package steps

import (
	"errors"

	"github.com/pganalyze/collector/setup/query"
	"github.com/pganalyze/collector/setup/state"
)

var ConfirmPgssAvailable = &state.Step{
	ID:          "check_pgss_available",
	Description: "Confirm the pg_stat_statements extension is ready to be installed",
	Check: func(s *state.SetupState) (bool, error) {
		row, err := s.QueryRunner.QueryRow(
			"SELECT true FROM pg_available_extensions WHERE name = 'pg_stat_statements'",
		)
		if err == query.ErrNoRows {
			return false, nil
		} else if err != nil {
			return false, err
		}
		return row.GetBool(0), nil
	},
	Run: func(s *state.SetupState) error {
		return errors.New("contrib extension pg_stat_statements is not available")
	},
}

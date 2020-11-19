package steps

import (
	"errors"
	"fmt"

	"github.com/pganalyze/collector/setup/query"
	s "github.com/pganalyze/collector/setup/state"
)

var CheckPgssAvailable = &s.Step{
	Description: "Prepare for pg_stat_statements install",
	Check: func(state *s.SetupState) (bool, error) {
		row, err := state.QueryRunner.QueryRow(
			fmt.Sprintf(
				"SELECT true FROM pg_available_extensions WHERE name = 'pg_stat_statements'",
			),
		)
		if err == query.ErrNoRows {
			return false, nil
		} else if err != nil {
			return false, err
		}
		return row.GetBool(0), nil
	},
	Run: func(state *s.SetupState) error {
		// TODO: install contrib package on systems where packaged separately
		return errors.New("extension pg_stat_statements is not available")
	},
}

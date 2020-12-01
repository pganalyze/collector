package steps

import (
	"errors"

	s "github.com/pganalyze/collector/setup/state"
)

var CheckReplicationStatus = &s.Step{
	Description: "Check whether the database is a replica, which is currently unsupported by pganalyze guided setup",
	Check: func(state *s.SetupState) (bool, error) {
		result, err := state.QueryRunner.QueryRow("SELECT pg_is_in_recovery()")
		if err != nil {
			return false, err
		}
		isInRecovery := result.GetBool(0)

		if isInRecovery {
			return false, errors.New("Postgres server is a replica; this is currently not supported")
		}
		return true, nil
	},
}

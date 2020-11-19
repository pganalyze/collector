package steps

import (
	"errors"

	s "github.com/pganalyze/collector/setup/state"
)

var CheckReplicationStatus = &s.Step{
	Description: "Check replication status",
	Check: func(state *s.SetupState) (bool, error) {
		result, err := state.QueryRunner.QueryRow("SELECT pg_is_in_recovery()")
		if err != nil {
			return false, err
		}
		isReplicationTarget := result.GetBool(0)

		if isReplicationTarget {
			return false, errors.New("not supported for replicas")
		}
		return true, nil
	},
}

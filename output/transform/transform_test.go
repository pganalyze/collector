package transform_test

import (
	"encoding/json"
	"testing"

	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/output/transform"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func TestStatements(t *testing.T) {
	newState := state.State{}
	diffState := state.DiffState{
		Statements: []state.DiffedPostgresStatement{
			state.DiffedPostgresStatement{
				NormalizedQuery: "SELECT 1",
				Calls:           1,
			},
			state.DiffedPostgresStatement{
				NormalizedQuery: "SELECT * FROM test",
				Calls:           13,
			},
		},
	}

	actual := transform.StateToSnapshot(newState, diffState)
	actualJSON, _ := json.Marshal(actual)

	fp1 := util.FingerprintQuery("SELECT 1")
	fp2 := util.FingerprintQuery("SELECT * FROM test")

	expected := pganalyze_collector.FullSnapshot{
		QueryReferences: []*pganalyze_collector.QueryReference{
			&pganalyze_collector.QueryReference{
				DatabaseIdx: 0,
				RoleIdx:     0,
				Fingerprint: fp1[:],
			},
			&pganalyze_collector.QueryReference{
				DatabaseIdx: 0,
				RoleIdx:     0,
				Fingerprint: fp2[:],
			},
		},
		QueryInformations: []*pganalyze_collector.QueryInformation{
			&pganalyze_collector.QueryInformation{
				QueryIdx:        0,
				NormalizedQuery: "SELECT 1",
				QueryIds:        []int64{0},
			},
			&pganalyze_collector.QueryInformation{
				QueryIdx:        1,
				NormalizedQuery: "SELECT * FROM test",
				QueryIds:        []int64{0},
			},
		},
		QueryStatistics: []*pganalyze_collector.QueryStatistic{
			&pganalyze_collector.QueryStatistic{
				QueryIdx: 0,
				Calls:    1,
			},
			&pganalyze_collector.QueryStatistic{
				QueryIdx: 1,
				Calls:    13,
			},
		},
	}
	expectedJSON, _ := json.Marshal(expected)

	if string(expectedJSON) != string(actualJSON) {
		t.Errorf("\nExpected:%+v\n\tActual: %+v\n\n", string(expectedJSON), string(actualJSON))
	}
}

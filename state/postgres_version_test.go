package state_test

import (
	"testing"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func TestSyncPostgresVersion(t *testing.T) {
	server := state.MakeServer(config.ServerConfig{}, false)
	if actual := server.GetLastPostgresVersion(); actual.Numeric != 0 {
		t.Errorf("expected unknown Postgres version to be zero, got %d", actual.Numeric)
	}

	server.SyncPostgresVersion(state.PostgresVersion{Numeric: 170005, Short: "17.5", IsAwsAurora: true})
	if actual := server.GetLastPostgresVersion(); actual.Numeric != 170005 || actual.Short != "17.5" || !actual.IsAwsAurora {
		t.Errorf("expected Postgres version 17.5 on Aurora, got %+v", actual)
	}

	// Simulate a major version upgrade detected by a subsequent snapshot
	server.SyncPostgresVersion(state.PostgresVersion{Numeric: 180001, Short: "18.1"})
	if actual := server.GetLastPostgresVersion(); actual.Numeric != 180001 || actual.IsAwsAurora {
		t.Errorf("expected Postgres version 18.1, got %+v", actual)
	}
}

func TestFingerprintOptsForVersion(t *testing.T) {
	tests := []struct {
		versionNum int
		expected   util.FingerprintOpts
	}{
		{0, util.FingerprintOptsDefault}, // Unknown version assumes Postgres 18+
		{state.PostgresVersion10, util.FingerprintOptsPG17Compat},
		{170005, util.FingerprintOptsPG17Compat},
		{state.PostgresVersion18, util.FingerprintOptsDefault},
		{180001, util.FingerprintOptsDefault},
	}
	for _, test := range tests {
		if actual := state.FingerprintOptsForVersion(test.versionNum); actual != test.expected {
			t.Errorf("FingerprintOptsForVersion(%d): expected %v, got %v", test.versionNum, test.expected, actual)
		}
	}
}

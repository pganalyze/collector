package transform

import (
	"testing"

	"github.com/guregu/null"
	snapshot "github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
)

func TestTransformPostgresConfigMarksNonUTF8SettingValues(t *testing.T) {
	ts := state.TransientState{
		Settings: []state.PostgresSetting{
			// The real pgsodium case: a binary boot_val that isn't valid UTF-8.
			{Name: "pgsodium.getkey_script", BootValue: null.StringFrom("p;\xd3\xf3DY")},
			// A normal setting must pass through untouched.
			{Name: "work_mem", CurrentValue: null.StringFrom("4MB"), BootValue: null.StringFrom("4MB")},
		},
	}

	s := transformPostgresConfig(snapshot.FullSnapshot{}, ts)

	byName := map[string]*snapshot.Setting{}
	for _, st := range s.Settings {
		byName[st.Name] = st
	}

	if got := byName["pgsodium.getkey_script"].BootValue.Value; got != settingValueNonUTF8 {
		t.Errorf("non-UTF-8 boot value = %q, want marker %q", got, settingValueNonUTF8)
	}
	if got := byName["work_mem"].BootValue.Value; got != "4MB" {
		t.Errorf("valid boot value should be untouched, got %q", got)
	}
	if got := byName["work_mem"].CurrentValue; got != "4MB" {
		t.Errorf("valid current value should be untouched, got %q", got)
	}
}

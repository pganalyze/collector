package transform

import (
	"testing"

	"github.com/guregu/null"
	snapshot "github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
)

func TestTransformPostgresConfigMasksAndMarksSettingValues(t *testing.T) {
	ts := state.TransientState{
		Settings: []state.PostgresSetting{
			// Denylisted GUC: masked regardless of UTF-8 validity. Its boot_val here is
			// invalid UTF-8 while its current value is *valid* UTF-8 garbage - both must
			// be masked so the value doesn't flip snapshot-to-snapshot.
			{Name: "pgsodium.getkey_script", CurrentValue: null.StringFrom("Zm9vYmFy"), BootValue: null.StringFrom("p;\xd3\xf3DY")},
			// Not denylisted but invalid UTF-8: falls back to the generic marker.
			{Name: "some_ext.blob", BootValue: null.StringFrom("x\xffy")},
			// A normal setting must pass through untouched.
			{Name: "work_mem", CurrentValue: null.StringFrom("4MB"), BootValue: null.StringFrom("4MB")},
		},
	}

	s := transformPostgresConfig(snapshot.FullSnapshot{}, ts)

	byName := map[string]*snapshot.Setting{}
	for _, st := range s.Settings {
		byName[st.Name] = st
	}

	// Denylisted: both value fields masked, including the valid-UTF-8 one.
	if got := byName["pgsodium.getkey_script"].BootValue.Value; got != settingValueMasked {
		t.Errorf("denylisted boot value = %q, want mask %q", got, settingValueMasked)
	}
	if got := byName["pgsodium.getkey_script"].CurrentValue; got != settingValueMasked {
		t.Errorf("denylisted current value (valid UTF-8) = %q, want mask %q", got, settingValueMasked)
	}
	// Not denylisted but invalid UTF-8: generic marker.
	if got := byName["some_ext.blob"].BootValue.Value; got != settingValueNonUTF8 {
		t.Errorf("non-UTF-8 boot value = %q, want marker %q", got, settingValueNonUTF8)
	}
	// Normal setting untouched.
	if got := byName["work_mem"].BootValue.Value; got != "4MB" {
		t.Errorf("valid boot value should be untouched, got %q", got)
	}
	if got := byName["work_mem"].CurrentValue; got != "4MB" {
		t.Errorf("valid current value should be untouched, got %q", got)
	}
}

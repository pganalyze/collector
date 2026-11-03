package output

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	snapshot "github.com/pganalyze/collector/output/pganalyze_collector"
)

// bad is the exact invalid-UTF-8 value observed in pgsodium.getkey_script's boot_val
// on a live Supabase instance.
const bad = "p;\xd3\xf3DY"

func TestMarshalSnapshotRejectsInvalidUTF8(t *testing.T) {
	fs := &snapshot.FullSnapshot{
		Settings: []*snapshot.Setting{
			{Name: "pgsodium.getkey_script", BootValue: &snapshot.NullString{Valid: true, Value: bad}},
		},
	}
	_, err := marshalSnapshot(fs)
	if err == nil {
		t.Fatal("expected marshalSnapshot to reject invalid UTF-8")
	}
	if !strings.Contains(err.Error(), ".settings[0].boot_value.value") {
		t.Errorf("error should name the offending field, got: %v", err)
	}
}

func TestFindInvalidUTF8(t *testing.T) {
	fs := &snapshot.FullSnapshot{
		Settings: []*snapshot.Setting{
			{Name: "work_mem", CurrentValue: "4MB"},
			{Name: "pgsodium.getkey_script", BootValue: &snapshot.NullString{Valid: true, Value: bad}},
		},
		System: &snapshot.System{
			SystemInformation: &snapshot.SystemInformation{
				ResourceTags: map[string]string{"good-key": bad, bad: "good-value"},
			},
		},
	}

	got := findInvalidUTF8(fs)
	sort.Strings(got)
	want := []string{
		`.settings[1].boot_value.value`,
		`.system.system_information.resource_tags["good-key"]`,
		`.system.system_information.resource_tags["p;\xd3\xf3DY"] (key)`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("findInvalidUTF8 = %q, want %q", got, want)
	}

	if got := findInvalidUTF8(&snapshot.FullSnapshot{Settings: []*snapshot.Setting{{Name: "work_mem", CurrentValue: "4MB"}}}); len(got) != 0 {
		t.Errorf("expected no paths for valid snapshot, got %q", got)
	}
}

package output

import (
	"log"
	"os"
	"testing"
	"unicode/utf8"

	snapshot "github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/util"
	"google.golang.org/protobuf/proto"
)

// bad is the exact invalid-UTF-8 value observed in pgsodium.getkey_script's boot_val
// on a live Supabase instance.
const bad = "p;\xd3\xf3DY"

func TestMarshalSnapshotRecoversNestedInvalidUTF8(t *testing.T) {
	// The real case: FullSnapshot -> repeated Setting -> BootValue (message) -> Value.
	fs := &snapshot.FullSnapshot{
		Settings: []*snapshot.Setting{
			{Name: "pgsodium.getkey_script", BootValue: &snapshot.NullString{Valid: true, Value: bad}},
		},
	}

	if _, err := proto.Marshal(fs); err == nil {
		t.Fatal("expected raw proto.Marshal to reject the invalid UTF-8")
	}

	logger := &util.Logger{Destination: log.New(os.Stderr, "", log.LstdFlags)}
	data, err := marshalSnapshot(fs, logger)
	if err != nil {
		t.Fatalf("marshalSnapshot should recover, got: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty marshaled data")
	}
	if v := fs.Settings[0].BootValue.Value; !utf8.ValidString(v) {
		t.Errorf("boot value still invalid after scrub: %q", v)
	}
}

func TestSanitizeInvalidUTF8Map(t *testing.T) {
	// Map with both a bad value and a bad key.
	si := &snapshot.SystemInformation{
		ResourceTags: map[string]string{"good-key": bad, bad: "good-value"},
	}
	if _, err := proto.Marshal(si); err == nil {
		t.Fatal("expected raw proto.Marshal to reject the invalid UTF-8 map entry")
	}

	sanitizeInvalidUTF8(si.ProtoReflect())

	if _, err := proto.Marshal(si); err != nil {
		t.Fatalf("marshal should succeed after scrub, got: %v", err)
	}
	if len(si.ResourceTags) != 2 {
		t.Errorf("expected 2 map entries after scrub, got %d", len(si.ResourceTags))
	}
	for k, v := range si.ResourceTags {
		if !utf8.ValidString(k) {
			t.Errorf("map key still invalid: %q", k)
		}
		if !utf8.ValidString(v) {
			t.Errorf("map value still invalid: %q", v)
		}
	}
}

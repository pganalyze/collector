package util_test

import (
	"encoding/binary"
	"encoding/hex"
	"reflect"
	"testing"

	"github.com/pganalyze/collector/util"
)

var fingerprintTests = []struct {
	input    string
	expected string
}{
	{
		"SELECT 1",
		"50fde20626009aba",
	},
	{
		"SELINVALID",
		"8e687e2b4dbec30c",
	},
	{
		"INSERT INTO x (a, b) VALUES (",
		"7a0d78e21e354216",
	},
	{
		"SELECT )",
		"4f75277b70af299c",
	},
	{
		"DELETE FROM x WHERE \"id\" IN ($1)",
		"6b0d33245a74c535",
	},
	{
		"DELETE FROM x WHERE \"id\" IN (12450548, 12450547, 12450546, 124",
		"6b0d33245a74c535",
	},
	{
		"DELETE FROM x WHERE \"id\" IN (15485697, 15485694, 15485693, 154",
		"6b0d33245a74c535",
	},
	{
		"SELECT * FROM x WHERE y = ''",
		"4ff39426bd074231",
	},
	{
		"SELECT * FROM x WHERE y = '",
		"4ff39426bd074231",
	},
	{
		"SELECT * FROM x AS \"abc\"",
		"4d956249fc96ed55",
	},
	{
		"SELECT * FROM x AS \"a",
		"4d956249fc96ed55",
	},
}

func TestFingerprint(t *testing.T) {
	for _, test := range fingerprintTests {
		fp := util.FingerprintQuery(test.input, "none", -1, util.FingerprintOptsPG17Compat)
		actual := make([]byte, 8)
		binary.BigEndian.PutUint64(actual, fp)
		expected, _ := hex.DecodeString(test.expected)

		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("Fingerprint(%s)\nexpected %s\nactual %s\n\n", test.input, test.expected, hex.EncodeToString(actual))
		}
	}
}

var fingerprintOptsTests = []struct {
	a         string
	b         string
	equalPG17 bool
	equalPG18 bool
}{
	// Postgres 18+ fingerprints the alias instead of the relation name
	{"SELECT * FROM x AS y", "SELECT * FROM z AS y", false, true},
	{"SELECT * FROM x AS y", "SELECT * FROM x AS z", true, false},
	// Postgres 18+ ignores schema names in SELECT/DML statements
	{"SELECT * FROM x", "SELECT * FROM public.x", false, true},
	// Schema names are always considered in utility statements
	{"VACUUM x", "VACUUM public.x", false, false},
}

func TestFingerprintOpts(t *testing.T) {
	for _, test := range fingerprintOptsTests {
		equalPG17 := util.FingerprintQuery(test.a, "none", -1, util.FingerprintOptsPG17Compat) == util.FingerprintQuery(test.b, "none", -1, util.FingerprintOptsPG17Compat)
		if equalPG17 != test.equalPG17 {
			t.Errorf("PG17 compat: Fingerprint(%s) == Fingerprint(%s)\nexpected %v\nactual %v\n\n", test.a, test.b, test.equalPG17, equalPG17)
		}
		equalPG18 := util.FingerprintQuery(test.a, "none", -1, util.FingerprintOptsDefault) == util.FingerprintQuery(test.b, "none", -1, util.FingerprintOptsDefault)
		if equalPG18 != test.equalPG18 {
			t.Errorf("Default: Fingerprint(%s) == Fingerprint(%s)\nexpected %v\nactual %v\n\n", test.a, test.b, test.equalPG18, equalPG18)
		}
	}
}

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
		fp := util.FingerprintQuery(test.input, "none", -1)
		actual := make([]byte, 8)
		binary.BigEndian.PutUint64(actual, fp)
		expected, _ := hex.DecodeString(test.expected)

		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("Fingerprint(%s)\nexpected %s\nactual %s\n\n", test.input, test.expected, hex.EncodeToString(actual))
		}
	}
}

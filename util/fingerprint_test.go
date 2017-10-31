package util_test

import (
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
		"02a281c251c3a43d2fe7457dff01f76c5cc523f8c8",
	},
	{
		"SELINVALID",
		"ee5571410c33aa5c2e7a9d424eb44fb3d22fec37be",
	},
	{
		"INSERT INTO x (a, b) VALUES (",
		"ee47d014d69c6f4aae4a597ea1430628396ecce69a",
	},
	{
		"SELECT )",
		"ee270a7ad0592e369455f1dac995cef3e35556411e",
	},
	{
		"DELETE FROM x WHERE \"id\" IN (?)",
		"02a52764bd41f8f4ca6a399039553faee86d2e8c82",
	},
	{
		"DELETE FROM x WHERE \"id\" IN (12450548, 12450547, 12450546, 124",
		"02a52764bd41f8f4ca6a399039553faee86d2e8c82",
	},
	{
		"DELETE FROM x WHERE \"id\" IN (15485697, 15485694, 15485693, 154",
		"02a52764bd41f8f4ca6a399039553faee86d2e8c82",
	},
	{
		"SELECT * FROM x WHERE y = ''",
		"02000980540197a51fb2e6736a28747cf6dbe52afd",
	},
	{
		"SELECT * FROM x WHERE y = '",
		"02000980540197a51fb2e6736a28747cf6dbe52afd",
	},
	{
		"SELECT * FROM x AS \"abc\"",
		"027a97a97ec7663a04add95792e3e9d71a6411ee31",
	},
	{
		"SELECT * FROM x AS \"a",
		"027a97a97ec7663a04add95792e3e9d71a6411ee31",
	},
}

func TestFingerprint(t *testing.T) {
	for _, test := range fingerprintTests {
		fp := util.FingerprintQuery(test.input)
		actual := fp[:]
		expected, _ := hex.DecodeString(test.expected)

		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("Fingerprint(%s)\nexpected %s\nactual %s\n\n", test.input, test.expected, hex.EncodeToString(actual))
		}
	}
}

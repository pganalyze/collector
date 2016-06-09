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
		"018e1acac181c6d28f4a923392cf1c4eda49ee4cd2",
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
		"016df58609447fc943efc2d07800fb57aa912cdfb7",
	},
	{
		"DELETE FROM x WHERE \"id\" IN (12450548, 12450547, 12450546, 124",
		"016df58609447fc943efc2d07800fb57aa912cdfb7",
	},
	{
		"DELETE FROM x WHERE \"id\" IN (15485697, 15485694, 15485693, 154",
		"016df58609447fc943efc2d07800fb57aa912cdfb7",
	},
	{
		"SELECT * FROM x WHERE y = ''",
		"01789da74d0b82bf2fb200ece6016136e57259fcf5",
	},
	{
		"SELECT * FROM x WHERE y = '",
		"01789da74d0b82bf2fb200ece6016136e57259fcf5",
	},
	{
		"SELECT * FROM x AS \"abc\"",
		"01e03715d33b89af5262a23d33cea681c474d99acf",
	},
	{
		"SELECT * FROM x AS \"a",
		"01e03715d33b89af5262a23d33cea681c474d99acf",
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

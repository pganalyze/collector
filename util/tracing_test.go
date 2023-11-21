package util_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/pganalyze/collector/util"
)

var timeFromStrTests = []struct {
	input     string
	expected  time.Time
	expectErr bool
}{
	{
		"1697666938.629721234",
		time.Unix(1697666938, 629721234).UTC(),
		false,
	},
	{
		"1697666938.629",
		time.Unix(1697666938, 629000000).UTC(),
		false,
	},
	{
		"1697666938",
		time.Unix(1697666938, 0).UTC(),
		false,
	},
	{
		"",
		time.Time{},
		true,
	},
	{
		"not a time",
		time.Time{},
		true,
	},
	{
		"1697666938.baddecimal",
		time.Time{},
		true,
	},
	{
		"1697666938.6297212340000", // nsec too long
		time.Time{},
		true,
	},
}

func TestTimeFromStr(t *testing.T) {
	for _, test := range timeFromStrTests {
		actual, err := util.TimeFromStr(test.input)
		if (err != nil) != test.expectErr {
			t.Errorf("TimeFromStr(%s): expected err: %t; actual: %s", test.input, test.expectErr, err)
		}
		if !reflect.DeepEqual(actual, test.expected) {
			t.Errorf("TimeFromStr(%s): expected %v; actual %v", test.input, test.expected, actual)
		}
	}
}

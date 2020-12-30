package util_test

import (
	"reflect"
	"testing"

	"github.com/pganalyze/collector/util"
)

var isUtilTests = []struct {
	input     string
	expected  []bool
	expectErr bool
}{
	{
		"SELECT 1",
		[]bool{false},
		false,
	},
	{
		"INSERT INTO my_table VALUES(123)",
		[]bool{false},
		false,
	},
	{
		"UPDATE my_table SET foo = 123",
		[]bool{false},
		false,
	},
	{
		"DELETE FROM my_table",
		[]bool{false},
		false,
	},
	{
		"SHOW fsync",
		[]bool{true},
		false,
	},
	{
		"SET fsync = off",
		[]bool{true},
		false,
	},
	{
		"SELECT 1; SELECT 2;",
		[]bool{false, false},
		false,
	},
	{
		"SELECT 1; SHOW fsync;",
		[]bool{false, true},
		false,
	},
	{
		"totally not valid sql",
		nil,
		true,
	},
}

func TestIsUtilityStmt(t *testing.T) {
	for _, test := range isUtilTests {
		actual, err := util.IsUtilityStmt(test.input)
		if (err != nil) != test.expectErr {
			t.Errorf("IsUtilityStmt(%s): expected err: %t; actual: %s", test.input, test.expectErr, err)
		}
		if !reflect.DeepEqual(actual, test.expected) {
			t.Errorf("IsUtilityStmt(%s): expected %v; actual %v", test.input, test.expected, actual)
		}
	}
}

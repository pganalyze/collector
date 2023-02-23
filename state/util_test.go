package state

import (
	"reflect"
	"testing"
)

var xidToXid8Tests = []struct {
	currentXactId Xid8
	xid           Xid
	expected      Xid8
}{
	{
		(2 << 32) + 12345,
		4294967295,
		(1 << 32) + 4294967295,
	},
	{
		(2 << 32) + 3,
		2147483652,
		(1 << 32) + 2147483652,
	},
	{
		(2 << 32) + 12345,
		3,
		(2 << 32) + 3,
	},
	{
		(2 << 32) + 12345,
		12345,
		(2 << 32) + 12345,
	},
	{
		(2 << 32) + 12345,
		0,
		0,
	},
	{
		0,
		12345,
		0,
	},
}

func TestXidToXid8(t *testing.T) {
	for _, test := range xidToXid8Tests {
		actual := XidToXid8(test.xid, test.currentXactId)

		if !reflect.DeepEqual(actual, test.expected) {
			t.Errorf("Converts Xid (%d) to Xid8 with currentXactId (%d) \nexpected %d\nactual %d\n\n", test.xid, test.currentXactId, test.expected, actual)
		}
	}
}

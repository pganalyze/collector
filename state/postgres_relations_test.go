package state

import (
	"reflect"
	"testing"
)

var fullFrozenXIDTests = []struct {
	currentXactId int64
	frozenXID     uint32
	expected      int64
}{
	{
		(2 << 32) + 12345,
		uint32(4294967295),
		(1 << 32) + 4294967295,
	},
	{
		(2 << 32) + 3,
		uint32(2147483652),
		(1 << 32) + 2147483652,
	},
	{
		(2 << 32) + 12345,
		uint32(3),
		(2 << 32) + 3,
	},
	{
		(2 << 32) + 12345,
		uint32(12345),
		(2 << 32) + 12345,
	},
}

func TestFullFrozenXID(t *testing.T) {
	relation := PostgresRelation{}
	for _, test := range fullFrozenXIDTests {
		relation.FrozenXID = Xid(test.frozenXID)
		actual := relation.FullFrozenXID(test.currentXactId)

		if !reflect.DeepEqual(actual, test.expected) {
			t.Errorf("FullFrozenXID with frozenXID (%d) and currentXactId (%d) \nexpected %d\nactual %d\n\n", test.frozenXID, test.currentXactId, test.expected, actual)
		}
	}
}

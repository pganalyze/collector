package state

import (
	"testing"
	"time"
)

var startedEarlier = time.Date(2026, 5, 14, 23, 30, 0, 0, time.UTC)
var startedLater = time.Date(2026, 5, 14, 23, 45, 0, 0, time.UTC)

var instanceIdentityMatchTests = []struct {
	name     string
	prev     PostgresInstanceIdentity
	curr     PostgresInstanceIdentity
	expected bool
}{
	{
		"same instance and postmaster",
		PostgresInstanceIdentity{PostmasterStartTime: startedEarlier, ServerAddr: "10.1.16.97"},
		PostgresInstanceIdentity{PostmasterStartTime: startedEarlier, ServerAddr: "10.1.16.97"},
		true,
	},
	{
		// An Aurora failover pointing the reader endpoint at the old writer
		"different instance",
		PostgresInstanceIdentity{PostmasterStartTime: startedEarlier, ServerAddr: "10.1.16.97"},
		PostgresInstanceIdentity{PostmasterStartTime: startedEarlier, ServerAddr: "10.1.20.14"},
		false,
	},
	{
		// The same instance, restarted, so its counters started over
		"same instance, restarted",
		PostgresInstanceIdentity{PostmasterStartTime: startedEarlier, ServerAddr: "10.1.16.97"},
		PostgresInstanceIdentity{PostmasterStartTime: startedLater, ServerAddr: "10.1.16.97"},
		false,
	},
	{
		// Unix socket connections have no address, so the start time decides
		"local connection, same postmaster",
		PostgresInstanceIdentity{PostmasterStartTime: startedEarlier},
		PostgresInstanceIdentity{PostmasterStartTime: startedEarlier},
		true,
	},
	{
		"local connection, restarted",
		PostgresInstanceIdentity{PostmasterStartTime: startedEarlier},
		PostgresInstanceIdentity{PostmasterStartTime: startedLater},
		false,
	},
	{
		// We keep diffing as before rather than dropping data over an identity we
		// were unable to determine
		"unknown previous identity",
		PostgresInstanceIdentity{},
		PostgresInstanceIdentity{PostmasterStartTime: startedLater, ServerAddr: "10.1.20.14"},
		true,
	},
	{
		"unknown current identity",
		PostgresInstanceIdentity{PostmasterStartTime: startedEarlier, ServerAddr: "10.1.16.97"},
		PostgresInstanceIdentity{},
		true,
	},
}

func TestPostgresInstanceIdentityMatches(t *testing.T) {
	for _, test := range instanceIdentityMatchTests {
		actual := test.prev.Matches(test.curr)
		if actual != test.expected {
			t.Errorf("%s: Matches(%+v, %+v)\nexpected %t\nactual %t\n", test.name, test.prev, test.curr, test.expected, actual)
		}

		reverse := test.curr.Matches(test.prev)
		if reverse != test.expected {
			t.Errorf("%s: Matches is not symmetric, reverse gave %t, expected %t\n", test.name, reverse, test.expected)
		}
	}
}

func TestPostgresInstanceIdentityIsZero(t *testing.T) {
	if !(PostgresInstanceIdentity{}).IsZero() {
		t.Errorf("expected the zero identity to report IsZero")
	}
	if (PostgresInstanceIdentity{PostmasterStartTime: startedEarlier}).IsZero() {
		t.Errorf("expected an identity with a start time not to report IsZero")
	}
	if (PostgresInstanceIdentity{ServerAddr: "10.1.16.97"}).IsZero() {
		t.Errorf("expected an identity with an address not to report IsZero")
	}
}

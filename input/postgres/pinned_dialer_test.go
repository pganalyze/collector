package postgres

import (
	"testing"
)

var canPinHostTests = []struct {
	host     string
	expected bool
}{
	{"prd-uswest2-core0-canary-db-cluster.cluster-ro-cvdyfnmmcfzd.us-west-2.rds.amazonaws.com", true},
	{"db.example.com", true},
	{"localhost", true},
	// Already an address, so there is no lookup result to share
	{"10.1.16.97", false},
	{"::1", false},
	{"127.0.0.1", false},
	// Unix socket directory
	{"/var/run/postgresql", false},
	// Nothing configured, pq falls back to localhost on its own
	{"", false},
}

func TestCanPinHost(t *testing.T) {
	for _, test := range canPinHostTests {
		actual := canPinHost(test.host)
		if actual != test.expected {
			t.Errorf("canPinHost(%q)\nexpected %t\nactual %t\n", test.host, test.expected, actual)
		}
	}
}

var pinnedDialerRedirectTests = []struct {
	name     string
	network  string
	address  string
	expected string
}{
	{
		"keeps the port pq derived from the connection parameters",
		"tcp",
		"prd-uswest2-core0-canary-db-cluster.cluster-ro-cvdyfnmmcfzd.us-west-2.rds.amazonaws.com:5432",
		"10.1.16.97:5432",
	},
	{
		"handles a non-default port",
		"tcp",
		"db.example.com:6432",
		"10.1.16.97:6432",
	},
	{
		// pq builds a socket path rather than a host:port pair for these
		"leaves Unix socket paths alone",
		"unix",
		"/var/run/postgresql/.s.PGSQL.5432",
		"/var/run/postgresql/.s.PGSQL.5432",
	},
	{
		// Should not happen, but we would rather dial what pq asked for than
		// produce a malformed address
		"passes through an address without a port",
		"tcp",
		"db.example.com",
		"db.example.com",
	},
}

func TestPinnedDialerRedirect(t *testing.T) {
	d := pinnedDialer{addr: "10.1.16.97"}

	for _, test := range pinnedDialerRedirectTests {
		actual := d.redirect(test.network, test.address)
		if actual != test.expected {
			t.Errorf("%s: redirect(%q, %q)\nexpected %s\nactual %s\n", test.name, test.network, test.address, test.expected, actual)
		}
	}
}

func TestPinnedDialerRedirectIPv6(t *testing.T) {
	d := pinnedDialer{addr: "2600:1f14::1"}

	actual := d.redirect("tcp", "db.example.com:5432")
	expected := "[2600:1f14::1]:5432"
	if actual != expected {
		t.Errorf("expected an IPv6 pin to be bracketed\nexpected %s\nactual %s\n", expected, actual)
	}
}

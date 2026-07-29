package postgres

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
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

// listen - Starts a listener standing in for a Postgres instance, and returns it
// along with its address and port.
func listen(t *testing.T) (net.Listener, string, string) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not start listener: %s", err)
	}
	t.Cleanup(func() { listener.Close() })
	addr, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("could not split listener address: %s", err)
	}
	return listener, addr, port
}

// The core behavior: pq asks for the configured hostname, and the dial goes to
// the pinned address instead, keeping the port pq derived from the connection
// parameters. The hostname here doesn't resolve at all (like a resolver outage),
// which also proves an established pin survives failed lookups.
func TestPinnedDialerDialsPin(t *testing.T) {
	_, addr, port := listen(t)

	resolved := &state.ResolvedHost{}
	resolved.Pin("db.invalid", addr)
	d := pinnedDialer{resolved: resolved, hostname: "db.invalid", logger: &util.Logger{}}

	conn, err := d.DialContext(context.Background(), "tcp", net.JoinHostPort("db.invalid", port))
	if err != nil {
		t.Fatalf("expected the dial to be redirected to the pinned address, got: %s", err)
	}
	conn.Close()
}

// A pinned address that stopped accepting connections costs one failed attempt
// within the dial, not a failed connection: the fallbacks from the latest lookup
// are tried next, and the pin moves to the address that actually connected.
func TestPinnedDialerFallsBackAndRepins(t *testing.T) {
	_, addr, port := listen(t)

	resolved := &state.ResolvedHost{}
	// 192.0.2.0/24 is reserved for documentation and is not routable, so this
	// stands in for an instance that has gone away
	resolved.Pin("localhost", "192.0.2.1")
	d := pinnedDialer{resolved: resolved, hostname: "localhost", logger: &util.Logger{}}

	// The deadline is divided among the attempts, so the unroutable address can
	// only burn its share of it before the fallbacks get their chance
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort("localhost", port))
	if err != nil {
		t.Fatalf("expected the dial to fall back to a resolved address, got: %s", err)
	}
	conn.Close()

	if resolved.Current() != addr {
		t.Errorf("expected the pin to move to the address that connected\nexpected %s\nactual %s\n", addr, resolved.Current())
	}
}

var addressRelatedErrorTests = []struct {
	name     string
	err      error
	expected bool
}{
	// Dial failures and timeouts are the pin's business
	{"net.OpError", &net.OpError{Op: "dial", Err: errors.New("connection refused")}, true},
	{"context.DeadlineExceeded", context.DeadlineExceeded, true},
	{"wrapped net error", &net.DNSError{Err: "no such host", Name: "db.example.com"}, true},
	// The server answered, so the address was fine
	{"pq auth error", errors.New("pq: password authentication failed for user \"pganalyze\""), false},
	{"missing database", errors.New("pq: database \"nope\" does not exist"), false},
	{"context.Canceled", context.Canceled, false},
}

func TestAddressRelatedError(t *testing.T) {
	for _, test := range addressRelatedErrorTests {
		actual := addressRelatedError(test.err)
		if actual != test.expected {
			t.Errorf("%s: addressRelatedError(%v)\nexpected %t\nactual %t\n", test.name, test.err, test.expected, actual)
		}
	}
}

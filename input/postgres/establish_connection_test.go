package postgres_test

import (
	"context"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/input/postgres"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

// setupPinTest - Builds a server pointed at TEST_DATABASE_URL over TCP, which is
// what the address pinning applies to. A URL without a hostname (a Unix socket)
// has nothing to pin, so there would be nothing to assert.
func setupPinTest(t *testing.T) (*state.Server, state.CollectionOpts, string) {
	testDatabaseUrl := os.Getenv("TEST_DATABASE_URL")
	if testDatabaseUrl == "" {
		t.Skipf("Skipping test requiring database connection since TEST_DATABASE_URL is not set")
	}
	u, err := url.Parse(testDatabaseUrl)
	if err != nil {
		t.Fatalf("Could not parse TEST_DATABASE_URL: %s", err)
	}
	hostname := u.Hostname()
	if hostname == "" {
		t.Skipf("Skipping test since TEST_DATABASE_URL does not connect over TCP")
	}

	server := state.MakeServer(config.ServerConfig{
		DbURL:                   testDatabaseUrl,
		MaxCollectorConnections: 10,
	}, false)

	return server, state.CollectionOpts{CollectorApplicationName: "pganalyze_test"}, hostname
}

// Every snapshot type opens its own connection, and they must all reach the same
// instance so that the cumulative counters they diff remain comparable.
func TestEstablishConnectionSharesResolvedAddress(t *testing.T) {
	server, opts, _ := setupPinTest(t)
	ctx := context.Background()
	logger := &util.Logger{}

	first, err := postgres.EstablishConnection(ctx, server, logger, opts, "")
	if err != nil {
		t.Fatalf("Could not connect to test database: %s", err)
	}
	defer first.Close()

	pinned := server.ResolvedHost.Current()
	if pinned == "" {
		t.Fatalf("expected the resolved address to be recorded on the server")
	}

	second, err := postgres.EstablishConnection(ctx, server, logger, opts, "")
	if err != nil {
		t.Fatalf("Could not open a second connection: %s", err)
	}
	defer second.Close()

	if server.ResolvedHost.Current() != pinned {
		t.Errorf("expected the second connection to reuse the pinned address\nexpected %s\nactual %s\n", pinned, server.ResolvedHost.Current())
	}

	// Both connections went to the same instance, so their identities agree and a
	// diff between statistics collected over them is valid
	firstIdentity, err := postgres.GetInstanceIdentity(ctx, first)
	if err != nil {
		t.Fatalf("Could not determine instance identity: %s", err)
	}
	if firstIdentity.IsZero() {
		t.Errorf("expected a non-zero instance identity")
	}
	if firstIdentity.ServerAddr == "" {
		t.Errorf("expected inet_server_addr() to be available over a TCP connection")
	}

	secondIdentity, err := postgres.GetInstanceIdentity(ctx, second)
	if err != nil {
		t.Fatalf("Could not determine instance identity for the second connection: %s", err)
	}
	if !firstIdentity.Matches(secondIdentity) {
		t.Errorf("expected both connections to report the same instance\nfirst %+v\nsecond %+v\n", firstIdentity, secondIdentity)
	}
}

// A pinned instance that went away costs one failed dial attempt, not a failed
// snapshot: the dialer falls back to a freshly resolved address within the same
// connection attempt and moves the pin to what actually connected.
func TestEstablishConnectionRecoversFromStalePin(t *testing.T) {
	server, opts, hostname := setupPinTest(t)
	// The deadline is what bounds the attempt against the unroutable address
	// before the dialer moves on to the fallbacks
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	logger := &util.Logger{}

	// 192.0.2.0/24 is reserved for documentation and is not routable, so this
	// stands in for an instance that has gone away
	server.ResolvedHost.Pin(hostname, "192.0.2.1")

	connection, err := postgres.EstablishConnection(ctx, server, logger, opts, "")
	if err != nil {
		t.Fatalf("expected the dialer to fall back to a freshly resolved address, got: %s", err)
	}
	defer connection.Close()

	pinned := server.ResolvedHost.Current()
	if pinned == "" || pinned == "192.0.2.1" {
		t.Errorf("expected the pin to move to the address that actually connected, have %q", pinned)
	}
}

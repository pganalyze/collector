package postgres

import (
	"context"
	"net"
	"strings"
	"time"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

// pinnedDialer - Dials the address this server's connections are pinned to
// instead of resolving the hostname pq was configured with.
//
// pq derives the address to dial and the name used for TLS verification from the
// same "host" connection parameter, but it takes the address through the dialer
// while reading the name directly out of the parameters (see tlsConf.ServerName
// in lib/pq's ssl.go). Replacing the address here therefore leaves sslmode
// verify-full, SNI and certificate hostname checks working against the
// configured hostname.
//
// The shared ResolvedHost is consulted at dial time rather than capturing an
// address when the pool is opened, so long-lived pools reconnect through the
// current pin instead of freezing whatever was pinned when they were created.
// The address that actually connects is reported back, which is what makes the
// pin "the instance we talk to" rather than "the first address of a lookup".
type pinnedDialer struct {
	dialer   net.Dialer
	resolved *state.ResolvedHost
	// The hostname from the connection parameters, which the pin applies to
	hostname string
	logger   *util.Logger
}

// fallbackDialTimeout - Bounds a dial attempt that still has fallback addresses
// behind it when there is no connection deadline (from connect_timeout or the
// context) to divide up, so that a blackholed pinned address cannot starve the
// fallbacks. The last address gets all remaining time, like a dial without
// pinning would.
const fallbackDialTimeout = 15 * time.Second

func (d pinnedDialer) Dial(network, address string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, address)
}

func (d pinnedDialer) DialTimeout(network, address string, timeout time.Duration) (net.Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return d.DialContext(ctx, network, address)
}

func (d pinnedDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	// pq builds a socket path rather than a host:port pair for Unix sockets, and
	// should not happen here (see canPinHost), but pass it through untouched
	if network == "unix" {
		return d.dialer.DialContext(ctx, network, address)
	}
	_, port, err := net.SplitHostPort(address)
	if err != nil {
		// Should not happen, but we would rather dial what pq asked for than
		// produce a malformed address
		return d.dialer.DialContext(ctx, network, address)
	}

	addrs, err := d.resolved.Get(ctx, d.hostname)
	if err != nil {
		// Fall back to letting the network stack resolve the hostname itself, so
		// that a problem in our own lookup can't stop collection outright
		d.logger.PrintVerbose("Could not resolve %s, leaving address resolution to the network: %s", d.hostname, err)
		return d.dialer.DialContext(ctx, network, address)
	}

	// Try the pinned address first, then the fallbacks from the latest lookup, so
	// a dead pinned instance costs one failed attempt inside this dial rather
	// than a failed snapshot
	var firstErr error
	for i, addr := range addrs {
		conn, err := d.dialAttempt(ctx, network, net.JoinHostPort(addr, port), len(addrs)-i)
		if err == nil {
			if d.resolved.Confirm(addr) {
				d.logger.PrintVerbose("Pinned address %s for host %s", addr, d.hostname)
			}
			return conn, nil
		}
		if firstErr == nil {
			firstErr = err
		}
		if ctx.Err() != nil {
			break
		}
	}
	return nil, firstErr
}

// dialAttempt - Dials one address. An attempt with more addresses behind it only
// gets a share of the remaining time (mirroring net.Dialer's partial deadlines),
// so that an address that swallows packets can't use up the whole connection
// deadline before the fallbacks get their chance.
func (d pinnedDialer) dialAttempt(ctx context.Context, network, address string, remaining int) (net.Conn, error) {
	if remaining > 1 {
		timeout := fallbackDialTimeout
		if deadline, ok := ctx.Deadline(); ok {
			timeout = time.Until(deadline) / time.Duration(remaining)
		}
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	return d.dialer.DialContext(ctx, network, address)
}

// canPinHost - Returns whether pinning a resolved address makes sense for this
// host. There is nothing to pin for Unix sockets or hosts that are already
// written as an IP address.
func canPinHost(host string) bool {
	if host == "" || strings.HasPrefix(host, "/") {
		return false
	}
	return net.ParseIP(host) == nil
}

package postgres

import (
	"context"
	"net"
	"strings"
	"time"
)

// pinnedDialer - Dials a fixed address instead of resolving the hostname pq was
// configured with.
//
// pq derives the address to dial and the name used for TLS verification from the
// same "host" connection parameter, but it takes the address through the dialer
// while reading the name directly out of the parameters (see tlsConf.ServerName
// in lib/pq's ssl.go). Replacing the address here therefore leaves sslmode
// verify-full, SNI and certificate hostname checks working against the
// configured hostname.
type pinnedDialer struct {
	dialer net.Dialer
	// The address to connect to instead of whatever the hostname resolves to
	addr string
}

// redirect - Rewrites the host portion of the address pq asked for, keeping the
// port it derived from the connection parameters. Unix socket paths are passed
// through untouched.
func (d pinnedDialer) redirect(network, address string) string {
	if network == "unix" {
		return address
	}
	_, port, err := net.SplitHostPort(address)
	if err != nil {
		return address
	}
	return net.JoinHostPort(d.addr, port)
}

func (d pinnedDialer) Dial(network, address string) (net.Conn, error) {
	return d.dialer.Dial(network, d.redirect(network, address))
}

func (d pinnedDialer) DialTimeout(network, address string, timeout time.Duration) (net.Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return d.DialContext(ctx, network, address)
}

func (d pinnedDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return d.dialer.DialContext(ctx, network, d.redirect(network, address))
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

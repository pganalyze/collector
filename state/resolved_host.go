package state

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

// ResolvedHostTTL - How long a resolved address is reused before we look the
// hostname up again.
//
// This only needs to be a backstop. The pin is dropped as soon as it stops
// working (see Invalidate callers), so the TTL exists for the case where the
// pinned instance still accepts connections but the hostname now points
// somewhere else - after an Aurora failover, the old writer keeps serving
// connections in its new reader role. Several full snapshot intervals is long
// enough that a pin normally spans many statistics cycles, and short enough
// that we notice such a change on our own.
const ResolvedHostTTL = 30 * time.Minute

// ResolvedHost - Caches the address a server's hostname resolved to, so that
// every connection the collector makes to one server reaches the same instance.
//
// The collector opens a separate connection for each kind of snapshot (full,
// query statistics, activity, logs) and closes it again afterwards, which means
// each one resolves the hostname independently. For a hostname that maps to a
// single instance that is fine, but an Aurora reader endpoint round-robins
// across readers and follows failovers, so consecutive connections can land on
// different instances. Since cumulative statistics are per-instance, we want
// separate connections - they run expensive queries and shouldn't queue behind
// each other - but a shared view of where those connections go.
//
// A stale pin cannot corrupt statistics on its own: the reference point for
// every diff is checked against PostgresInstanceIdentity, so an instance change
// costs one interval of data rather than producing a bogus spike.
type ResolvedHost struct {
	mutex    sync.Mutex
	hostname string
	addr     string
	pinnedAt time.Time
}

// Get - Returns the address to dial for the given hostname, resolving it if we
// have no usable pin yet.
//
// The returned address is only meant for dialing. Callers must keep passing the
// hostname to the Postgres driver so that TLS verification, SNI and anything
// else hostname-derived continue to use it.
func (r *ResolvedHost) Get(ctx context.Context, hostname string) (string, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.addr != "" && r.hostname == hostname && time.Since(r.pinnedAt) < ResolvedHostTTL {
		return r.addr, nil
	}

	addrs, err := net.DefaultResolver.LookupHost(ctx, hostname)
	if err != nil {
		return "", err
	}
	if len(addrs) == 0 {
		return "", fmt.Errorf("no addresses found for host %q", hostname)
	}

	r.hostname = hostname
	r.addr = addrs[0]
	r.pinnedAt = time.Now()

	return r.addr, nil
}

// Pin - Records the address to use for a hostname without looking it up, as a
// subsequent Get for that same hostname would have. Used by tests to set up a
// specific pin.
func (r *ResolvedHost) Pin(hostname string, addr string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.hostname = hostname
	r.addr = addr
	r.pinnedAt = time.Now()
}

// Invalidate - Drops the current pin, so the next Get resolves the hostname
// again. Called when connecting to the pinned address fails, since the instance
// it pointed at may be gone.
func (r *ResolvedHost) Invalidate() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.hostname = ""
	r.addr = ""
	r.pinnedAt = time.Time{}
}

// Current - Returns the currently pinned address, or an empty string if there is
// none. For logging and tests.
func (r *ResolvedHost) Current() string {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return r.addr
}

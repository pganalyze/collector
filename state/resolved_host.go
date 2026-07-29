package state

import (
	"context"
	"fmt"
	"net"
	"slices"
	"sync"
	"time"
)

// resolvedHostRecheckInterval - How often the hostname is looked up again while a
// pin is in place, to check that it still points at the pinned address.
//
// Lookups only hit the local resolver, but once a minute is enough: it matches the
// fastest snapshot cadence, and it means the burst of per-database schema
// connections within one full snapshot shares a single answer instead of each
// taking its own draw from a rotating endpoint.
const resolvedHostRecheckInterval = time.Minute

// resolvedHostEvictAfterMisses - How many consecutive lookups may omit the pinned
// address before the pin is dropped.
//
// This is the backstop for a pinned instance that still accepts connections but is
// no longer what the hostname points at - after an Aurora failover, the old writer
// keeps serving connections in its new reader role. Endpoints that return a stable
// answer (a writer endpoint, a reader endpoint with a single reader) contradict a
// stale pin on every lookup, so a moved hostname is noticed within about
// resolvedHostEvictAfterMisses * resolvedHostRecheckInterval. Requiring several
// consecutive misses matters for reader endpoints with multiple readers, which
// rotate a single record: a healthy pin is missing from any one answer, and
// evicting it would needlessly move our connections to a different instance.
const resolvedHostEvictAfterMisses = 10

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
// The pinned address is the one a connection last actually succeeded to (see
// Confirm), not merely the first address of a lookup. It is kept as long as the
// hostname keeps resolving to it, dropped when connecting through it fails (see
// Invalidate callers), and evicted when enough consecutive lookups stop including
// it. A resolver outage leaves an established pin untouched. The rest of the most
// recent answer is kept as dial fallbacks, so a dead pinned instance costs one
// failed dial attempt rather than a failed snapshot.
//
// A stale pin cannot corrupt statistics on its own: the reference point for
// every diff is checked against PostgresInstanceIdentity, so an instance change
// costs one interval of data rather than producing a bogus spike.
type ResolvedHost struct {
	mutex    sync.Mutex
	hostname string
	// The address a connection last succeeded to, empty if none
	addr string
	// The most recent lookup answer, used for dial fallbacks and to judge whether
	// the hostname still points at the pinned address
	lastAnswer   []string
	lastLookupAt time.Time
	// Consecutive lookups whose answer did not include the pinned address
	lookupMisses int
}

// Get - Returns the addresses to dial for the given hostname, in order: the
// pinned address first, then the rest of the most recent lookup answer as
// fallbacks.
//
// The returned addresses are only meant for dialing. Callers must keep passing
// the hostname to the Postgres driver so that TLS verification, SNI and anything
// else hostname-derived continue to use it.
//
// The mutex is deliberately held across the lookup: concurrent connections that
// arrive while no answer is cached should share one answer rather than each
// taking their own draw from a rotating endpoint.
func (r *ResolvedHost) Get(ctx context.Context, hostname string) ([]string, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.hostname != hostname {
		r.resetLocked()
		r.hostname = hostname
	}

	var lookupErr error
	if time.Since(r.lastLookupAt) >= resolvedHostRecheckInterval {
		lookupErr = r.lookupLocked(ctx)
	}

	addrs := r.dialOrderLocked()
	if len(addrs) == 0 {
		if lookupErr != nil {
			return nil, lookupErr
		}
		return nil, fmt.Errorf("no addresses found for host %q", hostname)
	}
	return addrs, nil
}

// lookupLocked - Resolves the hostname and folds the answer into the pin's
// evidence: an answer that includes the pinned address re-confirms it, and enough
// consecutive answers without it evict it, since the hostname has moved on even
// if the pinned instance still accepts our connections. A failed lookup leaves
// the pin and the cached answer untouched, so a resolver outage cannot take down
// a working pin.
func (r *ResolvedHost) lookupLocked(ctx context.Context) error {
	// Advance the clock even on failure, so a broken resolver delays at most one
	// connection per recheck interval
	r.lastLookupAt = time.Now()

	addrs, err := net.DefaultResolver.LookupHost(ctx, r.hostname)
	if err != nil {
		return err
	}
	if len(addrs) == 0 {
		return fmt.Errorf("no addresses found for host %q", r.hostname)
	}

	r.lastAnswer = addrs

	if r.addr == "" {
		return nil
	}
	if slices.Contains(addrs, r.addr) {
		r.lookupMisses = 0
	} else {
		r.lookupMisses++
		if r.lookupMisses >= resolvedHostEvictAfterMisses {
			r.addr = ""
			r.lookupMisses = 0
		}
	}
	return nil
}

// dialOrderLocked - The pinned address first, then the rest of the most recent
// answer as fallbacks.
func (r *ResolvedHost) dialOrderLocked() []string {
	addrs := make([]string, 0, len(r.lastAnswer)+1)
	if r.addr != "" {
		addrs = append(addrs, r.addr)
	}
	for _, addr := range r.lastAnswer {
		if addr != r.addr {
			addrs = append(addrs, addr)
		}
	}
	return addrs
}

// Confirm - Records the address a connection actually succeeded to, making it the
// pin that later connections dial first. Returns whether this changed the pin.
//
// Pinning what connected rather than what resolved first matters when a host
// resolves to addresses we cannot all reach - with IPv6 ordered first on an
// IPv4-only network, pinning the lookup result would wedge every connection on an
// unreachable address.
func (r *ResolvedHost) Confirm(addr string) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.addr == addr {
		return false
	}
	r.addr = addr
	r.lookupMisses = 0
	return true
}

// Pin - Records the address to use for a hostname without connecting, as a
// subsequent Confirm for that same hostname would have. Used by tests to set up a
// specific pin.
func (r *ResolvedHost) Pin(hostname string, addr string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.resetLocked()
	r.hostname = hostname
	r.addr = addr
}

// Invalidate - Drops the current pin and cached answer, so the next Get resolves
// the hostname again. Called when connecting fails with a network-level error,
// since the instance the pin pointed at may be gone.
func (r *ResolvedHost) Invalidate() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.resetLocked()
}

func (r *ResolvedHost) resetLocked() {
	r.hostname = ""
	r.addr = ""
	r.lastAnswer = nil
	r.lastLookupAt = time.Time{}
	r.lookupMisses = 0
}

// Current - Returns the currently pinned address, or an empty string if there is
// none. For logging and tests.
func (r *ResolvedHost) Current() string {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return r.addr
}

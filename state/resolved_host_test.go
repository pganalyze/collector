package state

import (
	"context"
	"slices"
	"testing"
	"time"
)

// forceRecheck - Ages the last lookup past the recheck interval, so the next Get
// resolves again. Standing in for the passage of time in tests.
func forceRecheck(r *ResolvedHost) {
	r.mutex.Lock()
	r.lastLookupAt = time.Now().Add(-resolvedHostRecheckInterval - time.Second)
	r.mutex.Unlock()
}

func TestResolvedHostSharesAnswerBetweenConnections(t *testing.T) {
	ctx := context.Background()
	r := &ResolvedHost{}

	first, err := r.Get(ctx, "localhost")
	if err != nil {
		t.Fatalf("could not resolve localhost: %s", err)
	}
	if len(first) == 0 {
		t.Fatalf("expected at least one resolved address")
	}

	// Poison the cached answer with an address that could not have come from a
	// lookup, to prove the second call reuses it rather than resolving again. This
	// is the whole purpose of the type: the full, query statistics and activity
	// snapshots each open their own connection and must all reach the same
	// instance, so within the recheck interval they must share one answer.
	r.mutex.Lock()
	r.lastAnswer = []string{"192.0.2.1"}
	r.mutex.Unlock()

	second, err := r.Get(ctx, "localhost")
	if err != nil {
		t.Fatalf("unexpected error on second lookup: %s", err)
	}
	if len(second) != 1 || second[0] != "192.0.2.1" {
		t.Errorf("expected the cached answer to be reused\nexpected [192.0.2.1]\nactual %v\n", second)
	}
}

func TestResolvedHostConfirmPinsFirst(t *testing.T) {
	ctx := context.Background()
	r := &ResolvedHost{}

	if _, err := r.Get(ctx, "localhost"); err != nil {
		t.Fatalf("could not resolve localhost: %s", err)
	}

	// A connection succeeded to this address, so later connections dial it first
	if changed := r.Confirm("192.0.2.1"); !changed {
		t.Errorf("expected the first Confirm to change the pin")
	}
	if changed := r.Confirm("192.0.2.1"); changed {
		t.Errorf("expected confirming the same address again to change nothing")
	}

	addrs, err := r.Get(ctx, "localhost")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if addrs[0] != "192.0.2.1" {
		t.Errorf("expected the pinned address to be dialed first\nexpected 192.0.2.1\nactual %s\n", addrs[0])
	}
}

func TestResolvedHostKeepsPinWhileAnswerContainsIt(t *testing.T) {
	ctx := context.Background()
	r := &ResolvedHost{}

	first, err := r.Get(ctx, "localhost")
	if err != nil {
		t.Fatalf("could not resolve localhost: %s", err)
	}
	r.Confirm(first[0])

	// The hostname keeps resolving to the pinned address, so no amount of
	// rechecking may evict it - unlike a fixed TTL, which would re-roll a
	// perfectly healthy pin
	for i := 0; i < 2*resolvedHostEvictAfterMisses; i++ {
		forceRecheck(r)
		if _, err := r.Get(ctx, "localhost"); err != nil {
			t.Fatalf("unexpected error on recheck %d: %s", i, err)
		}
	}
	if r.Current() != first[0] {
		t.Errorf("expected a re-confirmed pin to stay\nexpected %s\nactual %s\n", first[0], r.Current())
	}
}

func TestResolvedHostEvictsPinAfterConsecutiveMisses(t *testing.T) {
	ctx := context.Background()
	r := &ResolvedHost{}

	if _, err := r.Get(ctx, "localhost"); err != nil {
		t.Fatalf("could not resolve localhost: %s", err)
	}
	// An address the lookups will never return again: the hostname has moved on,
	// even though the instance behind the pin may still accept connections (after
	// an Aurora failover, the demoted writer keeps serving in its reader role)
	r.Confirm("192.0.2.1")

	for i := 0; i < resolvedHostEvictAfterMisses; i++ {
		if r.Current() != "192.0.2.1" {
			t.Fatalf("expected the pin to survive %d misses, gone after %d", resolvedHostEvictAfterMisses, i)
		}
		forceRecheck(r)
		if _, err := r.Get(ctx, "localhost"); err != nil {
			t.Fatalf("unexpected error on recheck %d: %s", i, err)
		}
	}

	if r.Current() != "" {
		t.Errorf("expected the pin to be evicted after %d consecutive misses, still have %s", resolvedHostEvictAfterMisses, r.Current())
	}
	addrs, err := r.Get(ctx, "localhost")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if slices.Contains(addrs, "192.0.2.1") {
		t.Errorf("expected only freshly resolved addresses after eviction, got %v", addrs)
	}
}

func TestResolvedHostKeepsPinThroughResolverFailure(t *testing.T) {
	ctx := context.Background()
	r := &ResolvedHost{}

	// .invalid never resolves (RFC 6761), standing in for a resolver outage. A
	// working pin must survive it: the lookups are only evidence gathering, and
	// no evidence is not evidence of a change.
	r.Pin("db.invalid", "192.0.2.1")

	addrs, err := r.Get(ctx, "db.invalid")
	if err != nil {
		t.Fatalf("expected the pin to be served despite the failed lookup, got: %s", err)
	}
	if len(addrs) != 1 || addrs[0] != "192.0.2.1" {
		t.Errorf("expected the pinned address\nexpected [192.0.2.1]\nactual %v\n", addrs)
	}
	if r.Current() != "192.0.2.1" {
		t.Errorf("expected the failed lookup to leave the pin alone, have %q", r.Current())
	}
}

func TestResolvedHostErrorsWithNothingToServe(t *testing.T) {
	ctx := context.Background()
	r := &ResolvedHost{}

	// No pin and no cached answer: the lookup failure is all we have
	if _, err := r.Get(ctx, "db.invalid"); err == nil {
		t.Errorf("expected an error resolving an invalid hostname with no pin to fall back on")
	}
}

func TestResolvedHostInvalidate(t *testing.T) {
	ctx := context.Background()
	r := &ResolvedHost{}

	if _, err := r.Get(ctx, "localhost"); err != nil {
		t.Fatalf("could not resolve localhost: %s", err)
	}
	r.Confirm("192.0.2.1")

	r.Invalidate()
	if r.Current() != "" {
		t.Errorf("expected no pinned address after Invalidate, got %s", r.Current())
	}

	// A failed connection drops the pin so we pick up a replacement instance
	addrs, err := r.Get(ctx, "localhost")
	if err != nil {
		t.Fatalf("could not resolve localhost after invalidating: %s", err)
	}
	if slices.Contains(addrs, "192.0.2.1") {
		t.Errorf("expected freshly resolved addresses after Invalidate, got the stale pin in %v", addrs)
	}
}

func TestResolvedHostReresolvesForDifferentHostname(t *testing.T) {
	ctx := context.Background()
	r := &ResolvedHost{}

	if _, err := r.Get(ctx, "localhost"); err != nil {
		t.Fatalf("could not resolve localhost: %s", err)
	}
	r.Confirm("192.0.2.1")

	// A pin only applies to the hostname it was resolved from, so that connecting
	// to a different host (e.g. after a config change) does not reuse it
	addrs, err := r.Get(ctx, "127.0.0.1")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(addrs) != 1 || addrs[0] != "127.0.0.1" {
		t.Errorf("expected a new lookup for a different hostname\nexpected [127.0.0.1]\nactual %v\n", addrs)
	}
	if r.Current() != "" {
		t.Errorf("expected the old hostname's pin to be dropped, have %q", r.Current())
	}
}

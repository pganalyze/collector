package state

import (
	"context"
	"testing"
	"time"
)

func TestResolvedHostReusesPin(t *testing.T) {
	ctx := context.Background()
	r := &ResolvedHost{}

	first, err := r.Get(ctx, "localhost")
	if err != nil {
		t.Fatalf("could not resolve localhost: %s", err)
	}
	if first == "" {
		t.Fatalf("expected a resolved address")
	}

	// Point the pin at an address that could not have come from a lookup, to prove
	// the second call reuses it rather than resolving again. This is the whole
	// purpose of the type: the full, query statistics and activity snapshots each
	// open their own connection and must all reach the same instance.
	r.mutex.Lock()
	r.addr = "192.0.2.1"
	r.mutex.Unlock()

	second, err := r.Get(ctx, "localhost")
	if err != nil {
		t.Fatalf("unexpected error on second lookup: %s", err)
	}
	if second != "192.0.2.1" {
		t.Errorf("expected the pinned address to be reused\nexpected 192.0.2.1\nactual %s\n", second)
	}
}

func TestResolvedHostInvalidate(t *testing.T) {
	ctx := context.Background()
	r := &ResolvedHost{}

	if _, err := r.Get(ctx, "localhost"); err != nil {
		t.Fatalf("could not resolve localhost: %s", err)
	}

	r.mutex.Lock()
	r.addr = "192.0.2.1"
	r.mutex.Unlock()

	r.Invalidate()
	if r.Current() != "" {
		t.Errorf("expected no pinned address after Invalidate, got %s", r.Current())
	}

	// A failed connection drops the pin so we pick up a replacement instance
	third, err := r.Get(ctx, "localhost")
	if err != nil {
		t.Fatalf("could not resolve localhost after invalidating: %s", err)
	}
	if third == "192.0.2.1" {
		t.Errorf("expected a freshly resolved address after Invalidate, got the stale pin")
	}
}

func TestResolvedHostReresolvesForDifferentHostname(t *testing.T) {
	ctx := context.Background()
	r := &ResolvedHost{}

	if _, err := r.Get(ctx, "localhost"); err != nil {
		t.Fatalf("could not resolve localhost: %s", err)
	}

	r.mutex.Lock()
	r.addr = "192.0.2.1"
	r.mutex.Unlock()

	// A pin only applies to the hostname it was resolved from, so that connecting
	// to a different host (e.g. after a config change) does not reuse it
	addr, err := r.Get(ctx, "127.0.0.1")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if addr != "127.0.0.1" {
		t.Errorf("expected a new lookup for a different hostname\nexpected 127.0.0.1\nactual %s\n", addr)
	}
}

func TestResolvedHostExpiresPin(t *testing.T) {
	ctx := context.Background()
	r := &ResolvedHost{}

	if _, err := r.Get(ctx, "localhost"); err != nil {
		t.Fatalf("could not resolve localhost: %s", err)
	}

	// Age the pin past its TTL. The TTL is the backstop for a pinned instance that
	// still accepts connections but is no longer what the hostname points at.
	r.mutex.Lock()
	r.addr = "192.0.2.1"
	r.pinnedAt = time.Now().Add(-ResolvedHostTTL - time.Minute)
	r.mutex.Unlock()

	addr, err := r.Get(ctx, "localhost")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if addr == "192.0.2.1" {
		t.Errorf("expected the pin to be resolved again once past its TTL")
	}
}

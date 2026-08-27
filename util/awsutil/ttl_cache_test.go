package awsutil

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestTtlCacheCachesValues(t *testing.T) {
	var cache ttlCache[string]
	var fetches int32

	for i := 0; i < 3; i++ {
		value, err := cache.get(context.Background(), "key", func(ctx context.Context) (string, error) {
			atomic.AddInt32(&fetches, 1)
			return "value", nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if value != "value" {
			t.Fatalf("unexpected value: %s", value)
		}
	}

	if fetches != 1 {
		t.Errorf("expected 1 fetch, got %d", fetches)
	}
}

func TestTtlCacheCachesErrors(t *testing.T) {
	var cache ttlCache[string]
	var fetches int32
	fetchErr := errors.New("fetch failed")

	for i := 0; i < 3; i++ {
		_, err := cache.get(context.Background(), "key", func(ctx context.Context) (string, error) {
			atomic.AddInt32(&fetches, 1)
			return "", fetchErr
		})
		if err != fetchErr {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if fetches != 1 {
		t.Errorf("expected 1 fetch (errors are cached), got %d", fetches)
	}
}

func TestTtlCacheExpires(t *testing.T) {
	var cache ttlCache[string]
	var fetches int32
	fetch := func(ctx context.Context) (string, error) {
		atomic.AddInt32(&fetches, 1)
		return "value", nil
	}

	cache.get(context.Background(), "key", fetch)

	// Simulate the entry having been fetched longer than the TTL ago
	cache.entries["key"].fetchedAt = time.Now().Add(-cacheTTL - time.Second)

	cache.get(context.Background(), "key", fetch)

	if fetches != 2 {
		t.Errorf("expected 2 fetches after expiry, got %d", fetches)
	}
}

func TestTtlCacheDeduplicatesConcurrentFetches(t *testing.T) {
	var cache ttlCache[string]
	var fetches int32
	fetchStarted := make(chan struct{})
	fetchFinish := make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		cache.get(context.Background(), "key", func(ctx context.Context) (string, error) {
			close(fetchStarted)
			<-fetchFinish
			atomic.AddInt32(&fetches, 1)
			return "value", nil
		})
	}()

	<-fetchStarted

	// These calls arrive while the first fetch is still in progress and must
	// wait for its result instead of fetching again
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			value, err := cache.get(context.Background(), "key", func(ctx context.Context) (string, error) {
				atomic.AddInt32(&fetches, 1)
				return "other", nil
			})
			if err != nil || value != "value" {
				t.Errorf("expected shared result \"value\", got %q (err: %v)", value, err)
			}
		}()
	}

	close(fetchFinish)
	wg.Wait()

	if fetches != 1 {
		t.Errorf("expected 1 fetch, got %d", fetches)
	}
}

func TestTtlCacheWaiterContextCancellation(t *testing.T) {
	var cache ttlCache[string]
	fetchStarted := make(chan struct{})
	fetchFinish := make(chan struct{})

	go cache.get(context.Background(), "key", func(ctx context.Context) (string, error) {
		close(fetchStarted)
		<-fetchFinish
		return "value", nil
	})

	<-fetchStarted

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := cache.get(ctx, "key", func(ctx context.Context) (string, error) {
		return "other", nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled waiting on in-flight fetch, got %v", err)
	}

	close(fetchFinish)
}

// The caller that starts a fetch may get canceled while the fetch is running
// (e.g. one server's collection run timing out, or the collector shutting
// down). This must not fail the fetch for the other servers sharing its
// result, and must not get cached as the fetch result.
func TestTtlCacheFetcherContextCancellation(t *testing.T) {
	var cache ttlCache[string]
	var fetches int32
	fetchStarted := make(chan struct{})
	fetchFinish := make(chan struct{})

	ctx, cancel := context.WithCancel(context.Background())
	go cache.get(ctx, "key", func(ctx context.Context) (string, error) {
		close(fetchStarted)
		<-fetchFinish
		atomic.AddInt32(&fetches, 1)
		if err := ctx.Err(); err != nil {
			return "", err
		}
		return "value", nil
	})

	<-fetchStarted
	cancel()
	close(fetchFinish)

	value, err := cache.get(context.Background(), "key", func(ctx context.Context) (string, error) {
		atomic.AddInt32(&fetches, 1)
		return "other", nil
	})
	if err != nil || value != "value" {
		t.Errorf("expected \"value\" despite fetching caller's cancellation, got %q (err: %v)", value, err)
	}
	if fetches != 1 {
		t.Errorf("expected 1 fetch, got %d", fetches)
	}
}

// An already-canceled caller must not start a new fetch (e.g. during
// shutdown), since it would outlive the caller and nobody may consume it
func TestTtlCacheCanceledCallerDoesNotFetch(t *testing.T) {
	var cache ttlCache[string]
	var fetches int32

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := cache.get(ctx, "key", func(ctx context.Context) (string, error) {
		atomic.AddInt32(&fetches, 1)
		return "value", nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
	if atomic.LoadInt32(&fetches) != 0 {
		t.Errorf("expected no fetch to be started, got %d", fetches)
	}
}

// A panicking fetch must not leave the entry permanently in progress (which
// would hang all future gets for the key), but instead turn into an error
func TestTtlCachePanickingFetch(t *testing.T) {
	var cache ttlCache[string]

	_, err := cache.get(context.Background(), "key", func(ctx context.Context) (string, error) {
		panic("fetch gone wrong")
	})
	if err == nil || !strings.Contains(err.Error(), "fetch gone wrong") {
		t.Errorf("expected panic to surface as error, got %v", err)
	}
}

func TestTtlCacheDistinctKeys(t *testing.T) {
	var cache ttlCache[string]
	var fetches int32
	fetch := func(ctx context.Context) (string, error) {
		atomic.AddInt32(&fetches, 1)
		return "value", nil
	}

	cache.get(context.Background(), "one", fetch)
	cache.get(context.Background(), "two", fetch)

	if fetches != 2 {
		t.Errorf("expected 2 fetches for distinct keys, got %d", fetches)
	}
}

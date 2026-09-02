package awsutil

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// How long cached results are reused before refetching. This bounds both how
// quickly we react to changes on the AWS side (e.g. failovers) and how often
// many servers in the same account hit the (account-wide) API rate limits.
const cacheTTL = 60 * time.Second

// Backstop timeout for a single shared fetch. This bounds how long a hung
// fetch can stall snapshots / log downloads for servers in the account.
const cacheFetchTimeout = 60 * time.Second

// ttlCache caches fetch results (including errors) for cacheTTL and
// deduplicates concurrent fetches of the same key, so many servers sharing an
// account do not repeat the same AWS API call
type ttlCache[T any] struct {
	mutex   sync.Mutex
	entries map[string]*ttlCacheEntry[T]
}

type ttlCacheEntry[T any] struct {
	done      chan struct{} // closed once value/err/fetchedAt are set
	value     T
	err       error
	fetchedAt time.Time
}

// get returns the cached value for key, fetching it if missing or expired.
// The fetch runs on its own goroutine, detached from the caller's context
// (with cacheFetchTimeout as a backstop): the result is shared across
// all servers in the account, so one server's cancellation or deadline must
// not fail - or worse, get cached for - the others. Each caller still honors
// its own context while waiting.
//
// Errors served from the cache (as opposed to those returned by the fetch
// itself) are marked as such, so log output makes clear that a fix of the
// underlying problem may not be reflected until the cache entry expires.
func (c *ttlCache[T]) get(ctx context.Context, key string, fetch func(context.Context) (T, error)) (T, error) {
	c.mutex.Lock()
	if c.entries == nil {
		c.entries = make(map[string]*ttlCacheEntry[T])
	}
	entry, ok := c.entries[key]
	if ok {
		select {
		case <-entry.done: // Completed fetch, check for expiry
			if time.Since(entry.fetchedAt) < cacheTTL {
				c.mutex.Unlock()
				if entry.err != nil {
					return entry.value, fmt.Errorf("%w (cached error, retries every %.0fs)", entry.err, cacheTTL.Seconds())
				}
				return entry.value, nil
			}
			// Expired, fall through and fetch again
		default: // Fetch in progress, wait for its result
			c.mutex.Unlock()
			return waitForEntry(ctx, entry)
		}
	}

	// Don't start a fetch on behalf of an already-canceled caller (e.g. during
	// shutdown), since it would outlive the caller and nobody may consume it
	if err := ctx.Err(); err != nil {
		c.mutex.Unlock()
		var zero T
		return zero, err
	}

	entry = &ttlCacheEntry[T]{done: make(chan struct{})}
	c.entries[key] = entry

	// Unlock now so that other callers that are also interested in the result
	// can start waiting on the done channel (synchronized by the mutex lock at
	// the start of the function)
	c.mutex.Unlock()

	go func() {
		defer func() {
			// Turn a panic in fetch into an error instead of crashing (or,
			// if we didn't close the done channel, hanging all waiters)
			if r := recover(); r != nil {
				entry.err = fmt.Errorf("panic during fetch of \"%s\": %v", key, r)
			}
			entry.fetchedAt = time.Now()
			close(entry.done)
		}()
		fetchCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cacheFetchTimeout)
		defer cancel()
		entry.value, entry.err = fetch(fetchCtx)
	}()

	return waitForEntry(ctx, entry)
}

func waitForEntry[T any](ctx context.Context, entry *ttlCacheEntry[T]) (T, error) {
	select {
	case <-entry.done:
		return entry.value, entry.err
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	}
}

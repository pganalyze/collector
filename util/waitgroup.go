package util

import (
	"sync"
	"time"
)

// WaitWithTimeout - Waits for the WaitGroup to finish, returning false if it did
// not finish within the given timeout
//
// The Goroutine waiting on the WaitGroup is left behind when the timeout hits, so
// this is only safe for callers that are on their way out.
func WaitWithTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	done := make(chan struct{})
	go func() {
		defer close(done)
		wg.Wait()
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-done:
		return true
	case <-timer.C:
		return false
	}
}

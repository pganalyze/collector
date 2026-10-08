package util_test

import (
	"sync"
	"testing"
	"time"

	"github.com/pganalyze/collector/util"
)

func TestWaitWithTimeoutFinishes(t *testing.T) {
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(10 * time.Millisecond)
	}()

	if !util.WaitWithTimeout(&wg, 30*time.Second) {
		t.Error("TestWaitWithTimeoutFinishes: expected waiting for a WaitGroup that finishes to succeed")
	}
}

// A collection Goroutine blocked on a Postgres read that never returns never
// marks itself done, which is what we must not wait for indefinitely
func TestWaitWithTimeoutGivesUp(t *testing.T) {
	var wg sync.WaitGroup
	stuck := make(chan struct{})
	defer close(stuck)

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-stuck
	}()

	if util.WaitWithTimeout(&wg, 10*time.Millisecond) {
		t.Error("TestWaitWithTimeoutGivesUp: expected waiting for a WaitGroup that never finishes to time out")
	}
}

package state

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestQueue_FIFOEviction(t *testing.T) {
	q := &Queue{data: make([]QueueItem, 5), capacity: 5}
	q.cond = sync.NewCond(&q.mu)
	ctx := context.Background()
	// First item should be evicted after pusing 6 items into 5-capacity queue
	for i := 0; i < 6; i++ {
		q.PushBytes(fmt.Sprintf("%d", i), []byte{byte(i)})
	}
	tx, err := q.Pop(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if tx.Kind != "1" || !bytes.Equal(tx.Snapshot, []byte{1}) {
		t.Errorf("expected item 1, got kind=%s snapshot=%v", tx.Kind, tx.Snapshot)
	}
	tx.Commit()
	// Eviction during in-flight transaction is a no-op; second item remains
	q2 := &Queue{data: make([]QueueItem, 2), capacity: 2}
	q2.cond = sync.NewCond(&q2.mu)
	q2.PushBytes("1", []byte("data"))
	tx2, _ := q2.Pop(ctx)
	q2.PushBytes("2", []byte("data")) // evicts tx1
	tx2.Commit()                      // safe no-op
	tx3, err := q2.Pop(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if tx3.Kind != "2" {
		t.Errorf("expected item 2 after eviction, got %s", tx3.Kind)
	}
	tx3.Commit()
}

func TestQueue_RollbackPreservesHead(t *testing.T) {
	q := NewQueue(nil)
	ctx := context.Background()
	q.PushBytes("1", []byte("data"))
	q.PushBytes("2", []byte("data"))
	tx, _ := q.Pop(ctx)
	tx.Rollback()
	tx2, err := q.Pop(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if tx2.Kind != "1" {
		t.Errorf("expected Kind 1 after rollback, got %v", tx2.Kind)
	}
	tx2.Commit()
}

func TestQueue_ConcurrentPushPopEviction(t *testing.T) {
	q := &Queue{data: make([]QueueItem, 3), capacity: 3}
	q.cond = sync.NewCond(&q.mu)
	var dropCount int64
	q.DropCallback = func(string, int64) { atomic.AddInt64(&dropCount, 1) }
	var wg sync.WaitGroup
	for p := 0; p < 8; p++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				q.PushBytes(fmt.Sprintf("p%d-i%d", id, i), []byte(fmt.Sprintf("data-%d-%d", id, i)))
			}
		}(p)
	}
	wg.Wait()
	q.Close()
	if dropCount == 0 {
		t.Error("expected drops under capacity pressure")
	}
}

func TestQueue_CloseUnblocksPop(t *testing.T) {
	q := NewQueue(nil)
	ctx := context.Background()
	var wg sync.WaitGroup
	var errCount int64
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := q.Pop(ctx)
			if err == nil {
				t.Error("expected error from closed queue")
			} else if err.Error() == "queue closed" {
				atomic.AddInt64(&errCount, 1)
			}
		}()
	}
	go q.Close()
	wg.Wait()
	if atomic.LoadInt64(&errCount) != 10 {
		t.Errorf("expected 10 'queue closed' errors, got %d", errCount)
	}
}

func TestQueue_ContextCancelUnblocksPop(t *testing.T) {
	q := NewQueue(nil)
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			q.Pop(ctx)
		}()
	}
	cancel()
	wg.Wait()
	q.Close()
}

func TestQueue_PushUnblockedByWaitingPop(t *testing.T) {
	q := NewQueue(nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var popDone sync.WaitGroup
	popDone.Add(1)
	go func() {
		defer popDone.Done()
		q.Pop(ctx)
	}()
	// Wait until Pop has entered cond.Wait() and released the mutex
	for i := 0; i < 100; i++ {
		q.mu.Lock()
		empty := q.size == 0
		q.mu.Unlock()
		if empty {
			break
		}
	}
	done := make(chan struct{})
	go func() {
		q.PushBytes("test", []byte("data"))
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Push blocked while Pop was waiting")
	}
	cancel()
	popDone.Wait()
}

func TestQueue_GlobalMemoryLimitEviction(t *testing.T) {
	original := QueueMemory
	defer func() { QueueMemory = original }()
	QueueMemory = NewMemoryLimit(200)
	q := NewQueue(nil)
	for i := 0; i < 100; i++ {
		q.PushBytes(fmt.Sprintf("item-%d", i), make([]byte, 10))
	}
	if q.size > 25 {
		t.Errorf("expected at most ~20 items, got %d", q.size)
	}
}

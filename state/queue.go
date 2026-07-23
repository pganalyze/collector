package state

import (
	"bytes"
	"compress/zlib"
	"context"
	"errors"
	"sync"

	"github.com/pganalyze/collector/util"
	"google.golang.org/protobuf/proto"
)

const DefaultCapacity = 500

// Queue of snapshots ready for submission to pganalyze. If snapshot
// submission fails, the original snapshot order is retained up until
// snapshots must be dropped to stay within the capacity and memory limits.
type Queue struct {
	mu           sync.Mutex
	cond         *sync.Cond
	data         []QueueItem
	capacity     int
	head         int
	tail         int
	size         int
	sizeBytes    int64
	inFlight     bool // true while an item is being uploaded (between Pop and Commit/Rollback)
	closed       bool
	Logger       *util.Logger
	DropCallback func(kind string, sizeBytes int64)
}

func NewQueue(logger *util.Logger) *Queue {
	q := &Queue{
		data:     make([]QueueItem, DefaultCapacity),
		capacity: DefaultCapacity,
		Logger:   logger,
	}
	q.cond = sync.NewCond(&q.mu)
	return q
}

// Push adds an item to the tail. If full or over the global memory limit,
// it drops the oldest items to make room, unless that item is currently
// being uploaded (in-flight). In that case, the new push is dropped instead.
func (q *Queue) Push(kind string, snapshot proto.Message) (err error) {
	data, err := proto.Marshal(snapshot)
	if err != nil {
		return
	}
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	w.Write(data)
	w.Close()
	return q.PushBytes(kind, buf.Bytes())
}

// Factored out from Push so it can be called by tests without building real snapshots
func (q *Queue) PushBytes(kind string, bytes []byte) (err error) {
	sizeBytes := int64(len(bytes))
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return
	}
	beforeSize := q.size
	q.makeSpace(sizeBytes)
	// If makeSpace couldn't evict (e.g., head is in-flight), drop the new item.
	if beforeSize == q.size && beforeSize >= q.capacity {
		QueueMemory.Remove(sizeBytes)
		q.logDrop(kind, sizeBytes)
		return
	}
	q.data[q.tail] = QueueItem{
		Kind:      kind,
		Snapshot:  bytes,
		SizeBytes: sizeBytes,
	}
	q.tail = (q.tail + 1) % q.capacity
	q.size++
	q.sizeBytes += sizeBytes
	q.cond.Signal()
	return
}

// Pop blocks until an item is ready or the queue closes.
// On success, it marks the head as in-flight and returns a Transaction handle.
func (q *Queue) Pop(ctx context.Context) (*Transaction, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for (q.size == 0 || q.inFlight) && !q.closed && ctx.Err() == nil {
		q.cond.Wait()
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if q.closed {
		return nil, errors.New("queue closed")
	}
	item := q.data[q.head]
	q.inFlight = true
	return &Transaction{
		Kind:      item.Kind,
		Snapshot:  item.Snapshot,
		SizeBytes: item.SizeBytes,
		q:         q,
	}, nil
}

// Close terminates the queue, unblocks waiting readers, and rejects new writes
func (q *Queue) Close() {
	q.mu.Lock()
	if !q.closed {
		q.closed = true
		q.cond.Broadcast()
	}
	q.mu.Unlock()
}

type QueueItem struct {
	Kind      string
	Snapshot  []byte
	SizeBytes int64
}

type Transaction struct {
	Kind      string
	Snapshot  []byte
	SizeBytes int64
	q         *Queue
}

// Commit advances past the head item, marking it as successfully uploaded.
func (t *Transaction) Commit() {
	t.q.mu.Lock()
	defer t.q.mu.Unlock()
	if !t.q.inFlight {
		return
	}
	t.q.data[t.q.head] = QueueItem{}
	t.q.head = (t.q.head + 1) % t.q.capacity
	t.q.size--
	t.q.sizeBytes -= t.SizeBytes
	QueueMemory.Remove(t.SizeBytes)
	t.q.inFlight = false
	t.q.cond.Broadcast()
}

// Rollback leaves the head item in place so it will be retried on the next Pop.
func (t *Transaction) Rollback() {
	t.q.mu.Lock()
	defer t.q.mu.Unlock()
	if !t.q.inFlight {
		return
	}
	t.q.inFlight = false
	t.q.cond.Broadcast()
}

// Does nothing if the head snapshot is in-flight to preserve ordering on retry.
func (q *Queue) makeSpace(sizeBytes int64) {
	QueueMemory.Add(sizeBytes)
	// Evict items to satisfy the global memory limit.
	evictedCount := 0
	for evictedCount <= 100 {
		if !QueueMemory.OverLimit() {
			break
		}
		if q.size == 0 || q.inFlight {
			break
		}
		evicted := q.data[q.head]
		q.data[q.head] = QueueItem{}
		q.head = (q.head + 1) % q.capacity
		q.size--
		q.sizeBytes -= evicted.SizeBytes
		QueueMemory.Remove(evicted.SizeBytes)
		q.logDrop(evicted.Kind, evicted.SizeBytes)
		evictedCount++
	}
	// Handle capacity-based eviction.
	if q.size == q.capacity && !q.inFlight {
		evicted := q.data[q.head]
		q.data[q.head] = QueueItem{}
		q.head = (q.head + 1) % q.capacity
		q.size--
		q.sizeBytes -= evicted.SizeBytes
		QueueMemory.Remove(evicted.SizeBytes)
		q.logDrop(evicted.Kind, evicted.SizeBytes)
	}
}

func (q *Queue) logDrop(kind string, sizeBytes int64) {
	if q.Logger != nil {
		q.Logger.PrintWarning("Dropped %s snapshot (%d bytes) from queue", kind, sizeBytes)
	}
	if q.DropCallback != nil {
		q.DropCallback(kind, sizeBytes)
	}
}

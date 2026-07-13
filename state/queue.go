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

// Queue of snapshots ready for submission to pganalyze
//   - Thread safe: Safe for concurrent use by multiple readers and writers.
//   - Limited capacity: If full, Push drops the oldest item to make room.
//   - Transactional: Pop locks the head item via a generation ID which is removed on Commit, and re-released on Rollback.
//   - Generation tracking: Increments a counter on every Push to uniquely identify each item.
//     If a Push evicts an in-flight item, its generation changes, safely ignoring late Commits or Rollbacks.
//   - Memory-bounded: When the global QueueMemory limit is exceeded, Push drops the oldest items
//     to stay within the budget. All queues share this single global limit.
type Queue struct {
	mu           sync.Mutex
	cond         *sync.Cond
	data         []QueueItem
	capacity     int
	head         int
	tail         int
	size         int
	sizeBytes    int64
	activeGen    uint64
	closed       bool
	nextGenID    uint64
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

// Push adds an item to the tail. If full, it drops the oldest item to make room.
// When the global queue memory limit is exceeded, Push drops the oldest items
// (starting with the current queue's oldest) to stay within the budget.
// All queues share the single global memory limit defined by QueueMemory.
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

// Factored out from Push so it can be called by tests without constructing real snapshots
func (q *Queue) PushBytes(kind string, bytes []byte) (err error) {
	sizeBytes := int64(len(bytes))
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return
	}
	q.makeSpace(sizeBytes)
	q.data[q.tail] = QueueItem{
		Kind:       kind,
		Snapshot:   bytes,
		SizeBytes:  sizeBytes,
		Generation: q.nextGenID,
	}
	q.tail = (q.tail + 1) % q.capacity
	q.size++
	q.sizeBytes += sizeBytes
	q.cond.Signal()
	return
}

// Pop blocks until an item is ready or the queue closes.
// On success, it locks the head item and returns a Transaction handle.
//
// The caller is woken by Push (Signal), Commit/Rollback (Broadcast), or Close (Broadcast).
func (q *Queue) Pop(ctx context.Context) (*Transaction, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for (q.size == 0 || q.activeGen != 0) && !q.closed && ctx.Err() == nil {
		q.cond.Wait()
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if q.closed {
		return nil, errors.New("queue closed")
	}
	item := q.data[q.head]
	q.activeGen = item.Generation
	return &Transaction{
		Kind:       item.Kind,
		Snapshot:   item.Snapshot,
		SizeBytes:  item.SizeBytes,
		generation: item.Generation,
		q:          q,
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
	Kind       string
	Snapshot   []byte
	SizeBytes  int64
	Generation uint64
}

type Transaction struct {
	Kind       string
	Snapshot   []byte
	SizeBytes  int64
	generation uint64
	q          *Queue
}

func (t *Transaction) Commit() {
	t.q.mu.Lock()
	defer t.q.mu.Unlock()
	// Validate that the transaction hasn't been evicted or superseded
	if t.q.activeGen != t.generation || t.q.data[t.q.head].Generation != t.generation {
		return
	}
	t.q.data[t.q.head] = QueueItem{}
	t.q.head = (t.q.head + 1) % t.q.capacity
	t.q.size--
	t.q.sizeBytes -= t.SizeBytes
	QueueMemory.Remove(t.SizeBytes)
	t.q.activeGen = 0
	t.q.cond.Broadcast()
}

func (t *Transaction) Rollback() {
	t.q.mu.Lock()
	defer t.q.mu.Unlock()
	// Validate that the transaction hasn't been evicted or superseded
	if t.q.activeGen != t.generation || t.q.data[t.q.head].Generation != t.generation {
		return
	}
	t.q.activeGen = 0
	t.q.cond.Broadcast()
}

func (q *Queue) makeSpace(sizeBytes int64) {
	QueueMemory.Add(sizeBytes)
	// Evict items to satisfy the global memory limit
	evictedCount := 0
	for evictedCount <= 100 {
		if !QueueMemory.OverLimit() {
			break
		}
		if q.size == 0 {
			// This queue is empty; another server's queue is the problem
			break
		}
		evicted := q.data[q.head]
		if q.activeGen == evicted.Generation {
			q.activeGen = 0
		}
		q.data[q.head] = QueueItem{}
		q.head = (q.head + 1) % q.capacity
		q.size--
		q.sizeBytes -= evicted.SizeBytes
		QueueMemory.Remove(evicted.SizeBytes)
		q.logDrop(evicted.Kind, evicted.SizeBytes)
		evictedCount++
	}
	// Handle capacity-based eviction
	q.nextGenID++
	if q.size == q.capacity {
		evicted := q.data[q.head]
		if q.activeGen == evicted.Generation {
			q.activeGen = 0
		}
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

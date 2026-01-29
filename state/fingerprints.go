package state

import (
	"github.com/brentp/intintmap"
	"github.com/pganalyze/collector/util"
	"sync"
	"time"
)

// 1 million entries in intintmap's internal flat array takes ~16 MB
const MAX_SIZE = 1000000
const FILL_FACTOR = 0.99

type Fingerprints struct {
	cache                 *intintmap.Map
	lock                  sync.RWMutex
	cleanedAt             time.Time
	newQueryIDs           []int64
	newQueriesProcessedAt time.Time
}

func NewFingerprints() *Fingerprints {
	return &Fingerprints{
		cache:                 intintmap.New(MAX_SIZE, FILL_FACTOR),
		lock:                  sync.RWMutex{},
		cleanedAt:             time.Now(),
		newQueriesProcessedAt: time.Now(),
	}
}

func (c *Fingerprints) Get(queryID int64, hasQueryText bool) (fingerprint int64, exists bool) {
	c.lock.RLock()
	fingerprint, exists = c.cache.Get(queryID)
	c.lock.RUnlock()
	if !exists && !hasQueryText {
		c.lock.Lock()
		c.newQueryIDs = append(c.newQueryIDs, queryID)
		c.lock.Unlock()
	}
	return
}

func (c *Fingerprints) Add(queryID int64, text string, filterQueryText string, trackActivityQuerySize int) (fingerprint int64, new bool) {
	if queryID == 0 {
		fingerprint = int64(util.FingerprintQuery(text, filterQueryText, trackActivityQuerySize))
		new = true // Treat missing query ID as a query that's always new
		return
	}
	fingerprint, exists := c.Get(queryID, true)
	if exists {
		return
	}
	c.lock.Lock()
	fingerprint = int64(util.FingerprintQuery(text, filterQueryText, trackActivityQuerySize))
	c.cache.Put(queryID, fingerprint)
	c.lock.Unlock()
	new = true
	return
}

// Called by GetStatementTexts to only look up query texts for new, unknown query IDs
func (c *Fingerprints) ProcessNewQueryIDs() []int64 {
	c.lock.Lock()
	defer c.lock.Unlock()
	newQueryIDs := c.newQueryIDs
	if len(newQueryIDs) >= 1000 || time.Since(c.newQueriesProcessedAt).Minutes() >= 5 {
		c.newQueryIDs = nil
		c.newQueriesProcessedAt = time.Now()
		return newQueryIDs
	}
	return nil
}

func (c *Fingerprints) Size() int {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.cache.Size()
}

// Retains 33% of entries, in an effort to avoid re-fingerprinting common queries.
// intintmap can't evict less used entries so isn't as CPU-efficient as an LRU cache,
// but since it's backed by a flat array it's much more memory-efficient. That
// allows us to have a larger cache that doesn't need to be emptied as often.
//
// This runs either if the max size is reached, or if enough time has passed to ensure
// new queries get reported to pganalyze even if they were in a lost snapshot.
func (c *Fingerprints) Cleanup() {
	if c.Size() < MAX_SIZE || time.Since(c.cleanedAt).Hours() < 6 {
		return
	}
	cache := intintmap.New(MAX_SIZE, FILL_FACTOR)
	index := 0
	c.cache.Each(func(key, value int64) {
		if index%3 == 0 {
			cache.Put(key, value)
		}
		index += 1
	})
	c.lock.Lock()
	c.cache = cache
	c.cleanedAt = time.Now()
	c.lock.Unlock()
}

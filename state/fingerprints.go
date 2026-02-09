package state

import (
	"github.com/pganalyze/collector/util"
	"sync"
)

// 500,000 entries use around 12 MB
const MAX_SIZE = 500_000

type Fingerprints struct {
	lock        sync.RWMutex
	cache       map[int64]uint64
	newQueryIDs []int64
}

func NewFingerprints() *Fingerprints {
	return &Fingerprints{
		lock:  sync.RWMutex{},
		cache: make(map[int64]uint64, MAX_SIZE),
	}
}

func (c *Fingerprints) Get(queryID int64) (uint64, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	fingerprint, exists := c.cache[queryID]
	return fingerprint, exists
}

func (c *Fingerprints) Add(queryID int64, text string, filterQueryText string, trackActivityQuerySize int) uint64 {
	if queryID == 0 {
		return util.FingerprintQuery(text, filterQueryText, trackActivityQuerySize)
	}
	fingerprint, exists := c.Get(queryID)
	if exists {
		return fingerprint
	}
	fingerprint, virtual := util.TryFingerprintQuery(text, filterQueryText, trackActivityQuerySize)
	if virtual {
		c.lock.Lock()
		c.newQueryIDs = append(c.newQueryIDs, queryID)
		c.lock.Unlock()
		return fingerprint
	}
	c.cleanup()
	c.lock.Lock()
	c.cache[queryID] = fingerprint
	c.lock.Unlock()
	return fingerprint
}

// Called by GetStatementTexts to only look up query texts for new, unknown query IDs
func (c *Fingerprints) TakeNewQueryIDs() []int64 {
	c.lock.Lock()
	defer c.lock.Unlock()
	newQueryIDs := c.newQueryIDs
	c.newQueryIDs = nil
	return newQueryIDs
}

func (c *Fingerprints) size() int {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return len(c.cache)
}

// Retains a random 50% sample of entries if the cache grows too large
func (c *Fingerprints) cleanup() {
	if c.size() < MAX_SIZE {
		return
	}
	c.lock.Lock()
	cache := make(map[int64]uint64, MAX_SIZE)
	index := 0
	for key, value := range c.cache {
		if index%2 == 0 {
			cache[key] = value
		}
		index += 1
	}
	c.cache = cache
	c.lock.Unlock()
}

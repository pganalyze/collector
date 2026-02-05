package state

import (
	"github.com/pganalyze/collector/util"
	"sync"
)

// 500,000 entries use around 12 MB
const MAX_SIZE = 500_000

type Fingerprints struct {
	cache map[int64]int64
	lock  sync.RWMutex
}

func NewFingerprints() *Fingerprints {
	return &Fingerprints{
		cache: make(map[int64]int64, MAX_SIZE),
		lock:  sync.RWMutex{},
	}
}

func (c *Fingerprints) Get(queryID int64) (int64, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	fingerprint, exists := c.cache[queryID]
	return fingerprint, exists
}

func (c *Fingerprints) Add(queryID int64, text string, filterQueryText string, trackActivityQuerySize int) int64 {
	if queryID == 0 {
		return int64(util.FingerprintQuery(text, filterQueryText, trackActivityQuerySize))
	}
	fingerprint, exists := c.Get(queryID)
	if exists {
		return fingerprint
	}
	fp, virtual := util.TryFingerprintQuery(text, filterQueryText, trackActivityQuerySize)
	fingerprint = int64(fp)
	if virtual {
		// Don't store virtual fingerprints so we can cache real fingerprints later
		return fingerprint
	}
	c.cleanup()
	c.lock.Lock()
	c.cache[queryID] = fingerprint
	c.lock.Unlock()
	return fingerprint
}

func (c *Fingerprints) size() int {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return len(c.cache)
}

// Retains a random 33% sample of entries if the cache grows too large
func (c *Fingerprints) cleanup() {
	if c.size() < MAX_SIZE {
		return
	}
	c.lock.Lock()
	cache := make(map[int64]int64, MAX_SIZE)
	index := 0
	for key, value := range c.cache {
		if index%3 == 0 {
			cache[key] = value
		}
		index += 1
	}
	c.cache = cache
	c.lock.Unlock()
}

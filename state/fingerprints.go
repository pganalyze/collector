package state

import (
	"github.com/pganalyze/collector/util"
	"sync"
	"sync/atomic"
)

// This uses around 9 MB per server, as measured by MemStats
const MAX_SIZE = 450_000

type Fingerprints struct {
	cache map[int64]uint64
	lock  sync.RWMutex
	size  atomic.Int32
}

func NewFingerprints() *Fingerprints {
	return &Fingerprints{
		cache: make(map[int64]uint64, MAX_SIZE),
		lock:  sync.RWMutex{},
	}
}

func (c *Fingerprints) Load(queryID int64) (uint64, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	fingerprint, exists := c.cache[queryID]
	return fingerprint, exists
}

func (c *Fingerprints) LoadOrStore(queryID int64, text string, filterQueryText string, trackActivityQuerySize int) uint64 {
	fingerprint, exists := c.Load(queryID)
	if exists {
		return fingerprint
	}
	fingerprint, virtual := util.TryFingerprintQuery(text, filterQueryText, trackActivityQuerySize)
	if virtual {
		// Don't store virtual fingerprints so we can cache real fingerprints later
		return fingerprint
	}
	c.cleanup()
	c.lock.Lock()
	defer c.lock.Unlock()
	c.cache[queryID] = fingerprint
	c.size.Add(1)
	return fingerprint
}

// Retains a random 50% sample of entries if the cache grows too large
func (c *Fingerprints) cleanup() {
	if c.size.Load() < MAX_SIZE {
		return
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	cache := make(map[int64]uint64, MAX_SIZE)
	index := 0
	for key, value := range c.cache {
		if index%2 == 0 {
			cache[key] = value
		}
		index += 1
	}
	c.size.Store(int32(len(cache)))
	c.cache = cache
}

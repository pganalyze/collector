package state

import (
	"github.com/brentp/intintmap"
	"github.com/pganalyze/collector/util"
	"sync"
)

// 500 thousand entries in intintmap's internal flat array takes ~8 MB
const MAX_SIZE = 500000
const FILL_FACTOR = 0.99

type Fingerprints struct {
	cache *intintmap.Map
	lock  sync.RWMutex
}

func NewFingerprints() *Fingerprints {
	return &Fingerprints{
		cache: intintmap.New(MAX_SIZE, FILL_FACTOR),
		lock:  sync.RWMutex{},
	}
}

func (c *Fingerprints) Get(queryID int64) (int64, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.cache.Get(queryID)
}

func (c *Fingerprints) Add(queryID int64, text string, filterQueryText string, trackActivityQuerySize int) int64 {
	if queryID == 0 {
		return int64(util.FingerprintQuery(text, filterQueryText, trackActivityQuerySize))
	}
	fingerprint, exists := c.Get(queryID)
	if exists {
		return fingerprint
	}
	c.lock.Lock()
	fingerprint = int64(util.FingerprintQuery(text, filterQueryText, trackActivityQuerySize))
	c.cache.Put(queryID, fingerprint)
	c.lock.Unlock()
	return fingerprint
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
func (c *Fingerprints) Cleanup() {
	if c.Size() < MAX_SIZE {
		return
	}
	c.lock.Lock()
	cache := intintmap.New(MAX_SIZE, FILL_FACTOR)
	index := 0
	c.cache.Each(func(key, value int64) {
		if index%3 == 0 {
			cache.Put(key, value)
		}
		index += 1
	})
	c.cache = cache
	c.lock.Unlock()
}

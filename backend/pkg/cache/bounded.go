package cache

import (
	"sync"
)

const DefaultBoundedCacheSize = 100

type BoundedCache[K comparable, V any] struct {
	mu      sync.RWMutex
	entries []boundedEntry[K, V]
	index   map[K]int
	maxSize int
}

type boundedEntry[K comparable, V any] struct {
	key   K
	value V
}

func NewBoundedCache[K comparable, V any](maxSize int) *BoundedCache[K, V] {
	if maxSize <= 0 {
		maxSize = DefaultBoundedCacheSize
	}
	return &BoundedCache[K, V]{
		entries: make([]boundedEntry[K, V], 0, maxSize),
		index:   make(map[K]int, maxSize),
		maxSize: maxSize,
	}
}

func (c *BoundedCache[K, V]) Get(key K) (V, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if idx, ok := c.index[key]; ok {
		return c.entries[idx].value, true
	}
	var zero V
	return zero, false
}

func (c *BoundedCache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.index[key]; ok {
		return
	}

	if len(c.entries) >= c.maxSize {
		oldest := c.entries[0]
		delete(c.index, oldest.key)
		c.entries = c.entries[1:]
		for k, idx := range c.index {
			c.index[k] = idx - 1
		}
	}

	c.index[key] = len(c.entries)
	c.entries = append(c.entries, boundedEntry[K, V]{key: key, value: value})
}

func (c *BoundedCache[K, V]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

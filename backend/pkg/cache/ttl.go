package cache

import (
	"sync"
	"time"
)

type ttlEntry[V any] struct {
	value     V
	expiresAt int64
}

// TTLCache is a thread-safe in-memory cache with per-entry TTL.
// Expired entries are lazily evicted on access.
type TTLCache[K comparable, V any] struct {
	mu      sync.RWMutex
	entries map[K]ttlEntry[V]
	ttl     time.Duration
	maxSize int
}

func NewTTLCache[K comparable, V any](ttl time.Duration, maxSize int) *TTLCache[K, V] {
	if maxSize <= 0 {
		maxSize = 256
	}
	return &TTLCache[K, V]{
		entries: make(map[K]ttlEntry[V], maxSize),
		ttl:     ttl,
		maxSize: maxSize,
	}
}

func (c *TTLCache[K, V]) Get(key K) (V, bool) {
	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok {
		var zero V
		return zero, false
	}
	if time.Now().UnixNano() <= entry.expiresAt {
		return entry.value, true
	}
	c.mu.Lock()
	entry, ok = c.entries[key]
	if ok && time.Now().UnixNano() > entry.expiresAt {
		delete(c.entries, key)
		ok = false
	}
	c.mu.Unlock()
	if !ok {
		var zero V
		return zero, false
	}
	return entry.value, true
}

func (c *TTLCache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.entries) >= c.maxSize {
		now := time.Now().UnixNano()
		for k, e := range c.entries {
			if now > e.expiresAt {
				delete(c.entries, k)
			}
		}
		if len(c.entries) >= c.maxSize {
			for k := range c.entries {
				delete(c.entries, k)
				break
			}
		}
	}

	c.entries[key] = ttlEntry[V]{
		value:     value,
		expiresAt: time.Now().Add(c.ttl).UnixNano(),
	}
}

func (c *TTLCache[K, V]) Delete(key K) {
	c.mu.Lock()
	delete(c.entries, key)
	c.mu.Unlock()
}

// DeleteAll evicts every entry from the cache regardless of TTL.
func (c *TTLCache[K, V]) DeleteAll() {
	c.mu.Lock()
	clear(c.entries)
	c.mu.Unlock()
}

package cache

import (
	"container/list"
	"sync"
	"time"
)

type ttlEntry[K comparable, V any] struct {
	value     V
	expiresAt int64
	elem      *list.Element
}

// TTLCache is a thread-safe in-memory cache with per-entry TTL and LRU eviction.
// Expired entries are lazily evicted on access. When at capacity, the least recently used key is evicted.
type TTLCache[K comparable, V any] struct {
	mu      sync.RWMutex
	entries map[K]ttlEntry[K, V]
	lru     *list.List
	ttl     time.Duration
	maxSize int
}

func NewTTLCache[K comparable, V any](ttl time.Duration, maxSize int) *TTLCache[K, V] {
	if maxSize <= 0 {
		maxSize = 256
	}
	return &TTLCache[K, V]{
		entries: make(map[K]ttlEntry[K, V], maxSize),
		lru:     list.New(),
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
		c.mu.Lock()
		if e, ok := c.entries[key]; ok && time.Now().UnixNano() <= e.expiresAt {
			c.lru.MoveToBack(e.elem)
			v := e.value
			c.mu.Unlock()
			return v, true
		}
		c.mu.Unlock()
		var zero V
		return zero, false
	}
	c.mu.Lock()
	entry, ok = c.entries[key]
	if ok && time.Now().UnixNano() > entry.expiresAt {
		c.lru.Remove(entry.elem)
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

	now := time.Now().UnixNano()
	for k, e := range c.entries {
		if now > e.expiresAt {
			c.lru.Remove(e.elem)
			delete(c.entries, k)
		}
	}
	if ent, exists := c.entries[key]; exists {
		ent.value = value
		ent.expiresAt = time.Now().Add(c.ttl).UnixNano()
		c.entries[key] = ent
		c.lru.MoveToBack(ent.elem)
		return
	}
	if len(c.entries) >= c.maxSize {
		front := c.lru.Front()
		if front != nil {
			evictKey, ok := front.Value.(K)
			if ok {
				c.lru.Remove(front)
				delete(c.entries, evictKey)
			}
		}
	}
	elem := c.lru.PushBack(key)
	c.entries[key] = ttlEntry[K, V]{
		value:     value,
		expiresAt: time.Now().Add(c.ttl).UnixNano(),
		elem:      elem,
	}
}

func (c *TTLCache[K, V]) Delete(key K) {
	c.mu.Lock()
	if e, ok := c.entries[key]; ok {
		c.lru.Remove(e.elem)
		delete(c.entries, key)
	}
	c.mu.Unlock()
}

// DeleteAll evicts every entry from the cache regardless of TTL.
func (c *TTLCache[K, V]) DeleteAll() {
	c.mu.Lock()
	c.entries = make(map[K]ttlEntry[K, V], c.maxSize)
	c.lru = list.New()
	c.mu.Unlock()
}

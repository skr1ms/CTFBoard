package cache

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBoundedCache_GetSet(t *testing.T) {
	cache := NewBoundedCache[string, int](10)

	val, ok := cache.Get("test")
	assert.False(t, ok)
	assert.Equal(t, 0, val)

	cache.Set("test", 42)

	val, ok = cache.Get("test")
	assert.True(t, ok)
	assert.Equal(t, 42, val)
}

func TestBoundedCache_Eviction(t *testing.T) {
	maxSize := 10
	cache := NewBoundedCache[string, int](maxSize)

	for i := 0; i < maxSize+5; i++ {
		cache.Set(string(rune('a'+i)), i)
	}

	assert.Equal(t, maxSize, cache.Len())

	_, ok := cache.Get("a")
	assert.False(t, ok)

	val, ok := cache.Get(string(rune('a' + 5)))
	assert.True(t, ok)
	assert.Equal(t, 5, val)
}

func TestBoundedCache_NoDuplicates(t *testing.T) {
	cache := NewBoundedCache[string, int](10)

	cache.Set("test", 1)
	cache.Set("test", 2)

	assert.Equal(t, 1, cache.Len())

	val, ok := cache.Get("test")
	require.True(t, ok)
	assert.Equal(t, 1, val)
}

func TestBoundedCache_Concurrent(t *testing.T) {
	cache := NewBoundedCache[int, int](50)

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				key := id*100 + j
				cache.Set(key, key)
				cache.Get(key)
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	assert.LessOrEqual(t, cache.Len(), 50)
}

func TestBoundedCache_DefaultSize(t *testing.T) {
	cache := NewBoundedCache[string, int](0)
	assert.Equal(t, DefaultBoundedCacheSize, cache.maxSize)
}

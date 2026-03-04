package cache

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTTLCache_SetAndGet(t *testing.T) {
	t.Parallel()
	c := NewTTLCache[string, int](time.Minute, 10)
	c.Set("a", 1)
	v, ok := c.Get("a")
	require.True(t, ok)
	assert.Equal(t, 1, v)
}

func TestTTLCache_Miss(t *testing.T) {
	t.Parallel()
	c := NewTTLCache[string, int](time.Minute, 10)
	_, ok := c.Get("missing")
	assert.False(t, ok)
}

func TestTTLCache_Expiry(t *testing.T) {
	t.Parallel()
	c := NewTTLCache[string, int](10*time.Millisecond, 10)
	c.Set("a", 42)
	time.Sleep(20 * time.Millisecond)
	_, ok := c.Get("a")
	assert.False(t, ok)
}

func TestTTLCache_Delete(t *testing.T) {
	t.Parallel()
	c := NewTTLCache[string, int](time.Minute, 10)
	c.Set("a", 1)
	c.Delete("a")
	_, ok := c.Get("a")
	assert.False(t, ok)
}

func TestTTLCache_MaxSize_Eviction(t *testing.T) {
	t.Parallel()
	c := NewTTLCache[int, int](time.Minute, 3)
	c.Set(1, 1)
	c.Set(2, 2)
	c.Set(3, 3)
	c.Set(4, 4)
	count := 0
	for i := 1; i <= 4; i++ {
		if _, ok := c.Get(i); ok {
			count++
		}
	}
	assert.LessOrEqual(t, count, 3)
}

func TestTTLCache_ConcurrentGetSet(t *testing.T) {
	t.Parallel()

	const iterations = 500
	c := NewTTLCache[string, int](1*time.Millisecond, 128)

	var wg sync.WaitGroup
	for i := range iterations {
		wg.Add(2)
		go func(val int) {
			defer wg.Done()
			c.Set("key", val)
		}(i)
		go func() {
			defer wg.Done()
			time.Sleep(2 * time.Millisecond)
			c.Get("key")
		}()
	}
	wg.Wait()

	c.Set("final", 999)
	v, ok := c.Get("final")
	require.True(t, ok)
	assert.Equal(t, 999, v)
}

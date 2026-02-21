package cache

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLRU_SetAndGet(t *testing.T) {
	c := NewLRU(10)

	c.Set("key1", "value1", 5*time.Minute)
	val, ok := c.Get("key1")
	require.True(t, ok)
	assert.Equal(t, "value1", val)
}

func TestLRU_GetMiss(t *testing.T) {
	c := NewLRU(10)

	val, ok := c.Get("nonexistent")
	assert.False(t, ok)
	assert.Nil(t, val)
}

func TestLRU_Expiration(t *testing.T) {
	c := NewLRU(10)

	c.Set("expired", "old", 1*time.Millisecond)
	time.Sleep(5 * time.Millisecond)

	val, ok := c.Get("expired")
	assert.False(t, ok)
	assert.Nil(t, val)
}

func TestLRU_Update(t *testing.T) {
	c := NewLRU(10)

	c.Set("key", "v1", 5*time.Minute)
	c.Set("key", "v2", 5*time.Minute)

	val, ok := c.Get("key")
	require.True(t, ok)
	assert.Equal(t, "v2", val)
	assert.Equal(t, 1, c.Len())
}

func TestLRU_Eviction(t *testing.T) {
	c := NewLRU(3)

	c.Set("a", 1, 5*time.Minute)
	c.Set("b", 2, 5*time.Minute)
	c.Set("c", 3, 5*time.Minute)

	// Access "a" to make it recently used
	c.Get("a")

	// Adding "d" should evict "b" (least recently used)
	c.Set("d", 4, 5*time.Minute)

	_, ok := c.Get("b")
	assert.False(t, ok, "b should have been evicted")

	val, ok := c.Get("a")
	require.True(t, ok, "a should still exist")
	assert.Equal(t, 1, val)

	val, ok = c.Get("d")
	require.True(t, ok)
	assert.Equal(t, 4, val)
}

func TestLRU_Delete(t *testing.T) {
	c := NewLRU(10)

	c.Set("key", "value", 5*time.Minute)
	c.Delete("key")

	_, ok := c.Get("key")
	assert.False(t, ok)
	assert.Equal(t, 0, c.Len())
}

func TestLRU_DeletePrefix(t *testing.T) {
	c := NewLRU(10)

	c.Set("channels:srv1:list", "ch1", 5*time.Minute)
	c.Set("channels:srv1:count", "10", 5*time.Minute)
	c.Set("channels:srv2:list", "ch2", 5*time.Minute)
	c.Set("members:srv1:list", "m1", 5*time.Minute)

	c.DeletePrefix("channels:srv1:")

	_, ok := c.Get("channels:srv1:list")
	assert.False(t, ok)
	_, ok = c.Get("channels:srv1:count")
	assert.False(t, ok)

	// Other prefixes should remain
	val, ok := c.Get("channels:srv2:list")
	require.True(t, ok)
	assert.Equal(t, "ch2", val)

	val, ok = c.Get("members:srv1:list")
	require.True(t, ok)
	assert.Equal(t, "m1", val)
}

func TestLRU_DefaultMaxSize(t *testing.T) {
	c := NewLRU(0)
	assert.Equal(t, 1024, c.maxSize)

	c2 := NewLRU(-5)
	assert.Equal(t, 1024, c2.maxSize)
}

func TestLRU_ConcurrentAccess(t *testing.T) {
	c := NewLRU(100)
	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c.Set(fmt.Sprintf("key-%d", i), i, 5*time.Minute)
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c.Get(fmt.Sprintf("key-%d", i))
		}(i)
	}

	// Concurrent deletes
	for i := 0; i < 25; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c.Delete(fmt.Sprintf("key-%d", i))
		}(i)
	}

	wg.Wait()
	// No race conditions or panics = pass
}

func TestLRU_DeleteNonexistent(t *testing.T) {
	c := NewLRU(10)
	// Should not panic
	c.Delete("nonexistent")
	c.DeletePrefix("none:")
}

func TestLRU_TypedValues(t *testing.T) {
	c := NewLRU(10)

	type Server struct {
		ID   string
		Name string
	}

	srv := &Server{ID: "1", Name: "Test"}
	c.Set("server:1", srv, 5*time.Minute)

	val, ok := c.Get("server:1")
	require.True(t, ok)
	got := val.(*Server)
	assert.Equal(t, "1", got.ID)
	assert.Equal(t, "Test", got.Name)
}

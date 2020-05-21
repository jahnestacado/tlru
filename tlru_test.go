package tlru

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	entry1 = Entry{"entry1", 1}
	entry2 = Entry{"entry2", 2}
	entry3 = Entry{"entry3", 3}
	entry4 = Entry{"entry4", 4}
)

func TestLRUCacheSet(t *testing.T) {
	evictionChannel := make(chan CacheEntry, 2)
	config := Config{
		Size:            2,
		TTL:             time.Millisecond,
		EvictionChannel: &evictionChannel,
	}

	cache := New(config)
	cache.Set(entry1)
	cache.Set(entry2)
	cache.Set(entry2)
	cache.Set(entry4)
	cache.Set(entry2)
	cache.Set(entry3)

	cachedEntryValue1 := cache.Get(entry1.Key)
	cachedEntryValue2 := cache.Get(entry2.Key)
	cachedEntryValue3 := cache.Get(entry3.Key)
	cachedEntryValue4 := cache.Get(entry4.Key)

	evictedEntry1 := <-evictionChannel
	evictedEntry4 := <-evictionChannel

	assert := assert.New(t)
	assert.Nil(cachedEntryValue1)
	assert.Nil(cachedEntryValue4)
	assert.Equal(entry1.Key, evictedEntry1.Key)
	assert.Equal(entry2.Value, cachedEntryValue2)
	assert.Equal(entry3.Value, cachedEntryValue3)
	assert.Equal(entry4.Key, evictedEntry4.Key)
}

func TestLRUCacheSetExpiration(t *testing.T) {
	evictionChannel := make(chan CacheEntry, 4)
	config := Config{
		Size:            10,
		TTL:             time.Millisecond,
		EvictionChannel: &evictionChannel,
	}

	cache := New(config)
	cache.Set(entry1)
	cache.Set(entry2)
	cache.Set(entry2)
	cache.Set(entry4)
	cache.Set(entry2)
	cache.Set(entry3)
	time.Sleep(time.Millisecond)

	cachedEntryValue1 := cache.Get(entry1.Key)
	cachedEntryValue2 := cache.Get(entry2.Key)
	cachedEntryValue3 := cache.Get(entry3.Key)
	cachedEntryValue4 := cache.Get(entry4.Key)

	evictedEntry1 := <-evictionChannel
	evictedEntry2 := <-evictionChannel
	evictedEntry3 := <-evictionChannel
	evictedEntry4 := <-evictionChannel

	assert := assert.New(t)
	assert.Nil(cachedEntryValue1)
	assert.Nil(cachedEntryValue2)
	assert.Nil(cachedEntryValue3)
	assert.Nil(cachedEntryValue4)
	assert.Equal(entry1.Key, evictedEntry1.Key)
	assert.Equal(entry2.Key, evictedEntry2.Key)
	assert.Equal(entry3.Key, evictedEntry3.Key)
	assert.Equal(entry4.Key, evictedEntry4.Key)
}

func TestLRUCacheKeys(t *testing.T) {
	config := Config{
		Size: 10,
		TTL:  time.Millisecond,
	}
	cache := New(config)

	cache.Set(entry1)
	cache.Set(entry2)
	cache.Set(entry2)
	cache.Set(entry4)
	cache.Set(entry3)
	keys := cache.Keys()

	assert := assert.New(t)
	assert.Equal(4, len(keys))
	assert.Contains(keys, entry1.Key)
	assert.Contains(keys, entry2.Key)
	assert.Contains(keys, entry3.Key)
	assert.Contains(keys, entry4.Key)
}

func TestLRUCacheKeysOneExpired(t *testing.T) {
	evictionChannel := make(chan CacheEntry, 1)
	config := Config{
		Size:            10,
		TTL:             time.Millisecond,
		EvictionChannel: &evictionChannel,
	}
	cache := New(config)

	cache.Set(entry1)
	time.Sleep(time.Millisecond)
	cache.Set(entry2)
	cache.Set(entry2)
	cache.Set(entry4)
	cache.Set(entry3)
	keys := cache.Keys()

	evictedEntry1 := <-evictionChannel

	assert := assert.New(t)
	assert.Equal(3, len(keys))
	assert.NotContains(keys, entry1.Key)
	assert.Equal(entry1.Key, evictedEntry1.Key)
	assert.Contains(keys, entry2.Key)
	assert.Contains(keys, entry3.Key)
	assert.Contains(keys, entry4.Key)
}

func TestLRUCacheKeysAllExpired(t *testing.T) {
	config := Config{
		Size: 10,
		TTL:  time.Millisecond,
	}
	cache := New(config)

	cache.Set(entry1)
	cache.Set(entry2)
	cache.Set(entry2)
	cache.Set(entry4)
	cache.Set(entry3)
	time.Sleep(time.Millisecond)
	keys := cache.Keys()

	assert := assert.New(t)
	assert.Equal(0, len(keys))
}

func TestLRUCacheValues(t *testing.T) {
	config := Config{
		Size: 10,
		TTL:  time.Millisecond,
	}
	cache := New(config)

	cache.Set(entry1)
	cache.Set(entry2)
	cache.Set(entry2)
	cache.Set(entry4)
	cache.Set(entry3)

	assert := assert.New(t)
	values := cache.Values()
	assert.Equal(4, len(values))
	assert.Contains(values, entry1.Value)
	assert.Contains(values, entry2.Value)
	assert.Contains(values, entry3.Value)
	assert.Contains(values, entry4.Value)
}

func TestLRUCacheValuesOneExpired(t *testing.T) {
	evictionChannel := make(chan CacheEntry, 1)
	config := Config{
		Size:            10,
		TTL:             time.Millisecond,
		EvictionChannel: &evictionChannel,
	}
	cache := New(config)

	cache.Set(entry1)
	time.Sleep(time.Millisecond)
	cache.Set(entry2)
	cache.Set(entry2)
	cache.Set(entry4)
	cache.Set(entry3)
	values := cache.Values()

	evictedEntry1 := <-evictionChannel

	assert := assert.New(t)
	assert.Equal(3, len(values))
	assert.NotContains(values, entry1.Value)
	assert.Equal(entry1.Value, evictedEntry1.Value)
	assert.Contains(values, entry2.Value)
	assert.Contains(values, entry3.Value)
	assert.Contains(values, entry4.Value)
}

func TestLRUCacheValuesAllExpired(t *testing.T) {
	evictionChannel := make(chan CacheEntry, 4)
	config := Config{
		Size:            10,
		TTL:             time.Millisecond,
		EvictionChannel: &evictionChannel,
	}
	cache := New(config)

	cache.Set(entry1)
	cache.Set(entry2)
	cache.Set(entry2)
	cache.Set(entry4)
	cache.Set(entry3)
	time.Sleep(time.Millisecond)
	values := cache.Values()

	evictedEntry1 := <-evictionChannel
	evictedEntry2 := <-evictionChannel
	evictedEntry4 := <-evictionChannel
	evictedEntry3 := <-evictionChannel

	assert := assert.New(t)
	assert.Equal(0, len(values))
	assert.Equal(entry1.Value, evictedEntry1.Value)
	assert.Equal(entry2.Value, evictedEntry2.Value)
	assert.Equal(entry3.Value, evictedEntry3.Value)
	assert.Equal(entry4.Value, evictedEntry4.Value)
}

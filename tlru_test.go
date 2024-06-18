// * tlru <https://github.com/jahnestacado/tlru>
// * Copyright (c) 2020 Ioannis Tzanellis
// * Licensed under the MIT License (MIT).
package tlru

import (
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	entry1   = Entry[string, int]{Key: "entry1", Value: 1}
	entry2   = Entry[string, int]{Key: "entry2", Value: 2}
	entry3   = Entry[string, int]{Key: "entry3", Value: 3}
	entry4   = Entry[string, int]{Key: "entry4", Value: 4}
	policies = []evictionPolicy{LRA, LRI}
)

// Unit tests
// -----------------------------------------------------------------------------
func TestLRUCacheSetAndGet(t *testing.T) {
	assert := assert.New(t)
	for _, policy := range policies {
		config := Config[string, int]{
			MaxSize:        1,
			TTL:            time.Minute,
			EvictionPolicy: policy,
		}
		cache := New(config)

		cache.Set(entry1.Key, entry1.Value)

		cachedEntry1 := cache.Get(entry1.Key)
		nonExistentEntry := cache.Get("non-existent-key")

		assert.Equal(entry1.Value, cachedEntry1.Value)
		assert.Nil(nonExistentEntry)
	}
}

func TestLRUCacheDelete(t *testing.T) {
	assert := assert.New(t)
	for _, policy := range policies {
		config := Config[string, int]{
			MaxSize:        10,
			TTL:            time.Minute,
			EvictionPolicy: policy,
		}
		cache := New(config)

		cache.Set(entry1.Key, entry1.Value)
		cache.Set(entry2.Key, entry2.Value)

		entries := cache.Keys()
		assert.Equal(2, len(entries))
		cachedEntry1 := cache.Get(entry1.Key)
		cachedEntry2 := cache.Get(entry2.Key)
		assert.Equal(entry1.Value, cachedEntry1.Value)
		assert.Equal(entry2.Value, cachedEntry2.Value)

		cache.Delete(entry2.Key)
		cache.Delete("non-existent-key")

		entries = cache.Keys()
		assert.Equal(1, len(entries))
		cachedEntry1 = cache.Get(entry1.Key)
		cachedEntry2 = cache.Get(entry2.Key)
		assert.Equal(entry1.Value, cachedEntry1.Value)
		assert.Nil(cachedEntry2)
	}
}

func TestLRUCacheKeys(t *testing.T) {
	assert := assert.New(t)
	for _, policy := range policies {
		config := Config[string, int]{
			MaxSize:        10,
			TTL:            time.Minute,
			EvictionPolicy: policy,
		}
		cache := New(config)

		cache.Set(entry1.Key, entry1.Value)
		cache.Set(entry2.Key, entry2.Value)
		cache.Set(entry2.Key, entry2.Value)
		cache.Set(entry4.Key, entry4.Value)
		cache.Set(entry3.Key, entry3.Value)
		keys := cache.Keys()

		assert.Equal(4, len(keys))
		assert.Contains(keys, entry1.Key)
		assert.Contains(keys, entry2.Key)
		assert.Contains(keys, entry3.Key)
		assert.Contains(keys, entry4.Key)
	}
}

func TestLRUCacheEntries(t *testing.T) {
	assert := assert.New(t)
	for _, policy := range policies {
		config := Config[string, int]{
			MaxSize:        10,
			TTL:            time.Minute,
			EvictionPolicy: policy,
		}
		cache := New(config)

		cache.Set(entry1.Key, entry1.Value)
		cache.Set(entry2.Key, entry2.Value)
		cache.Set(entry2.Key, entry2.Value)
		cache.Set(entry4.Key, entry4.Value)
		cache.Set(entry3.Key, entry3.Value)

		cachedEntries := cache.Entries()
		assert.Equal(4, len(cachedEntries))
		entries := map[interface{}]Entry[string, int]{
			entry1.Value: entry1,
			entry2.Value: entry2,
			entry3.Value: entry3,
			entry4.Value: entry4,
		}
		for _, cachedEntry := range cachedEntries {
			assert.Equal(entries[cachedEntry.Value].Value, cachedEntry.Value)
		}
	}
}

func TestCacheClear(t *testing.T) {
	assert := assert.New(t)
	for _, policy := range policies {
		config := Config[string, int]{
			MaxSize:        10,
			TTL:            time.Second,
			EvictionPolicy: policy,
		}
		cache := New(config)

		cache.Set(entry1.Key, entry1.Value)
		cache.Set(entry2.Key, entry2.Value)

		entries := cache.Keys()
		assert.Equal(2, len(entries))
		cachedEntry1 := cache.Get(entry1.Key)
		cachedEntry2 := cache.Get(entry2.Key)
		assert.Equal(entry1.Value, cachedEntry1.Value)
		assert.Equal(entry2.Value, cachedEntry2.Value)

		cache.Clear()

		entriesAfterClear := cache.Keys()
		assert.Equal(0, len(entriesAfterClear))
		cachedEntry1 = cache.Get(entry1.Key)
		cachedEntry2 = cache.Get(entry2.Key)
		assert.Nil(cachedEntry1)
		assert.Nil(cachedEntry2)

		cache.Set(entry2.Key, entry2.Value)
		cachedEntry2 = cache.Get(entry2.Key)
		assert.Equal(entry2.Value, cachedEntry2.Value)

		cache.Set(entry1.Key, entry1.Value)
		time.Sleep(2 * config.TTL)
		entriesAfterTTLExpired := cache.Keys()
		assert.Equal(0, len(entriesAfterTTLExpired))
	}
}

func TestLRUCacheTTLEvictionDaemon(t *testing.T) {
	assert := assert.New(t)
	for _, policy := range policies {
		evictionChannel := make(chan EvictedEntry[string, int], 0)
		ttl := 5 * time.Millisecond
		config := Config[string, int]{
			MaxSize:                   10,
			TTL:                       ttl,
			EvictionChannel:           &evictionChannel,
			EvictionPolicy:            policy,
			GarbageCollectionInterval: ttl,
		}
		cache := New(config)

		var (
			evictedEntry1 EvictedEntry[string, int]
			evictedEntry2 EvictedEntry[string, int]
		)

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			evictedEntry1 = <-evictionChannel
			evictedEntry2 = <-evictionChannel
		}()

		expiredEntryTimestamp := time.Date(1900, 2, 1, 12, 30, 0, 0, time.UTC)
		expiredEntry1 := Entry[string, int]{Key: "expired-1", Value: 1, Timestamp: &expiredEntryTimestamp}
		expiredEntry2 := Entry[string, int]{Key: "expired-2", Value: 2, Timestamp: &expiredEntryTimestamp}
		nonExpiredEntryTimestamp := time.Now().Add(time.Minute)
		nonExpiredEntry := Entry[string, int]{Key: "non-expired", Value: 1, Timestamp: &nonExpiredEntryTimestamp}

		cache.SetWithTimestamp(nonExpiredEntry.Key, nonExpiredEntry.Value, *nonExpiredEntry.Timestamp)
		cache.SetWithTimestamp(expiredEntry1.Key, expiredEntry1.Value, *expiredEntry1.Timestamp)
		cache.SetWithTimestamp(expiredEntry2.Key, expiredEntry2.Value, *expiredEntry2.Timestamp)
		wg.Wait()

		assert.Equal(expiredEntry1.Key, evictedEntry1.Key)
		assert.Equal(expiredEntry2.Key, evictedEntry2.Key)
		assert.Equal(EvictionReasonExpired, evictedEntry1.Reason)
		assert.Equal(EvictionReasonExpired, evictedEntry2.Reason)

		cachedNonExpiredEntry := cache.Get(nonExpiredEntry.Key)
		cachedExpiredEntry1 := cache.Get(expiredEntry1.Key)
		cachedExpiredEntry2 := cache.Get(expiredEntry2.Key)

		assert.Equal(nonExpiredEntry.Value, cachedNonExpiredEntry.Value)
		assert.Nil(cachedExpiredEntry1)
		assert.Nil(cachedExpiredEntry2)
	}
}

func TestLRUCacheSetAndGetWithProvidedTimestamp(t *testing.T) {
	assert := assert.New(t)
	for _, policy := range policies {
		evictionChannel := make(chan EvictedEntry[string, int], 1)
		config := Config[string, int]{
			MaxSize:         10,
			TTL:             time.Minute,
			EvictionChannel: &evictionChannel,
			EvictionPolicy:  policy,
		}
		cache := New(config)

		expiredEntryTimestamp := time.Date(1900, 2, 1, 12, 30, 0, 0, time.UTC)
		expiredEntry := Entry[string, int]{Key: "expired", Value: 1, Timestamp: &expiredEntryTimestamp}
		nonExpiredEntryTimestamp := time.Now()
		nonExpiredEntry := Entry[string, int]{Key: "non-expired", Value: 1, Timestamp: &nonExpiredEntryTimestamp}

		cache.Set(entry1.Key, entry1.Value)
		cache.SetWithTimestamp(expiredEntry.Key, expiredEntry.Value, *expiredEntry.Timestamp)
		cache.SetWithTimestamp(nonExpiredEntry.Key, nonExpiredEntry.Value, *nonExpiredEntry.Timestamp)

		cachedEntry1 := cache.Get(entry1.Key)
		cachedNonExpiredEntry := cache.Get(nonExpiredEntry.Key)
		cachedExpiredEntry := cache.Get(expiredEntry.Key)
		evictedExpiredEntry := <-evictionChannel

		assert.Equal(expiredEntry.Key, evictedExpiredEntry.Key)
		assert.Equal(EvictionReasonExpired, evictedExpiredEntry.Reason)

		assert.Equal(entry1.Value, cachedEntry1.Value)
		assert.Equal(cachedNonExpiredEntry.Value, cachedNonExpiredEntry.Value)
		assert.Nil(cachedExpiredEntry)
	}
}

func TestLRUCacheGetState(t *testing.T) {
	assert := assert.New(t)
	for _, policy := range policies {
		config := Config[string, int]{
			MaxSize:        3,
			TTL:            time.Minute,
			EvictionPolicy: policy,
		}
		cache := New(config)

		cache.Set(entry1.Key, entry1.Value)
		cache.Set(entry2.Key, entry2.Value)

		state := cache.GetState()

		assert.Equal(policy, state.EvictionPolicy)
		assert.Equal(2, len(state.Entries))
		assert.Equal(state.Entries[0].Key, entry2.Key)
		assert.Equal(state.Entries[1].Key, entry1.Key)
	}
}

func TestLRUCacheSetStateError(t *testing.T) {
	assert := assert.New(t)
	state := State[string, int]{
		EvictionPolicy: LRI,
		ExtractedAt:    time.Now(),
	}

	config := Config[string, int]{
		MaxSize: 1,
		TTL:     time.Minute,
	}
	cache := New(config)

	err := cache.SetState(state)
	assert.Error(err)
}

func TestLRUCacheSetState(t *testing.T) {
	assert := assert.New(t)
	for _, policy := range policies {
		state := State[string, int]{
			EvictionPolicy: policy,
			ExtractedAt:    time.Now(),
			Entries: []StateEntry[string, int]{
				{
					Key:        entry1.Key,
					Value:      entry1.Value,
					Counter:    1,
					LastUsedAt: time.Now(),
					CreatedAt:  time.Now(),
				},
				{
					Key:        entry2.Key,
					Value:      entry2.Value,
					Counter:    2,
					LastUsedAt: time.Date(1900, 2, 1, 12, 30, 0, 0, time.UTC),
					CreatedAt:  time.Date(1900, 2, 1, 12, 30, 0, 0, time.UTC),
				},
			},
		}
		evictionChannel := make(chan EvictedEntry[string, int], 1)
		config := Config[string, int]{
			MaxSize:         3,
			TTL:             time.Minute,
			EvictionChannel: &evictionChannel,
			EvictionPolicy:  policy,
		}
		cache := New(config)
		cache.Set(entry4.Key, entry4.Value)
		cache.Set(entry3.Key, entry3.Value)

		err := cache.SetState(state)
		assert.NoError(err)
		cachedEntry1 := cache.Get(state.Entries[0].Key)
		cachedEntry2 := cache.Get(state.Entries[1].Key)
		evictedEntry2 := <-evictionChannel
		cachedEntry3 := cache.Get(entry3.Key)
		cachedEntry4 := cache.Get(entry4.Key)

		assert.Equal(state.Entries[0].Value, cachedEntry1.Value)
		assert.Equal(int64(2)-int64(policy*policy), cachedEntry1.Counter)

		assert.Equal(entry2.Key, evictedEntry2.Key)
		assert.Equal(int64(2), evictedEntry2.Counter)
		assert.Equal(EvictionReasonExpired, evictedEntry2.Reason)
		assert.Nil(cachedEntry2)
		assert.Nil(cachedEntry3)
		assert.Nil(cachedEntry4)
	}
}

func TestLRUCacheGetStateAndSetState(t *testing.T) {
	assert := assert.New(t)

	for _, policy := range policies {
		config := Config[string, int]{
			MaxSize:        3,
			TTL:            time.Minute,
			EvictionPolicy: policy,
		}
		cache := New(config)
		cache.Set(entry4.Key, entry4.Value)
		cache.Set(entry3.Key, entry3.Value)

		entries := cache.Entries()
		assert.Equal(2, len(entries))

		state := cache.GetState()
		assert.Equal(2, len(state.Entries))

		cache.Clear()

		err := cache.SetState(state)
		assert.NoError(err)

		cachedEntry3 := cache.Get(entry3.Key)
		cachedEntry4 := cache.Get(entry4.Key)

		assert.Equal(entry3.Value, cachedEntry3.Value)
		assert.Equal(entry4.Value, cachedEntry4.Value)
	}
}

func TestEvictionReasonsToString(t *testing.T) {
	assert := assert.New(t)

	assert.Equal("Dropped", EvictionReasonDropped.String())
	assert.Equal("Expired", EvictionReasonExpired.String())
	assert.Equal("Deleted", EvictionReasonDeleted.String())
}

func TestLRUCacheHas(t *testing.T) {
	assert := assert.New(t)
	for _, policy := range policies {
		config := Config[string, int]{
			MaxSize:        10,
			TTL:            time.Minute,
			EvictionPolicy: policy,
		}
		cache := New(config)

		cache.Set(entry1.Key, entry1.Value)
		cache.Set(entry2.Key, entry2.Value)

		hasEntry1Key := cache.Has(entry1.Key)
		hasEntry2Key := cache.Has(entry2.Key)
		hasEntry3Key := cache.Has(entry3.Key)

		assert.True(hasEntry1Key)
		assert.True(hasEntry2Key)
		assert.False(hasEntry3Key)
	}
}

// Integration tests - LRA evictionPolicy
// -----------------------------------------------------------------------------
func TestLRUCacheSetWithDuplicateKeyErrorLRA(t *testing.T) {
	assert := assert.New(t)
	evictionChannel := make(chan EvictedEntry[string, int], 1)
	config := Config[string, int]{
		MaxSize:         2,
		TTL:             time.Minute,
		EvictionChannel: &evictionChannel,
		EvictionPolicy:  LRA,
	}

	cache := New(config)
	err := cache.Set(entry1.Key, entry1.Value)
	assert.NoError(err)
	err = cache.Set(entry2.Key, entry2.Value)
	assert.NoError(err)
	err = cache.Set(entry2.Key, entry2.Value)
	assert.Error(err)

	cachedEntry1 := cache.Get(entry1.Key)
	cachedEntry2 := cache.Get(entry2.Key)

	assert.Equal(entry1.Value, cachedEntry1.Value)
	assert.Equal(entry2.Value, cachedEntry2.Value)
}

func TestLRUCacheSetWithEvictionReasonExpiredLRA(t *testing.T) {
	assert := assert.New(t)
	evictionChannel := make(chan EvictedEntry[string, int], 0)
	ttl := time.Nanosecond
	config := Config[string, int]{
		MaxSize:                   10,
		TTL:                       time.Nanosecond,
		EvictionChannel:           &evictionChannel,
		EvictionPolicy:            LRA,
		GarbageCollectionInterval: ttl,
	}
	cache := New(config)

	var (
		evictedEntry1 EvictedEntry[string, int]
		evictedEntry2 EvictedEntry[string, int]
		evictedEntry3 EvictedEntry[string, int]
		evictedEntry4 EvictedEntry[string, int]
	)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		evictedEntry1 = <-evictionChannel
		evictedEntry2 = <-evictionChannel
		evictedEntry3 = <-evictionChannel
		evictedEntry4 = <-evictionChannel
	}()

	cache.Set(entry1.Key, entry1.Value)
	cache.Set(entry2.Key, entry2.Value)
	cache.Set(entry3.Key, entry3.Value)
	cache.Set(entry4.Key, entry4.Value)
	wg.Wait()

	cachedEntry1 := cache.Get(entry1.Key)
	cachedEntry2 := cache.Get(entry2.Key)
	cachedEntry3 := cache.Get(entry3.Key)
	cachedEntry4 := cache.Get(entry4.Key)

	assert.Nil(cachedEntry1)
	assert.Nil(cachedEntry2)
	assert.Nil(cachedEntry3)
	assert.Nil(cachedEntry4)

	assert.Equal(EvictionReasonExpired, evictedEntry1.Reason)
	assert.Equal(EvictionReasonExpired, evictedEntry2.Reason)
	assert.Equal(EvictionReasonExpired, evictedEntry3.Reason)
	assert.Equal(EvictionReasonExpired, evictedEntry4.Reason)

	assert.Equal(int64(0), evictedEntry1.Counter)
	assert.Equal(int64(0), evictedEntry2.Counter)
	assert.Equal(int64(0), evictedEntry3.Counter)
	assert.Equal(int64(0), evictedEntry4.Counter)

	assert.Equal(entry1.Key, evictedEntry1.Key)
	assert.Equal(entry2.Key, evictedEntry2.Key)
	assert.Equal(entry3.Key, evictedEntry3.Key)
	assert.Equal(entry4.Key, evictedEntry4.Key)
}

func TestLRUCacheKeysWithAllEvictionReasonsLRA(t *testing.T) {
	assert := assert.New(t)

	evictionChannel := make(chan EvictedEntry[string, int], 2)
	ttl := 5 * time.Millisecond
	config := Config[string, int]{
		MaxSize:                   2,
		TTL:                       ttl,
		EvictionChannel:           &evictionChannel,
		EvictionPolicy:            LRA,
		GarbageCollectionInterval: ttl,
	}
	cache := New(config)

	var (
		evictedEntry1 EvictedEntry[string, int]
		evictedEntry2 EvictedEntry[string, int]
		evictedEntry4 EvictedEntry[string, int]
	)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		evictedEntry1 = <-evictionChannel
	}()

	cache.Set(entry1.Key, entry1.Value)
	wg.Wait()
	cache.Set(entry2.Key, entry2.Value)
	cache.Set(entry3.Key, entry3.Value)
	cache.Set(entry4.Key, entry4.Value)
	evictedEntry2 = <-evictionChannel
	cache.Get(entry4.Key)
	cache.Delete(entry4.Key)
	evictedEntry4 = <-evictionChannel

	assert.Equal(entry1.Key, evictedEntry1.Key)
	assert.Equal(entry2.Key, evictedEntry2.Key)
	assert.Equal(entry4.Key, evictedEntry4.Key)

	assert.Equal(EvictionReasonExpired, evictedEntry1.Reason)
	assert.Equal(EvictionReasonDropped, evictedEntry2.Reason)
	assert.Equal(EvictionReasonDeleted, evictedEntry4.Reason)

	assert.Equal(int64(0), evictedEntry1.Counter)
	assert.Equal(int64(0), evictedEntry2.Counter)
	assert.Equal(int64(1), evictedEntry4.Counter)

	keys := cache.Keys()
	assert.Equal(1, len(keys))
	assert.NotContains(keys, entry1.Key, entry2.Key, entry4.Key)
	assert.Contains(keys, entry3.Key)
}

func TestLRUCacheKeysWithAllExpiredLRA(t *testing.T) {
	assert := assert.New(t)

	evictionChannel := make(chan EvictedEntry[string, int], 0)
	ttl := time.Millisecond
	config := Config[string, int]{
		MaxSize:                   10,
		TTL:                       ttl,
		EvictionChannel:           &evictionChannel,
		EvictionPolicy:            LRA,
		GarbageCollectionInterval: ttl,
	}
	cache := New(config)

	var (
		evictedEntry1 EvictedEntry[string, int]
		evictedEntry2 EvictedEntry[string, int]
		evictedEntry3 EvictedEntry[string, int]
		evictedEntry4 EvictedEntry[string, int]
	)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		evictedEntry1 = <-evictionChannel
		evictedEntry2 = <-evictionChannel
		evictedEntry3 = <-evictionChannel
		evictedEntry4 = <-evictionChannel
	}()

	cache.Set(entry1.Key, entry1.Value)
	cache.Set(entry2.Key, entry2.Value)
	cache.Set(entry3.Key, entry3.Value)
	cache.Set(entry4.Key, entry4.Value)
	wg.Wait()

	assert.Equal(EvictionReasonExpired, evictedEntry1.Reason)
	assert.Equal(EvictionReasonExpired, evictedEntry2.Reason)
	assert.Equal(EvictionReasonExpired, evictedEntry3.Reason)
	assert.Equal(EvictionReasonExpired, evictedEntry4.Reason)

	assert.Equal(int64(0), evictedEntry1.Counter)
	assert.Equal(int64(0), evictedEntry2.Counter)
	assert.Equal(int64(0), evictedEntry3.Counter)
	assert.Equal(int64(0), evictedEntry4.Counter)

	keys := cache.Keys()
	assert.Equal(0, len(keys))
}

func TestLRUCacheEntriesWithAllExpiredLRA(t *testing.T) {
	assert := assert.New(t)

	evictionChannel := make(chan EvictedEntry[string, int])
	ttl := time.Nanosecond
	config := Config[string, int]{
		MaxSize:                   10,
		TTL:                       ttl,
		EvictionChannel:           &evictionChannel,
		EvictionPolicy:            LRA,
		GarbageCollectionInterval: ttl,
	}
	cache := New(config)

	var (
		evictedEntry1 EvictedEntry[string, int]
		evictedEntry2 EvictedEntry[string, int]
		evictedEntry3 EvictedEntry[string, int]
		evictedEntry4 EvictedEntry[string, int]
	)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		evictedEntry1 = <-evictionChannel
		evictedEntry2 = <-evictionChannel
		evictedEntry3 = <-evictionChannel
		evictedEntry4 = <-evictionChannel
	}()

	cache.Set(entry1.Key, entry1.Value)
	cache.Set(entry2.Key, entry2.Value)
	cache.Set(entry3.Key, entry3.Value)
	cache.Set(entry4.Key, entry4.Value)
	wg.Wait()

	assert.Equal(entry1.Value, evictedEntry1.Value)
	assert.Equal(entry2.Value, evictedEntry2.Value)
	assert.Equal(entry3.Value, evictedEntry3.Value)
	assert.Equal(entry4.Value, evictedEntry4.Value)

	assert.Equal(EvictionReasonExpired, evictedEntry1.Reason)
	assert.Equal(EvictionReasonExpired, evictedEntry2.Reason)
	assert.Equal(EvictionReasonExpired, evictedEntry3.Reason)
	assert.Equal(EvictionReasonExpired, evictedEntry4.Reason)

	assert.Equal(int64(0), evictedEntry1.Counter)
	assert.Equal(int64(0), evictedEntry2.Counter)
	assert.Equal(int64(0), evictedEntry3.Counter)
	assert.Equal(int64(0), evictedEntry4.Counter)

	entries := cache.Entries()
	assert.Equal(0, len(entries))
}

// Integration test - LRI evictionPolicy
// -----------------------------------------------------------------------------
func TestLRUCacheSetWithEvictionReasonDroppedLRI(t *testing.T) {
	assert := assert.New(t)

	evictionChannel := make(chan EvictedEntry[string, int], 1)
	config := Config[string, int]{
		MaxSize:         2,
		TTL:             time.Minute,
		EvictionChannel: &evictionChannel,
		EvictionPolicy:  LRI,
	}

	cache := New(config)
	cache.Set(entry1.Key, entry1.Value)
	cache.Set(entry2.Key, entry2.Value)
	cache.Set(entry2.Key, entry2.Value)
	cache.Set(entry4.Key, entry4.Value)
	evictedEntry1 := <-evictionChannel
	cache.Set(entry2.Key, entry2.Value)
	cache.Set(entry3.Key, entry3.Value)
	evictedEntry4 := <-evictionChannel

	cachedEntry1 := cache.Get(entry1.Key)
	cachedEntry2 := cache.Get(entry2.Key)
	cachedEntry3 := cache.Get(entry3.Key)
	cachedEntry4 := cache.Get(entry4.Key)

	assert.Nil(cachedEntry1)
	assert.Nil(cachedEntry4)

	assert.Equal(entry1.Key, evictedEntry1.Key)
	assert.Equal(entry4.Key, evictedEntry4.Key)

	assert.Equal(EvictionReasonDropped, evictedEntry1.Reason)
	assert.Equal(EvictionReasonDropped, evictedEntry4.Reason)

	assert.Equal(int64(1), evictedEntry1.Counter)
	assert.Equal(int64(1), evictedEntry4.Counter)

	assert.Equal(entry2.Value, cachedEntry2.Value)
	assert.Equal(int64(3), cachedEntry2.Counter)
	assert.Equal(entry3.Value, cachedEntry3.Value)
	assert.Equal(int64(1), cachedEntry3.Counter)
}

func TestLRUCacheSetWithAllExpiredLRI(t *testing.T) {
	evictionChannel := make(chan EvictedEntry[string, int], 0)
	ttl := time.Millisecond
	config := Config[string, int]{
		MaxSize:                   10,
		TTL:                       ttl,
		EvictionChannel:           &evictionChannel,
		EvictionPolicy:            LRI,
		GarbageCollectionInterval: ttl,
	}
	cache := New(config)

	var (
		evictedEntry1 EvictedEntry[string, int]
		evictedEntry2 EvictedEntry[string, int]
		evictedEntry3 EvictedEntry[string, int]
		evictedEntry4 EvictedEntry[string, int]
	)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		evictedEntry1 = <-evictionChannel
		evictedEntry4 = <-evictionChannel
		evictedEntry2 = <-evictionChannel
		evictedEntry3 = <-evictionChannel
	}()

	cache.Set(entry1.Key, entry1.Value)
	cache.Set(entry2.Key, entry2.Value)
	cache.Set(entry2.Key, entry2.Value)
	cache.Set(entry4.Key, entry4.Value)
	cache.Set(entry2.Key, entry2.Value)
	cache.Set(entry3.Key, entry3.Value)

	wg.Wait()

	cachedEntry1 := cache.Get(entry1.Key)
	cachedEntry2 := cache.Get(entry2.Key)
	cachedEntry3 := cache.Get(entry3.Key)
	cachedEntry4 := cache.Get(entry4.Key)

	assert := assert.New(t)
	assert.Nil(cachedEntry1)
	assert.Nil(cachedEntry2)
	assert.Nil(cachedEntry3)
	assert.Nil(cachedEntry4)

	assert.Equal(EvictionReasonExpired, evictedEntry1.Reason)
	assert.Equal(EvictionReasonExpired, evictedEntry2.Reason)
	assert.Equal(EvictionReasonExpired, evictedEntry3.Reason)
	assert.Equal(EvictionReasonExpired, evictedEntry4.Reason)

	assert.Equal(int64(1), evictedEntry1.Counter)
	assert.Equal(int64(3), evictedEntry2.Counter)
	assert.Equal(int64(1), evictedEntry3.Counter)
	assert.Equal(int64(1), evictedEntry4.Counter)

	assert.Equal(entry1.Key, evictedEntry1.Key)
	assert.Equal(entry2.Key, evictedEntry2.Key)
	assert.Equal(entry3.Key, evictedEntry3.Key)
	assert.Equal(entry4.Key, evictedEntry4.Key)
}

func TestLRUCacheKeysWithOneExpirationLRI(t *testing.T) {
	assert := assert.New(t)

	evictionChannel := make(chan EvictedEntry[string, int], 0)
	ttl := time.Millisecond
	config := Config[string, int]{
		MaxSize:                   10,
		TTL:                       ttl,
		EvictionChannel:           &evictionChannel,
		EvictionPolicy:            LRI,
		GarbageCollectionInterval: ttl,
	}
	cache := New(config)

	cache.Set(entry1.Key, entry1.Value)
	time.Sleep(2 * config.TTL)
	evictedEntry1 := <-evictionChannel
	cache.Set(entry2.Key, entry2.Value)
	cache.Set(entry2.Key, entry2.Value)
	cache.Set(entry4.Key, entry4.Value)
	cache.Set(entry3.Key, entry3.Value)

	assert.Equal(entry1.Key, evictedEntry1.Key)
	assert.Equal(EvictionReasonExpired, evictedEntry1.Reason)
	assert.Equal(int64(1), evictedEntry1.Counter)

	keys := cache.Keys()
	assert.Equal(3, len(keys))
	assert.NotContains(keys, entry1.Key)
	assert.Contains(keys, entry2.Key)
	assert.Contains(keys, entry3.Key)
	assert.Contains(keys, entry4.Key)
}

func TestLRUCacheKeysWithAllExpiredLRI(t *testing.T) {
	assert := assert.New(t)

	evictionChannel := make(chan EvictedEntry[string, int], 0)
	ttl := time.Millisecond
	config := Config[string, int]{
		MaxSize:                   10,
		TTL:                       ttl,
		EvictionChannel:           &evictionChannel,
		EvictionPolicy:            LRI,
		GarbageCollectionInterval: ttl,
	}
	cache := New(config)

	var (
		evictedEntry1 EvictedEntry[string, int]
		evictedEntry2 EvictedEntry[string, int]
		evictedEntry3 EvictedEntry[string, int]
		evictedEntry4 EvictedEntry[string, int]
	)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		evictedEntry1 = <-evictionChannel
		evictedEntry2 = <-evictionChannel
		evictedEntry3 = <-evictionChannel
		evictedEntry4 = <-evictionChannel
	}()

	cache.Set(entry1.Key, entry1.Value)
	cache.Set(entry2.Key, entry2.Value)
	cache.Set(entry2.Key, entry2.Value)
	cache.Set(entry4.Key, entry4.Value)
	cache.Set(entry4.Key, entry4.Value)
	cache.Set(entry4.Key, entry4.Value)
	cache.Set(entry3.Key, entry3.Value)
	cache.Set(entry4.Key, entry4.Value)
	wg.Wait()

	assert.Equal(EvictionReasonExpired, evictedEntry1.Reason)
	assert.Equal(EvictionReasonExpired, evictedEntry2.Reason)
	assert.Equal(EvictionReasonExpired, evictedEntry3.Reason)
	assert.Equal(EvictionReasonExpired, evictedEntry4.Reason)

	assert.Equal(int64(1), evictedEntry1.Counter)
	assert.Equal(int64(2), evictedEntry2.Counter)
	assert.Equal(int64(1), evictedEntry3.Counter)
	assert.Equal(int64(4), evictedEntry4.Counter)

	keys := cache.Keys()
	assert.Equal(0, len(keys))
}

func TestLRUCacheEntriesWithOneExpirationLRI(t *testing.T) {
	assert := assert.New(t)

	evictionChannel := make(chan EvictedEntry[string, int], 0)
	ttl := time.Millisecond
	config := Config[string, int]{
		MaxSize:                   10,
		TTL:                       ttl,
		EvictionChannel:           &evictionChannel,
		EvictionPolicy:            LRI,
		GarbageCollectionInterval: ttl,
	}
	cache := New(config)

	var evictedEntry1 EvictedEntry[string, int]
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		evictedEntry1 = <-evictionChannel
	}()

	cache.Set(entry1.Key, entry1.Value)
	cache.Set(entry1.Key, entry1.Value)
	wg.Wait()
	cache.Set(entry2.Key, entry2.Value)
	cache.Set(entry2.Key, entry2.Value)
	cache.Set(entry4.Key, entry4.Value)
	cache.Set(entry4.Key, entry4.Value)
	cache.Set(entry3.Key, entry3.Value)
	cache.Set(entry3.Key, entry3.Value)

	cachedEntries := cache.Entries()

	assert.NotContains(cachedEntries, entry1.Value)
	assert.Equal(EvictionReasonExpired, evictedEntry1.Reason)
	assert.Equal(entry1.Value, evictedEntry1.Value)
	assert.Equal(int64(2), evictedEntry1.Counter)

	assert.Equal(3, len(cachedEntries))
	entries := map[interface{}]Entry[string, int]{
		entry2.Value: entry2,
		entry3.Value: entry3,
		entry4.Value: entry4,
	}
	for _, cachedEntry := range cachedEntries {
		assert.Equal(entries[cachedEntry.Value].Value, cachedEntry.Value)
		assert.Equal(int64(2), cachedEntry.Counter)
	}
}

func TestLRUCacheEntriesWithAllExpiredLRI(t *testing.T) {
	assert := assert.New(t)

	evictionChannel := make(chan EvictedEntry[string, int], 0)
	ttl := 2 * time.Millisecond
	config := Config[string, int]{
		MaxSize:                   10,
		TTL:                       ttl,
		EvictionChannel:           &evictionChannel,
		EvictionPolicy:            LRI,
		GarbageCollectionInterval: ttl,
	}
	cache := New(config)

	var (
		evictedEntry1 EvictedEntry[string, int]
		evictedEntry2 EvictedEntry[string, int]
		evictedEntry3 EvictedEntry[string, int]
		evictedEntry4 EvictedEntry[string, int]
	)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		evictedEntry1 = <-evictionChannel
		evictedEntry2 = <-evictionChannel
		evictedEntry4 = <-evictionChannel
		evictedEntry3 = <-evictionChannel
	}()

	cache.Set(entry1.Key, entry1.Value)
	cache.Set(entry2.Key, entry2.Value)
	cache.Set(entry2.Key, entry2.Value)
	cache.Set(entry4.Key, entry4.Value)
	cache.Set(entry3.Key, entry3.Value)
	wg.Wait()

	assert.Equal(entry1.Value, evictedEntry1.Value)
	assert.Equal(entry2.Value, evictedEntry2.Value)
	assert.Equal(entry3.Value, evictedEntry3.Value)
	assert.Equal(entry4.Value, evictedEntry4.Value)

	assert.Equal(EvictionReasonExpired, evictedEntry1.Reason)
	assert.Equal(EvictionReasonExpired, evictedEntry2.Reason)
	assert.Equal(EvictionReasonExpired, evictedEntry3.Reason)
	assert.Equal(EvictionReasonExpired, evictedEntry4.Reason)

	assert.Equal(int64(1), evictedEntry1.Counter)
	assert.Equal(int64(2), evictedEntry2.Counter)
	assert.Equal(int64(1), evictedEntry3.Counter)
	assert.Equal(int64(1), evictedEntry4.Counter)

	entries := cache.Entries()
	assert.Equal(0, len(entries))
}

// Race condition test - Both eviction policies
// -----------------------------------------------------------------------------
func TestForRaceConditionsForBothEvictionPolicies(t *testing.T) {
	assert := assert.New(t)
	size := 10000
	for i := range policies {
		config := Config[string, int]{
			MaxSize:        size,
			TTL:            time.Millisecond,
			EvictionPolicy: policies[i],
		}
		cache := New(config)

		var wg sync.WaitGroup
		for x := 0; x < size; x++ {

			wg.Add(1)
			v := strconv.Itoa(x)
			go func() {
				cache.Set(v, 0)
				wg.Done()
			}()
			wg.Add(1)
			go func() {
				cache.Get(v)
				wg.Done()
			}()
			wg.Add(1)
			go func() {
				cache.Delete(v)
				wg.Done()
			}()
			wg.Add(1)
			go func() {
				cache.Has(v)
				wg.Done()
			}()
			wg.Add(1)
			go func() {
				cache.GetState()
				wg.Done()
			}()
			wg.Add(1)
			go func() {
				cache.SetState(State[string, int]{
					EvictionPolicy: policies[i],
					ExtractedAt:    time.Now(),
				})
				wg.Done()
			}()
			wg.Add(1)
			go func() {
				cache.Keys()
				wg.Done()
			}()
			wg.Add(1)
			go func() {
				cache.Entries()
				wg.Done()
			}()
			wg.Add(1)
			go func() {
				cache.Clear()
				wg.Done()
			}()

		}

		wg.Wait()
		assert.True(true)
	}
}

// func TestDestroy(t *testing.T) {
// 	assert := assert.New(t)

// 	evictionChannel := make(chan EvictedEntry[string, int], 2)
// 	ttl := 5 * time.Millisecond
// 	config := Config[string, int]{
// 		MaxSize:                   2,
// 		TTL:                       ttl,
// 		EvictionChannel:           &evictionChannel,
// 		EvictionPolicy:            LRA,
// 		GarbageCollectionInterval: ttl,
// 	}
// 	cache := New(config)

// 	var (
// 		evictedEntry1 EvictedEntry[string, int]
// 		evictedEntry2 EvictedEntry[string, int]
// 		evictedEntry4 EvictedEntry[string, int]
// 	)

// 	var wg sync.WaitGroup
// 	wg.Add(1)
// 	go func() {
// 		defer wg.Done()
// 		evictedEntry1 = <-evictionChannel
// 	}()

// 	cache.Set(entry1.Key, entry1.Value)
// 	wg.Wait()
// 	cache.Set(entry2.Key, entry2.Value)
// 	cache.Set(entry3.Key, entry3.Value)
// 	cache.Set(entry4.Key, entry3.Value)
// 	evictedEntry2 = <-evictionChannel
// 	cache.Get(entry4.Key)
// 	cache.Delete(entry4.Key)
// 	evictedEntry4 = <-evictionChannel

// 	assert.Equal(entry1.Key, evictedEntry1.Key)
// 	assert.Equal(entry2.Key, evictedEntry2.Key)
// 	assert.Equal(entry4.Key, evictedEntry4.Key)

// 	assert.Equal(EvictionReasonExpired, evictedEntry1.Reason)
// 	assert.Equal(EvictionReasonDropped, evictedEntry2.Reason)
// 	assert.Equal(EvictionReasonDeleted, evictedEntry4.Reason)

// 	assert.Equal(int64(0), evictedEntry1.Counter)
// 	assert.Equal(int64(0), evictedEntry2.Counter)
// 	assert.Equal(int64(1), evictedEntry4.Counter)

// 	keys := cache.Keys()
// 	assert.Equal(1, len(keys))
// 	assert.NotContains(keys, entry1.Key, entry2.Key, entry4.Key)
// 	assert.Contains(keys, entry3.Key)

// 	Destroy(&cache)

// 	_, ok := <-evictionChannel

// 	assert.False(ok)
// 	assert.Nil(cache)
// }

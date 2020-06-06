// * tlru <https://github.com/jahnestacado/tlru>
// * Copyright (c) 2020 Ioannis Tzanellis
// * Licensed under the MIT License (MIT).
package tlru

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	entry1   = Entry{Key: "entry1", Value: 1}
	entry2   = Entry{Key: "entry2", Value: 2}
	entry3   = Entry{Key: "entry3", Value: 3}
	entry4   = Entry{Key: "entry4", Value: 4}
	policies = []evictionPolicy{LRA, LRI}
)

// Unit tests
// -----------------------------------------------------------------------------
func TestLRUCacheSetAndGet(t *testing.T) {
	assert := assert.New(t)
	for _, policy := range policies {
		config := Config{
			Size:           1,
			TTL:            time.Minute,
			EvictionPolicy: policy,
		}
		cache := New(config)

		cache.Set(entry1)

		cachedEntry1 := cache.Get(entry1.Key)
		nonExistentEntry := cache.Get("non-existent-key")

		assert.Equal(entry1.Value, cachedEntry1.Value)
		assert.Nil(nonExistentEntry)
	}
}

func TestLRUCacheDelete(t *testing.T) {
	assert := assert.New(t)
	for _, policy := range policies {
		config := Config{
			Size:           10,
			TTL:            time.Minute,
			EvictionPolicy: policy,
		}
		cache := New(config)

		cache.Set(entry1)
		cache.Set(entry2)

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
		config := Config{
			Size:           10,
			TTL:            time.Minute,
			EvictionPolicy: policy,
		}
		cache := New(config)

		cache.Set(entry1)
		cache.Set(entry2)
		cache.Set(entry2)
		cache.Set(entry4)
		cache.Set(entry3)
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
		config := Config{
			Size:           10,
			TTL:            time.Minute,
			EvictionPolicy: policy,
		}
		cache := New(config)

		cache.Set(entry1)
		cache.Set(entry2)
		cache.Set(entry2)
		cache.Set(entry4)
		cache.Set(entry3)

		cachedEntries := cache.Entries()
		assert.Equal(4, len(cachedEntries))
		entries := map[interface{}]Entry{
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

func TestLRUCacheClear(t *testing.T) {
	assert := assert.New(t)
	for _, policy := range policies {
		config := Config{
			Size:           10,
			TTL:            time.Minute,
			EvictionPolicy: policy,
		}
		cache := New(config)

		cache.Set(entry1)
		cache.Set(entry2)

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

		cache.Set(entry2)
		cachedEntry2 = cache.Get(entry2.Key)
		assert.Equal(entry2.Value, cachedEntry2.Value)
	}
}

func TestLRUCacheTTLEvictionDaemon(t *testing.T) {
	assert := assert.New(t)
	for _, policy := range policies {
		evictionChannel := make(chan EvictedEntry, 0)
		config := Config{
			Size:            10,
			TTL:             5 * time.Millisecond,
			EvictionChannel: &evictionChannel,
			EvictionPolicy:  policy,
		}
		cache := New(config)

		var (
			evictedEntry1 EvictedEntry
			evictedEntry2 EvictedEntry
		)

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			evictedEntry1 = <-evictionChannel
			evictedEntry2 = <-evictionChannel
		}()

		expiredEntryTimestamp := time.Date(1900, 2, 1, 12, 30, 0, 0, time.UTC)
		expiredEntry1 := Entry{Key: "expired-1", Value: 1, Timestamp: &expiredEntryTimestamp}
		expiredEntry2 := Entry{Key: "expired-2", Value: 2, Timestamp: &expiredEntryTimestamp}
		nonExpiredEntryTimestamp := time.Now().Add(time.Minute)
		nonExpiredEntry := Entry{Key: "non-expired", Value: 1, Timestamp: &nonExpiredEntryTimestamp}

		cache.Set(nonExpiredEntry)
		cache.Set(expiredEntry1)
		cache.Set(expiredEntry2)
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
		evictionChannel := make(chan EvictedEntry, 1)
		config := Config{
			Size:            10,
			TTL:             time.Minute,
			EvictionChannel: &evictionChannel,
			EvictionPolicy:  policy,
		}
		cache := New(config)

		expiredEntryTimestamp := time.Date(1900, 2, 1, 12, 30, 0, 0, time.UTC)
		expiredEntry := Entry{Key: "expired", Value: 1, Timestamp: &expiredEntryTimestamp}
		nonExpiredEntryTimestamp := time.Now()
		nonExpiredEntry := Entry{Key: "non-expired", Value: 1, Timestamp: &nonExpiredEntryTimestamp}

		cache.Set(entry1)
		cache.Set(expiredEntry)
		cache.Set(nonExpiredEntry)

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
		config := Config{
			Size:           3,
			TTL:            time.Minute,
			EvictionPolicy: policy,
		}
		cache := New(config)

		cache.Set(entry1)
		cache.Set(entry2)

		state := cache.GetState()

		assert.Equal(policy, state.EvictionPolicy)
		assert.Equal(2, len(state.Entries))
		assert.Equal(state.Entries[0].Key, entry2.Key)
		assert.Equal(state.Entries[1].Key, entry1.Key)
	}
}

func TestLRUCacheSetStateError(t *testing.T) {
	assert := assert.New(t)
	state := State{
		EvictionPolicy: LRI,
		ExtractedAt:    time.Now(),
	}

	config := Config{
		Size: 1,
		TTL:  time.Minute,
	}
	cache := New(config)

	err := cache.SetState(state)
	assert.Error(err)
}

func TestLRUCacheSetState(t *testing.T) {
	assert := assert.New(t)
	for _, policy := range policies {
		state := State{
			EvictionPolicy: policy,
			ExtractedAt:    time.Now(),
			Entries: []StateEntry{
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
		evictionChannel := make(chan EvictedEntry, 1)
		config := Config{
			Size:            3,
			TTL:             time.Minute,
			EvictionChannel: &evictionChannel,
			EvictionPolicy:  policy,
		}
		cache := New(config)
		cache.Set(entry4)
		cache.Set(entry3)

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
		config := Config{
			Size:           3,
			TTL:            time.Minute,
			EvictionPolicy: policy,
		}
		cache := New(config)
		cache.Set(entry4)
		cache.Set(entry3)

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

// Integration tests - LRA evictionPolicy
// -----------------------------------------------------------------------------
func TestLRUCacheSetWithDuplicateKeyErrorLRA(t *testing.T) {
	assert := assert.New(t)
	evictionChannel := make(chan EvictedEntry, 1)
	config := Config{
		Size:            2,
		TTL:             time.Minute,
		EvictionChannel: &evictionChannel,
		EvictionPolicy:  LRA,
	}

	cache := New(config)
	err := cache.Set(entry1)
	assert.NoError(err)
	err = cache.Set(entry2)
	assert.NoError(err)
	err = cache.Set(entry2)
	assert.Error(err)

	cachedEntry1 := cache.Get(entry1.Key)
	cachedEntry2 := cache.Get(entry2.Key)

	assert.Equal(entry1.Value, cachedEntry1.Value)
	assert.Equal(entry2.Value, cachedEntry2.Value)
}

func TestLRUCacheSetWithEvictionReasonExpiredLRA(t *testing.T) {
	assert := assert.New(t)
	evictionChannel := make(chan EvictedEntry, 0)
	config := Config{
		Size:            10,
		TTL:             time.Nanosecond,
		EvictionChannel: &evictionChannel,
		EvictionPolicy:  LRA,
	}
	cache := New(config)

	var (
		evictedEntry1 EvictedEntry
		evictedEntry2 EvictedEntry
		evictedEntry3 EvictedEntry
		evictedEntry4 EvictedEntry
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

	cache.Set(entry1)
	cache.Set(entry2)
	cache.Set(entry3)
	cache.Set(entry4)
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

	evictionChannel := make(chan EvictedEntry, 2)
	config := Config{
		Size:            2,
		TTL:             5 * time.Millisecond,
		EvictionChannel: &evictionChannel,
		EvictionPolicy:  LRA,
	}
	cache := New(config)

	var (
		evictedEntry1 EvictedEntry
		evictedEntry2 EvictedEntry
		evictedEntry4 EvictedEntry
	)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		evictedEntry1 = <-evictionChannel
	}()

	cache.Set(entry1)
	wg.Wait()
	cache.Set(entry2)
	cache.Set(entry3)
	cache.Set(entry4)
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

	evictionChannel := make(chan EvictedEntry, 0)
	config := Config{
		Size:            10,
		TTL:             time.Millisecond,
		EvictionChannel: &evictionChannel,
		EvictionPolicy:  LRA,
	}
	cache := New(config)

	var (
		evictedEntry1 EvictedEntry
		evictedEntry2 EvictedEntry
		evictedEntry3 EvictedEntry
		evictedEntry4 EvictedEntry
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

	cache.Set(entry1)
	cache.Set(entry2)
	cache.Set(entry3)
	cache.Set(entry4)
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

	evictionChannel := make(chan EvictedEntry, 0)
	config := Config{
		Size:            10,
		TTL:             time.Nanosecond,
		EvictionChannel: &evictionChannel,
		EvictionPolicy:  LRA,
	}
	cache := New(config)

	var (
		evictedEntry1 EvictedEntry
		evictedEntry2 EvictedEntry
		evictedEntry3 EvictedEntry
		evictedEntry4 EvictedEntry
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

	cache.Set(entry1)
	cache.Set(entry2)
	cache.Set(entry4)
	cache.Set(entry3)
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

	evictionChannel := make(chan EvictedEntry, 1)
	config := Config{
		Size:            2,
		TTL:             time.Minute,
		EvictionChannel: &evictionChannel,
		EvictionPolicy:  LRI,
	}

	cache := New(config)
	cache.Set(entry1)
	cache.Set(entry2)
	cache.Set(entry2)
	cache.Set(entry4)
	evictedEntry1 := <-evictionChannel
	cache.Set(entry2)
	cache.Set(entry3)
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
	evictionChannel := make(chan EvictedEntry, 0)
	config := Config{
		Size:            10,
		TTL:             time.Millisecond,
		EvictionChannel: &evictionChannel,
		EvictionPolicy:  LRI,
	}
	cache := New(config)

	var (
		evictedEntry1 EvictedEntry
		evictedEntry2 EvictedEntry
		evictedEntry3 EvictedEntry
		evictedEntry4 EvictedEntry
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

	cache.Set(entry1)
	cache.Set(entry2)
	cache.Set(entry2)
	cache.Set(entry4)
	cache.Set(entry2)
	cache.Set(entry3)
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

	evictionChannel := make(chan EvictedEntry, 0)
	config := Config{
		Size:            10,
		TTL:             time.Millisecond,
		EvictionChannel: &evictionChannel,
		EvictionPolicy:  LRI,
	}
	cache := New(config)

	cache.Set(entry1)
	time.Sleep(2 * config.TTL)
	evictedEntry1 := <-evictionChannel
	cache.Set(entry2)
	cache.Set(entry2)
	cache.Set(entry4)
	cache.Set(entry3)

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

	evictionChannel := make(chan EvictedEntry, 0)
	config := Config{
		Size:            10,
		TTL:             time.Millisecond,
		EvictionChannel: &evictionChannel,
		EvictionPolicy:  LRI,
	}
	cache := New(config)

	var (
		evictedEntry1 EvictedEntry
		evictedEntry2 EvictedEntry
		evictedEntry3 EvictedEntry
		evictedEntry4 EvictedEntry
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

	cache.Set(entry1)
	cache.Set(entry2)
	cache.Set(entry2)
	cache.Set(entry4)
	cache.Set(entry4)
	cache.Set(entry4)
	cache.Set(entry3)
	cache.Set(entry4)
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

	evictionChannel := make(chan EvictedEntry, 0)
	config := Config{
		Size:            10,
		TTL:             time.Millisecond,
		EvictionChannel: &evictionChannel,
		EvictionPolicy:  LRI,
	}
	cache := New(config)

	var evictedEntry1 EvictedEntry
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		evictedEntry1 = <-evictionChannel
	}()

	cache.Set(entry1)
	cache.Set(entry1)
	wg.Wait()
	cache.Set(entry2)
	cache.Set(entry2)
	cache.Set(entry4)
	cache.Set(entry4)
	cache.Set(entry3)
	cache.Set(entry3)

	cachedEntries := cache.Entries()

	assert.NotContains(cachedEntries, entry1.Value)
	assert.Equal(EvictionReasonExpired, evictedEntry1.Reason)
	assert.Equal(entry1.Value, evictedEntry1.Value)
	assert.Equal(int64(2), evictedEntry1.Counter)

	assert.Equal(3, len(cachedEntries))
	entries := map[interface{}]Entry{
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

	evictionChannel := make(chan EvictedEntry, 0)
	config := Config{
		Size:            10,
		TTL:             2 * time.Millisecond,
		EvictionChannel: &evictionChannel,
		EvictionPolicy:  LRI,
	}
	cache := New(config)

	var (
		evictedEntry1 EvictedEntry
		evictedEntry2 EvictedEntry
		evictedEntry3 EvictedEntry
		evictedEntry4 EvictedEntry
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

	cache.Set(entry1)
	cache.Set(entry2)
	cache.Set(entry2)
	cache.Set(entry4)
	cache.Set(entry3)
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

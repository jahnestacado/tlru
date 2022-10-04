package main

import (
	"fmt"
	"time"

	"github.com/jahnestacado/tlru"
)

var (
	entry1 = tlru.Entry{Key: "entry-1", Value: 1}
	entry2 = tlru.Entry{Key: "entry-2", Value: 2}
	entry3 = tlru.Entry{Key: "entry-3", Value: 3}
	entry4 = tlru.Entry{Key: "entry-4", Value: 4}
	entry5 = tlru.Entry{Key: "entry-5", Value: 5}

	ttl = 2 * time.Millisecond
)

func main() {
	evictionChannel := make(chan tlru.EvictedEntry, 0)
	config := tlru.Config{
		MaxSize:                   3,
		TTL:                       ttl,
		EvictionPolicy:            tlru.LRI,
		EvictionChannel:           &evictionChannel,
		GarbageCollectionInterval: &ttl,
	}
	cache := tlru.New(config)

	go func() {
		for {
			evictedEntry := <-evictionChannel
			fmt.Printf("Entry with key: '%s' has been evicted with reason: %s\n", evictedEntry.Key, evictedEntry.Reason.String())
			// Entry with key: 'entry-1' has been evicted with reason: Expired
			// Entry with key: 'entry-3' has been evicted with reason: Dropped
			// Entry with key: 'entry-5' has been evicted with reason: Deleted
		}
	}()

	cache.Set(entry1)
	time.Sleep(2 * ttl)
	cache.Set(entry2)
	cache.Set(entry3)
	cache.Set(entry2)
	cache.Set(entry4)
	cache.Set(entry4)
	cache.Set(entry5)
	cache.Set(entry4)

	cache.Delete(entry5.Key)

	keys := cache.Keys()
	fmt.Printf("Keys in cache: %v\n", keys)
	// Keys in cache: [entry-2 entry-4] (The key order is not guaranteed)

	cachedEntry2 := cache.Get(entry2.Key)
	fmt.Printf("Entry with key: '%s' has been inserted %d times \n", entry2.Key, cachedEntry2.Counter)
	// Entry with key: 'entry-2' has been inserted 2 times
	cachedEntry4 := cache.Get(entry4.Key)
	fmt.Printf("Entry with '%s' has been inserted %d times \n", entry4.Key, cachedEntry4.Counter)
	// Entry with key: 'entry-4' has been inserted 3 times
}

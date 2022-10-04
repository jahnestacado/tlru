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
		MaxSize:                   2,
		TTL:                       ttl,
		EvictionPolicy:            tlru.LRA,
		EvictionChannel:           &evictionChannel,
		GarbageCollectionInterval: &ttl,
	}
	cache := tlru.New(config)

	go func() {
		for {
			evictedEntry := <-evictionChannel
			fmt.Printf("Entry with key: '%s' has been evicted with reason: %s\n", evictedEntry.Key, evictedEntry.Reason.String())
			// Entry with key: 'entry-1' has been evicted with reason: Expired
			// Entry with key: 'entry-2' has been evicted with reason: Dropped
			// Entry with key: 'entry-4' has been evicted with reason: Deleted
		}
	}()

	cache.Set(entry1)
	time.Sleep(2 * ttl)
	cache.Set(entry2)
	cache.Set(entry3)
	cache.Set(entry4)
	cache.Delete(entry4.Key)
	cache.Set(entry5)

	// Duplicate keys are not allowed in LRA
	err := cache.Set(entry5)
	if err != nil {
		fmt.Println(err.Error())
		// tlru.Set: Key 'entry-5' already exist. Entry replacement is not allowed in LRA EvictionPolicy
	}

	keys := cache.Keys()
	fmt.Printf("Keys in cache: %v\n", keys)
	// Keys in cache: [key5 key3] (The key order is not guaranteed)

	cache.Get(entry3.Key)
	cachedEntry3 := cache.Get(entry3.Key)
	fmt.Printf("Entry with key: '%s' has been accessed %d times \n", entry3.Key, cachedEntry3.Counter)
	// Entry with key: 'entry-3' has been accessed 2 times

	cache.Get(entry5.Key)
	cache.Get(entry5.Key)
	cachedEntry5 := cache.Get(entry5.Key)
	fmt.Printf("Entry with key: '%s' has been accessed %d times \n", entry5.Key, cachedEntry5.Counter)
	// Entry with key: 'entry-5' has been accessed 3 times
}

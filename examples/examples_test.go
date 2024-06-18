package main

import (
	"fmt"
	"time"

	"github.com/jahnestacado/tlru/v3"
)

var (
	entry1 = tlru.Entry[string, int]{Key: "entry-1", Value: 1}
	entry2 = tlru.Entry[string, int]{Key: "entry-2", Value: 2}
	entry3 = tlru.Entry[string, int]{Key: "entry-3", Value: 3}
	entry4 = tlru.Entry[string, int]{Key: "entry-4", Value: 4}
	entry5 = tlru.Entry[string, int]{Key: "entry-5", Value: 5}

	ttl = 2 * time.Millisecond
)

func ExampleLRA() {
	evictionChannel := make(chan tlru.EvictedEntry[string, int])
	config := tlru.Config[string, int]{
		MaxSize:                   2,
		TTL:                       ttl,
		EvictionPolicy:            tlru.LRA,
		EvictionChannel:           &evictionChannel,
		GarbageCollectionInterval: ttl,
	}
	cache := tlru.New(config)

	go func() {
		for {
			evictedEntry := <-evictionChannel
			fmt.Printf("Entry with key: '%s' has been evicted with reason: %s\n", evictedEntry.Key, evictedEntry.Reason.String())
		}
	}()

	cache.Set(entry1.Key, entry1.Value)
	time.Sleep(2 * ttl)
	cache.Set(entry2.Key, entry2.Value)
	cache.Set(entry3.Key, entry3.Value)
	cache.Set(entry4.Key, entry4.Value)
	cache.Delete(entry4.Key)
	cache.Set(entry5.Key, entry5.Value)

	// Duplicate keys are not allowed in LRA
	err := cache.Set(entry5.Key, entry5.Value)
	if err != nil {
		fmt.Println(err.Error())
	}

	fmt.Printf("Number of Keys in cache: %d\n", len(cache.Keys()))
	fmt.Printf("Entry with key: 'entry-3' is in cache: %t\n", cache.Has(entry3.Key))
	fmt.Printf("Entry with key: 'entry-5' is in cache: %t\n", cache.Has(entry5.Key))

	cache.Get(entry3.Key)
	cachedEntry3 := cache.Get(entry3.Key)
	fmt.Printf("Entry with key: '%s' has been accessed %d times\n", entry3.Key, cachedEntry3.Counter)

	cache.Get(entry5.Key)
	cache.Get(entry5.Key)
	cachedEntry5 := cache.Get(entry5.Key)
	fmt.Printf("Entry with key: '%s' has been accessed %d times\n", entry5.Key, cachedEntry5.Counter)

	// Output:
	// Entry with key: 'entry-1' has been evicted with reason: Expired
	// Entry with key: 'entry-2' has been evicted with reason: Dropped
	// Entry with key: 'entry-4' has been evicted with reason: Deleted
	// tlru.Set: Key 'entry-5' already exist. Entry replacement is not allowed in LRA EvictionPolicy
	// Number of Keys in cache: 2
	// Entry with key: 'entry-3' is in cache: true
	// Entry with key: 'entry-5' is in cache: true
	// Entry with key: 'entry-3' has been accessed 2 times
	// Entry with key: 'entry-5' has been accessed 3 times
}

func ExampleLRI() {
	evictionChannel := make(chan tlru.EvictedEntry[string, int])
	config := tlru.Config[string, int]{
		MaxSize:                   3,
		TTL:                       ttl,
		EvictionPolicy:            tlru.LRI,
		EvictionChannel:           &evictionChannel,
		GarbageCollectionInterval: ttl,
	}
	cache := tlru.New(config)

	go func() {
		for {
			evictedEntry := <-evictionChannel
			fmt.Printf("Entry with key: '%s' has been evicted with reason: %s\n", evictedEntry.Key, evictedEntry.Reason.String())
		}
	}()

	cache.Set(entry1.Key, entry1.Value)
	time.Sleep(2 * ttl)
	cache.Set(entry2.Key, entry2.Value)
	cache.Set(entry3.Key, entry3.Value)
	cache.Set(entry2.Key, entry2.Value)
	cache.Set(entry4.Key, entry4.Value)
	cache.Set(entry4.Key, entry4.Value)
	cache.Set(entry5.Key, entry5.Value)
	cache.Set(entry4.Key, entry4.Value)

	cache.Delete(entry5.Key)

	fmt.Printf("Number of Keys in cache: %d\n", len(cache.Keys()))
	fmt.Printf("Entry with key: 'entry-2' is in cache: %t\n", cache.Has(entry2.Key))
	fmt.Printf("Entry with key: 'entry-4' is in cache: %t\n", cache.Has(entry4.Key))

	cachedEntry2 := cache.Get(entry2.Key)
	fmt.Printf("Entry with key: '%s' has been inserted %d times\n", entry2.Key, cachedEntry2.Counter)
	cachedEntry4 := cache.Get(entry4.Key)
	fmt.Printf("Entry with key: '%s' has been inserted %d times\n", entry4.Key, cachedEntry4.Counter)

	// Output:
	// Entry with key: 'entry-1' has been evicted with reason: Expired
	// Entry with key: 'entry-3' has been evicted with reason: Dropped
	// Entry with key: 'entry-5' has been evicted with reason: Deleted
	// Number of Keys in cache: 2
	// Entry with key: 'entry-2' is in cache: true
	// Entry with key: 'entry-4' is in cache: true
	// Entry with key: 'entry-2' has been inserted 2 times
	// Entry with key: 'entry-4' has been inserted 3 times
}

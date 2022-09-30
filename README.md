<p align="center">
  <p align="center">
  <a href="https://github.com/jahnestacado/tlru/actions"><img alt="build"
  src="https://github.com/jahnestacado/tlru/actions/workflows/build.yaml/badge.svg"></a>
    <a href="https://github.com/jahnestacado/tlru/blob/master/LICENSE"><img alt="Software License" src="https://img.shields.io/github/license/mashape/apistatus.svg?style=flat-square"></a>
    <a href="https://goreportcard.com/report/github.com/jahnestacado/tlru"><img alt="Go Report Card" src="https://goreportcard.com/badge/github.com/jahnestacado/tlru?style=flat-square&fuckgithubcache=1"></a>
    <a href="https://godoc.org/github.com/jahnestacado/tlru">
        <img alt="Docs" src="https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square">
    </a>
    <a href="https://codecov.io/gh/jahnestacado/tlru">
  <img src="https://codecov.io/gh/jahnestacado/tlru/branch/master/graph/badge.svg" />
</a>
  </p>
  <p align="center">
    <img width="48.6%" height="51.4%"  src="https://github.com/jahnestacado/tlru/blob/master/resources/gopher.png?raw=true" /img>
  </p>
</p>

# TLRU

### A Time-aware Least Recently Used cache implementation in Go with configurable eviction policy

## Features

- Thread safe
- Entry expiration based on TTL (Time to live)
- LRA (Least Recently Accessed) eviction policy (default)
- LRI (Least Recently Inserted) eviction policy
- Communication of evicted entries via EvictionChannel
- Cache state extraction/ state re-hydration

## API

[Check GoDocs](https://godoc.org/github.com/jahnestacado/tlru#TLRU)

## Eviction Policies

### LRA (Least Recently Accessed) eviction policy

When an entry from the cache is accessed via the `Get` method it is marked as the most recently used entry. This will cause the entrys lifetime in the cache to be prolonged. LRA is the default eviction policy.

- Behavior upon `Get`
  - If the key entry exists then the entry is marked as the most recently used entry
  - If the key entry exists then the entrys Counter is incremented and the LastUsedAt property is updated
  - If an entry for the specified key doesn't exist then it returns nil

* Behavior upon `Set`
  - If the key entry doesn't exist then it inserts it as the most recently used entry with Counter = 0
  - If the key entry already exists then it will return an error
  - If the cache is full (Config.Size) then the least recently accessed entry(the node before the tailNode) will be dropped and an EvictedEntry will be emitted to the EvictionChannel(if present) with EvictionReasonDropped

#### Example

```go
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
		Size:            2,
		TTL:             ttl,
		EvictionPolicy:  tlru.LRA,
		EvictionChannel: &evictionChannel,
	}
	cache := tlru.New(config)

	go func() {
		for evictedEntry := range evictionChannel {
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

```

### LRI (Least Recently Inserted) eviction policy

When an entry is inserted via the `Set` method in the cache, it will be marked as the most recently used entry. This will cause the entrys lifetime in the cache to be prolonged. In contrast to LRA, this eviction policy allows
multiple insertion of entries with the same key.

- Behavior upon `Get`
  - If an entry for the specified key doesn't exist then it returns nil

* Behavior upon `Set`
  - If the key entry doesn't exist then it inserts it as the most recently used entry with Counter = 1
  - If the key entry already exists then it will update the Value, Counter and LastUsedAt properties of
    the existing entry and mark it as the most recently used entry
  - If the cache is full (Config.Size) then the least recently inserted entry(the node before the tailNode)
    will be dropped and an EvictedEntry will be emitted to the EvictionChannel(if present) with EvictionReasonDropped

#### Example

```go
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
		Size:            3,
		TTL:             ttl,
		EvictionPolicy:  tlru.LRI,
		EvictionChannel: &evictionChannel,
	}
	cache := tlru.New(config)

	go func() {
		for evictedEntry := range evictionChannel {
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

```

### Entry timestamp

Upon entry insertion(`Set`) TLRU provides also the option of defining an `Entry.Timestamp`.

If the `Timestamp` property is provided TTL will be checked against that timestamp(until the entry is marked as the most recently used entry again, which will update internally the `LastUsedAt` property).

This is more relevant for the LRI eviction policy which allows multiple insertions of an entry with the same key. A common use case for custom timestamps is
the use of ingestion timestamps

```go
config := tlru.Config{
  Size:            100,
  TTL:             ttl,
  EvictionPolicy:  tlru.LRI,
}
cache := tlru.New(config)

// entries from message broker/ingestion pipeline
entry1 := tlru.Entry{Key:"entry-1", Value: 1, Timestamp: ingestionTimestamp1}
entry2 := tlru.Entry{Key:"entry-2", Value: 2, Timestamp: ingestionTimestamp2}

cache.Set(entry1)
cache.Set(entry2)

```

### Extract/Rehydrate cache state

TLRU provides two methods which allows cache state extraction and state rehydration

```go

cache := tlru.New(config)
cache.Set(entry1)
cache.Set(entry2)

// ...
// State extraction
state := cache.GetState()
// We can now serialize that state and put it in a persistent storage

// ...
// Get state from persistent storage and deserialize it

// State rehydration
err := cache.SetState(state)
// The cache now will set its internal state to the provided state

```

### Run Tests

```sh
make test
```

### Benchmarks

```sh
make bench
```

```
21.6.0 Darwin Kernel Version 21.6.0
go version go1.18.2 darwin/amd64

goos: darwin
goarch: amd64
pkg: github.com/jahnestacado/tlru
cpu: Intel(R) Core(TM) i9-9880H CPU @ 2.30GHz
BenchmarkGet_EmptyCache_LRA-16                        							23577234	    	45.52 ns/op
BenchmarkGet_EmptyCache_LRI-16                        							27169711	    	45.60 ns/op
BenchmarkGet_NonExistingKey_LRA-16                      						48853821	    	27.36 ns/op
BenchmarkGet_NonExistingKey_LRI-16                      						45801547	    	25.17 ns/op
BenchmarkGet_ExistingKey_LRA-16                        	 						 4631190	    	234.6 ns/op
BenchmarkGet_ExistingKey_LRI-16                        	 						 5760564	    	182.7 ns/op
BenchmarkGet_FullCache_1000000_Parallel_LRA-16                						19806835	    	58.29 ns/op
BenchmarkGet_FullCache_1000000_Parallel_LRI-16                						16803621	    	63.58 ns/op
BenchmarkGet_FullCache_1000000_WithTinyTTL_Parallel_LRA-16          	 				 9002697	    	165.8 ns/op
BenchmarkGet_FullCache_1000000_WithTinyTTL_Parallel_LRI-16          	 				 8929759	    	132.2 ns/op
BenchmarkHas_FullCache_1000000_Parallel-16                  						14102200	    	73.63 ns/op
BenchmarkSet_LRA-16                              	 						 1387353	    	829.3 ns/op
BenchmarkSet_LRI-16                              		 					 1379410	    	832.6 ns/op
BenchmarkSet_EvictionChannelAttached_LRA-16                 	 				  	  990680	   	 1197 ns/op
BenchmarkSet_EvictionChannelAttached_LRI-16                  	 				 	 1000000	   	 1162 ns/op
BenchmarkSet_ExistingKey_LRA-16                        	 						 3146720	    	417.8 ns/op
BenchmarkSet_ExistingKey_LRI-16                        	 						 3876129	    	304.1 ns/op
BenchmarkSet_Parallel_LRA-16                         	 						 1403163	    	943.2 ns/op
BenchmarkSet_Parallel_LRI-16                         	 						 1391424	    	940.9 ns/op
BenchmarkDelete_FullCache_1000000_Parallel_LRA-16               	 				 3293047	    	386.2 ns/op
BenchmarkDelete_FullCache_1000000_Parallel_LRI-16               	 			 	 2478804	    	410.1 ns/op
BenchmarkDelete_FullCache_1000000_Parallel_EvictionChannelAttached_LRA-16   				 1696992	    	589.3 ns/op
BenchmarkDelete_FullCache_1000000_Parallel_EvictionChannelAttached_LRI-16   	 			 1825086	    	571.8 ns/op
BenchmarkKeys_EmptyCache_LRA-16                       							33238173	    	36.14 ns/op
BenchmarkKeys_EmptyCache_LRI-16                      						 	33220628	    	35.83 ns/op
BenchmarkKeys_FullCache_1000000_LRA-16                    	  				    	       7	    153565363 ns/op
BenchmarkKeys_FullCache_1000000_LRI-16                    	   					       7	    151832828 ns/op
BenchmarkEntries_EmptyCache_LRA-16                      						36623247	    	38.97 ns/op
BenchmarkEntries_EmptyCache_LRI-16                      						31233552	    	35.47 ns/op
BenchmarkEntries_FullCache_1000000_LRA-16                   	   					       4	    256872533 ns/op
BenchmarkEntries_FullCache_1000000_LRI-16                   	   				   	       4	    260215655 ns/op
```

### License

Copyright (c) 2020 Ioannis Tzanellis  
[Released under the MIT license](https://github.com/jahnestacado/tlru/blob/master/LICENSE)

_Credits to [Ashley McNamara](https://github.com/ashleymcnamara) for the [gopher artwork](https://github.com/ashleymcnamara/gophers)_

_This package is a custom implementation of [TLRU](https://en.wikipedia.org/wiki/Cache_replacement_policies#Time_aware_least_recently_used_(TLRU))_

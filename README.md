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
  - If the cache is full (Config.MaxSize) then the least recently accessed entry(the node before the tailNode) will be dropped and an EvictedEntry will be emitted to the EvictionChannel(if present) with EvictionReasonDropped

#### Example

```go
package main

import (
	"fmt"
	"time"

	"github.com/jahnestacado/tlru"
)

var (
	entry1 = Entry[string, int]{Key: "entry1", Value: 1}
	entry2 = Entry[string, int]{Key: "entry2", Value: 2}
	entry3   = Entry[string, int]{Key: "entry3", Value: 3}
	entry4   = Entry[string, int]{Key: "entry4", Value: 4}
	entry5 = Entry[string, int]{Key: "entry-5", Value: 5}

	ttl = 2 * time.Millisecond
)

func main() {
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
  - If the cache is full (Config.MaxSize) then the least recently inserted entry(the node before the tailNode)
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

```

### Entry timestamp

Upon entry insertion(`Set`) TLRU provides also the option of defining an `Entry.Timestamp`.

If the `Timestamp` property is provided TTL will be checked against that timestamp(until the entry is marked as the most recently used entry again, which will update internally the `LastUsedAt` property).

This is more relevant for the LRI eviction policy which allows multiple insertions of an entry with the same key. A common use case for custom timestamps is
the use of ingestion timestamps

```go
config := tlru.Config[string, int]{
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
BenchmarkGet_EmptyCache_LRA-16                                              	29549968	        39.03 ns/op
BenchmarkGet_EmptyCache_LRI-16                                              	32756817	        39.33 ns/op
BenchmarkGet_NonExistingKey_LRA-16                                          	57142465	        20.85 ns/op
BenchmarkGet_NonExistingKey_LRI-16                                          	58921844	        21.27 ns/op
BenchmarkGet_ExistingKey_LRA-16                                             	16200504	        97.92 ns/op
BenchmarkGet_ExistingKey_LRI-16                                             	14944048	        82.09 ns/op
BenchmarkGet_FullCache_100000_Parallel_LRA-16                               	20769705	        56.68 ns/op
BenchmarkGet_FullCache_100000_Parallel_LRI-16                               	25313176	        46.10 ns/op
BenchmarkGet_FullCache_100000_WithTinyTTL_Parallel_LRA-16                   	24289372	        53.52 ns/op
BenchmarkGet_FullCache_100000_WithTinyTTL_Parallel_LRI-16                   	25021800	        51.01 ns/op
BenchmarkHas_FullCache_100000_Parallel-16                                   	22699959	        45.98 ns/op
BenchmarkSet_LRA-16                                                         	 2975786	       433.9 ns/op
BenchmarkSet_LRI-16                                                         	 2961002	       472.0 ns/op
BenchmarkSet_EvictionChannelAttached_LRA-16                                 	 1503304	       787.7 ns/op
BenchmarkSet_EvictionChannelAttached_LRI-16                                 	 1521462	       789.4 ns/op
BenchmarkSet_ExistingKey_LRA-16                                             	 5351412	       220.3 ns/op
BenchmarkSet_ExistingKey_LRI-16                                             	 6769933	       187.5 ns/op
BenchmarkSet_Parallel_LRA-16                                                	 2195949	       558.7 ns/op
BenchmarkSet_Parallel_LRI-16                                                	 2235992	       532.7 ns/op
BenchmarkDelete_FullCache_100000_Parallel_LRA-16                            	 7426208	       161.9 ns/op
BenchmarkDelete_FullCache_100000_Parallel_LRI-16                            	 7534471	       164.5 ns/op
BenchmarkDelete_FullCache_100000_Parallel_EvictionChannelAttached_LRA-16    	 6218023	       164.5 ns/op
BenchmarkDelete_FullCache_100000_Parallel_EvictionChannelAttached_LRI-16    	 7338567	       169.2 ns/op
BenchmarkKeys_EmptyCache_LRA-16                                             	27554497	        42.56 ns/op
BenchmarkKeys_EmptyCache_LRI-16                                             	27431692	        43.75 ns/op
BenchmarkKeys_FullCache_100000_LRA-16                                       	      96	  11775475 ns/op
BenchmarkKeys_FullCache_100000_LRI-16                                       	      82	  12257743 ns/op
BenchmarkEntries_EmptyCache_LRA-16                                          	26738907	        42.27 ns/op
BenchmarkEntries_EmptyCache_LRI-16                                          	28352583	        42.90 ns/op
BenchmarkEntries_FullCache_100000_LRA-16                                    	      52	  20547679 ns/op
BenchmarkEntries_FullCache_100000_LRI-16                                    	      62	  19623093 ns/op
```

### License

Copyright (c) 2020 Ioannis Tzanellis  
[Released under the MIT license](https://github.com/jahnestacado/tlru/blob/master/LICENSE)

_Credits to [Ashley McNamara](https://github.com/ashleymcnamara) for the [gopher artwork](https://github.com/ashleymcnamara/gophers)_

_This package is a custom implementation of [TLRU](https://en.wikipedia.org/wiki/Cache_replacement_policies#Time_aware_least_recently_used_(TLRU)) concept_

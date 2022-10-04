// * tlru <https://github.com/jahnestacado/tlru>
// * Copyright (c) 2020 Ioannis Tzanellis
// * Licensed under the MIT License (MIT).
package tlru

import (
	"strconv"
	"testing"
	"time"
)

const (
	bigSize   = 1000000
	smallSize = 10
	tinyTTL   = 50 * time.Nanosecond
)

var (
	lraConfig = Config{
		MaxSize:        bigSize,
		TTL:            time.Minute,
		EvictionPolicy: LRA,
	}

	lriConfig = Config{
		MaxSize:        bigSize,
		TTL:            time.Minute,
		EvictionPolicy: LRI,
	}
)

func BenchmarkGet_EmptyCache_LRA(b *testing.B) {
	cache := New(lraConfig)
	for i := 0; i < b.N; i++ {
		cache.Get(strconv.Itoa(i))
	}
}

func BenchmarkGet_EmptyCache_LRI(b *testing.B) {
	cache := New(lriConfig)
	for i := 0; i < b.N; i++ {
		cache.Get(strconv.Itoa(i))
	}
}

func BenchmarkGet_NonExistingKey_LRA(b *testing.B) {
	cache := New(lraConfig)

	for i := 0; i < bigSize; i++ {
		cache.Set(Entry{Key: strconv.Itoa(i), Value: i})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get("non-existent-key")
	}
}

func BenchmarkGet_NonExistingKey_LRI(b *testing.B) {
	cache := New(lriConfig)

	for i := 0; i < bigSize; i++ {
		cache.Set(Entry{Key: strconv.Itoa(i), Value: i})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get("non-existent-key")
	}
}

func BenchmarkGet_ExistingKey_LRA(b *testing.B) {
	cache := New(lraConfig)

	for i := 0; i < bigSize; i++ {
		cache.Set(Entry{Key: strconv.Itoa(i), Value: i})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get(strconv.Itoa(i))
	}
}

func BenchmarkGet_ExistingKey_LRI(b *testing.B) {
	cache := New(lriConfig)

	for i := 0; i < bigSize; i++ {
		cache.Set(Entry{Key: strconv.Itoa(i), Value: i})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get(strconv.Itoa(i))
	}
}

func BenchmarkGet_FullCache_1000000_Parallel_LRA(b *testing.B) {
	cache := New(lraConfig)

	for i := 0; i < bigSize; i++ {
		cache.Set(Entry{Key: strconv.Itoa(i), Value: lraConfig})
	}

	b.ResetTimer()

	i := 0
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i++
			cache.Get(strconv.Itoa(i))
		}
	})
}

func BenchmarkGet_FullCache_1000000_Parallel_LRI(b *testing.B) {
	cache := New(lriConfig)

	for i := 0; i < bigSize; i++ {
		cache.Set(Entry{Key: strconv.Itoa(i), Value: lraConfig})
	}

	b.ResetTimer()

	i := 0
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i++
			cache.Get(strconv.Itoa(i))
		}
	})
}

func BenchmarkGet_FullCache_1000000_WithTinyTTL_Parallel_LRA(b *testing.B) {
	config := Config{
		MaxSize:        bigSize,
		TTL:            tinyTTL,
		EvictionPolicy: LRA,
	}
	cache := New(config)

	for i := 0; i < bigSize; i++ {
		cache.Set(Entry{Key: strconv.Itoa(i), Value: lraConfig})
	}

	b.ResetTimer()

	i := 0
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i++
			cache.Get(strconv.Itoa(i))
		}
	})
}

func BenchmarkGet_FullCache_1000000_WithTinyTTL_Parallel_LRI(b *testing.B) {
	config := Config{
		MaxSize:        bigSize,
		TTL:            tinyTTL,
		EvictionPolicy: LRI,
	}
	cache := New(config)

	for i := 0; i < bigSize; i++ {
		cache.Set(Entry{Key: strconv.Itoa(i), Value: lraConfig})
	}

	b.ResetTimer()

	i := 0
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i++
			cache.Get(strconv.Itoa(i))
		}
	})
}

func BenchmarkHas_FullCache_1000000_Parallel(b *testing.B) {
	cache := New(lraConfig)

	for i := 0; i < bigSize; i++ {
		cache.Set(Entry{Key: strconv.Itoa(i), Value: lraConfig})
	}

	b.ResetTimer()

	i := 0
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i++
			cache.Has(strconv.Itoa(i))
		}
	})
}

func BenchmarkSet_LRA(b *testing.B) {
	cache := New(lraConfig)

	for i := 0; i < b.N; i++ {
		cache.Set(Entry{Key: strconv.Itoa(i), Value: i})
	}
}

func BenchmarkSet_LRI(b *testing.B) {
	cache := New(lriConfig)

	for i := 0; i < b.N; i++ {
		cache.Set(Entry{Key: strconv.Itoa(i), Value: i})
	}
}

func BenchmarkSet_EvictionChannelAttached_LRA(b *testing.B) {
	evictionChannel := make(chan EvictedEntry, 0)
	config := Config{
		MaxSize:         smallSize,
		TTL:             time.Minute,
		EvictionChannel: &evictionChannel,
		EvictionPolicy:  LRA,
	}
	cache := New(config)

	go func() {
		for {
			<-evictionChannel
		}
	}()

	for i := 0; i < b.N; i++ {
		cache.Set(Entry{Key: strconv.Itoa(i), Value: i})
	}
}

func BenchmarkSet_EvictionChannelAttached_LRI(b *testing.B) {
	evictionChannel := make(chan EvictedEntry, 0)
	config := Config{
		MaxSize:         smallSize,
		TTL:             time.Minute,
		EvictionChannel: &evictionChannel,
		EvictionPolicy:  LRI,
	}
	cache := New(config)

	go func() {
		for {
			<-evictionChannel
		}
	}()

	for i := 0; i < b.N; i++ {
		cache.Set(Entry{Key: strconv.Itoa(i), Value: i})
	}
}

func BenchmarkSet_ExistingKey_LRA(b *testing.B) {
	cache := New(lraConfig)
	cache.Set(Entry{Key: "existing-key", Value: bigSize})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cache.Set(Entry{Key: "existing-key", Value: bigSize})
	}
}

func BenchmarkSet_ExistingKey_LRI(b *testing.B) {
	cache := New(lriConfig)
	cache.Set(Entry{Key: "existing-key", Value: bigSize})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cache.Set(Entry{Key: "existing-key", Value: bigSize})
	}
}

func BenchmarkSet_Parallel_LRA(b *testing.B) {
	cache := New(lraConfig)

	i := 0
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i++
			cache.Set(Entry{Key: strconv.Itoa(i), Value: i})
		}
	})
}

func BenchmarkSet_Parallel_LRI(b *testing.B) {
	cache := New(lriConfig)

	i := 0
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i++
			cache.Set(Entry{Key: strconv.Itoa(i), Value: i})
		}
	})
}

func BenchmarkDelete_FullCache_1000000_Parallel_LRA(b *testing.B) {
	cache := New(lraConfig)

	for i := 0; i < bigSize; i++ {
		cache.Set(Entry{Key: strconv.Itoa(i), Value: lraConfig})
	}

	b.ResetTimer()

	i := 0
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i++
			cache.Delete(strconv.Itoa(i))
		}
	})
}

func BenchmarkDelete_FullCache_1000000_Parallel_LRI(b *testing.B) {
	cache := New(lriConfig)

	for i := 0; i < bigSize; i++ {
		cache.Set(Entry{Key: strconv.Itoa(i), Value: lraConfig})
	}

	b.ResetTimer()

	i := 0
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i++
			cache.Delete(strconv.Itoa(i))
		}
	})
}

func BenchmarkDelete_FullCache_1000000_Parallel_EvictionChannelAttached_LRA(b *testing.B) {
	evictionChannel := make(chan EvictedEntry, 0)
	config := Config{
		MaxSize:         bigSize,
		TTL:             time.Minute,
		EvictionChannel: &evictionChannel,
		EvictionPolicy:  LRA,
	}
	cache := New(config)

	for i := 0; i < bigSize; i++ {
		cache.Set(Entry{Key: strconv.Itoa(i), Value: config})
	}

	b.ResetTimer()

	go func() {
		for {
			<-evictionChannel
		}
	}()

	i := 0
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i++
			cache.Delete(strconv.Itoa(i))
		}
	})
}

func BenchmarkDelete_FullCache_1000000_Parallel_EvictionChannelAttached_LRI(b *testing.B) {
	evictionChannel := make(chan EvictedEntry, 0)
	config := Config{
		MaxSize:         bigSize,
		TTL:             time.Minute,
		EvictionChannel: &evictionChannel,
		EvictionPolicy:  LRI,
	}
	cache := New(config)

	for i := 0; i < bigSize; i++ {
		cache.Set(Entry{Key: strconv.Itoa(i), Value: config})
	}

	b.ResetTimer()
	go func() {
		for {
			<-evictionChannel
		}
	}()

	i := 0
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i++
			cache.Delete(strconv.Itoa(i))
		}
	})
}

func BenchmarkKeys_EmptyCache_LRA(b *testing.B) {
	cache := New(lraConfig)
	for i := 0; i < b.N; i++ {
		cache.Keys()
	}
}

func BenchmarkKeys_EmptyCache_LRI(b *testing.B) {
	cache := New(lriConfig)
	for i := 0; i < b.N; i++ {
		cache.Keys()
	}
}

func BenchmarkKeys_FullCache_1000000_LRA(b *testing.B) {
	cache := New(lraConfig)

	for i := 0; i < bigSize; i++ {
		cache.Set(Entry{Key: strconv.Itoa(i), Value: lraConfig})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Keys()
	}
}

func BenchmarkKeys_FullCache_1000000_LRI(b *testing.B) {
	cache := New(lriConfig)

	for i := 0; i < bigSize; i++ {
		cache.Set(Entry{Key: strconv.Itoa(i), Value: lraConfig})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Keys()
	}
}

func BenchmarkEntries_EmptyCache_LRA(b *testing.B) {
	cache := New(lraConfig)
	for i := 0; i < b.N; i++ {
		cache.Entries()
	}
}

func BenchmarkEntries_EmptyCache_LRI(b *testing.B) {
	cache := New(lriConfig)
	for i := 0; i < b.N; i++ {
		cache.Entries()
	}
}

func BenchmarkEntries_FullCache_1000000_LRA(b *testing.B) {
	cache := New(lraConfig)

	for i := 0; i < bigSize; i++ {
		cache.Set(Entry{Key: strconv.Itoa(i), Value: lraConfig})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Entries()
	}
}

func BenchmarkEntries_FullCache_1000000_LRI(b *testing.B) {
	cache := New(lriConfig)

	for i := 0; i < bigSize; i++ {
		cache.Set(Entry{Key: strconv.Itoa(i), Value: lraConfig})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Entries()
	}
}

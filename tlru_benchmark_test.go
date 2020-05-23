package tlru

import (
	"strconv"
	"testing"
	"time"
)

const (
	bigSize   = 1000000
	smallSize = 10
)

var (
	readFlavorConfig = Config{
		Size:   bigSize,
		TTL:    time.Minute,
		Flavor: Read,
	}

	writeFlavorConfig = Config{
		Size:   bigSize,
		TTL:    time.Minute,
		Flavor: Write,
	}
)

func BenchmarkSet_ReadFlavor(b *testing.B) {
	cache := New(readFlavorConfig)

	for i := 0; i < b.N; i++ {
		cache.Set(Entry{Key: strconv.Itoa(i), Value: i})
	}
}

func BenchmarkSet_WriteFlavor(b *testing.B) {
	cache := New(writeFlavorConfig)

	for i := 0; i < b.N; i++ {
		cache.Set(Entry{Key: strconv.Itoa(i), Value: i})
	}
}

func BenchmarkSet_EvictionChannelAttached_ReadFlavor(b *testing.B) {
	evictionChannel := make(chan EvictedEntry, 1)
	config := Config{
		Size:            smallSize,
		TTL:             time.Minute,
		EvictionChannel: &evictionChannel,
		Flavor:          Read,
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

func BenchmarkSet_EvictionChannelAttached_WriteFlavor(b *testing.B) {
	evictionChannel := make(chan EvictedEntry, 1)
	config := Config{
		Size:            smallSize,
		TTL:             time.Minute,
		EvictionChannel: &evictionChannel,
		Flavor:          Write,
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

func BenchmarkSet_ExistingKey_ReadFlavor(b *testing.B) {
	cache := New(readFlavorConfig)
	cache.Set(Entry{Key: "existing-key", Value: bigSize})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cache.Set(Entry{Key: "existing-key", Value: bigSize})
	}
}

func BenchmarkSet_ExistingKey_WriteFlavor(b *testing.B) {
	cache := New(writeFlavorConfig)
	cache.Set(Entry{Key: "existing-key", Value: bigSize})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cache.Set(Entry{Key: "existing-key", Value: bigSize})
	}
}

func BenchmarkGet_EmptyCache_ReadFlavor(b *testing.B) {
	cache := New(readFlavorConfig)
	for i := 0; i < b.N; i++ {
		cache.Get(strconv.Itoa(i))
	}
}

func BenchmarkGet_EmptyCache_WriteFlavor(b *testing.B) {
	cache := New(writeFlavorConfig)
	for i := 0; i < b.N; i++ {
		cache.Get(strconv.Itoa(i))
	}
}

func BenchmarkGet_NonExistingKey_ReadFlavor(b *testing.B) {
	cache := New(readFlavorConfig)

	for i := 0; i < bigSize; i++ {
		cache.Set(Entry{Key: strconv.Itoa(i), Value: i})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get("non-existent-key")
	}
}

func BenchmarkGet_NonExistingKey_WriteFlavor(b *testing.B) {
	cache := New(writeFlavorConfig)

	for i := 0; i < bigSize; i++ {
		cache.Set(Entry{Key: strconv.Itoa(i), Value: i})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get("non-existent-key")
	}
}

func BenchmarkGet_ExistingKey_ReadFlavor(b *testing.B) {
	cache := New(readFlavorConfig)

	for i := 0; i < bigSize; i++ {
		cache.Set(Entry{Key: strconv.Itoa(i), Value: i})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get(strconv.Itoa(i))
	}
}

func BenchmarkGet_ExistingKey_WriteFlavor(b *testing.B) {
	cache := New(writeFlavorConfig)

	for i := 0; i < bigSize; i++ {
		cache.Set(Entry{Key: strconv.Itoa(i), Value: i})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get(strconv.Itoa(i))
	}
}

func BenchmarkSet_Parallel_ReadFlavor(b *testing.B) {
	cache := New(readFlavorConfig)

	i := 0
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i++
			cache.Set(Entry{Key: strconv.Itoa(i), Value: i})
		}
	})
}

func BenchmarkSet_Parallel_WriteFlavor(b *testing.B) {
	cache := New(writeFlavorConfig)

	i := 0
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i++
			cache.Set(Entry{Key: strconv.Itoa(i), Value: i})
		}
	})
}

func BenchmarkGet_FullCache_Parallel_ReadFlavor(b *testing.B) {
	cache := New(readFlavorConfig)

	for i := 0; i < bigSize; i++ {
		cache.Set(Entry{Key: strconv.Itoa(i), Value: readFlavorConfig})
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

func BenchmarkGet_FullCache_Parallel_WriteFlavor(b *testing.B) {
	cache := New(writeFlavorConfig)

	for i := 0; i < bigSize; i++ {
		cache.Set(Entry{Key: strconv.Itoa(i), Value: readFlavorConfig})
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

func BenchmarkDelete_FullCache_Parallel_ReadFlavor(b *testing.B) {
	cache := New(readFlavorConfig)

	for i := 0; i < bigSize; i++ {
		cache.Set(Entry{Key: strconv.Itoa(i), Value: readFlavorConfig})
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

func BenchmarkDelete_FullCache_Parallel_WriteFlavor(b *testing.B) {
	cache := New(writeFlavorConfig)

	for i := 0; i < bigSize; i++ {
		cache.Set(Entry{Key: strconv.Itoa(i), Value: readFlavorConfig})
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

func BenchmarkDelete_FullCache_Parallel_EvictionChannelAttached_ReadFlavor(b *testing.B) {
	evictionChannel := make(chan EvictedEntry, 1)
	config := Config{
		Size:            bigSize,
		TTL:             time.Minute,
		EvictionChannel: &evictionChannel,
		Flavor:          Read,
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

func BenchmarkDelete_FullCache_Parallel_EvictionChannelAttached_WriteFlavor(b *testing.B) {
	evictionChannel := make(chan EvictedEntry, 1)
	config := Config{
		Size:            bigSize,
		TTL:             time.Minute,
		EvictionChannel: &evictionChannel,
		Flavor:          Write,
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

func BenchmarkKeys_EmptyCache_ReadFlavor(b *testing.B) {
	cache := New(readFlavorConfig)
	for i := 0; i < b.N; i++ {
		cache.Keys()
	}
}

func BenchmarkKeys_EmptyCache_WriteFlavor(b *testing.B) {
	cache := New(writeFlavorConfig)
	for i := 0; i < b.N; i++ {
		cache.Keys()
	}
}

func BenchmarkKeys_FullCache_ReadFlavor(b *testing.B) {
	cache := New(readFlavorConfig)

	for i := 0; i < bigSize; i++ {
		cache.Set(Entry{Key: strconv.Itoa(i), Value: readFlavorConfig})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Keys()
	}
}

func BenchmarkKeys_FullCache_WriteFlavor(b *testing.B) {
	cache := New(writeFlavorConfig)

	for i := 0; i < bigSize; i++ {
		cache.Set(Entry{Key: strconv.Itoa(i), Value: readFlavorConfig})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Keys()
	}
}

func BenchmarkEntries_EmptyCache_ReadFlavor(b *testing.B) {
	cache := New(readFlavorConfig)
	for i := 0; i < b.N; i++ {
		cache.Entries()
	}
}

func BenchmarkEntries_EmptyCache_WriteFlavor(b *testing.B) {
	cache := New(writeFlavorConfig)
	for i := 0; i < b.N; i++ {
		cache.Entries()
	}
}

func BenchmarkEntries_FullCache_ReadFlavor(b *testing.B) {
	cache := New(readFlavorConfig)

	for i := 0; i < bigSize; i++ {
		cache.Set(Entry{Key: strconv.Itoa(i), Value: readFlavorConfig})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Entries()
	}
}

func BenchmarkEntries_FullCache_WriteFlavor(b *testing.B) {
	cache := New(writeFlavorConfig)

	for i := 0; i < bigSize; i++ {
		cache.Set(Entry{Key: strconv.Itoa(i), Value: readFlavorConfig})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Entries()
	}
}

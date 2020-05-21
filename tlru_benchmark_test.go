package tlru

import (
	"strconv"
	"testing"
	"time"
)

var (
	c = Config{
		Size: 1000000,
		TTL:  1 * time.Second,
	}
)

func createEntry(k string, v interface{}) Entry {
	return Entry{
		Key:   k,
		Value: v,
	}
}

func BenchmarkInsert(b *testing.B) {
	cache := New(c)
	for i := 0; i < b.N; i++ {
		cache.Set(createEntry(strconv.Itoa(i), i))
	}
}

func BenchmarkParallelInsert(b *testing.B) {
	cache := New(c)

	i := 0
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i++
			cache.Set(createEntry(strconv.Itoa(i), []byte("user:data:cached")))
		}
	})
}

func BenchmarkGet_EmptyCache(b *testing.B) {
	cache := New(c)
	for i := 0; i < b.N; i++ {
		cache.Get(strconv.Itoa(i))
	}
}

func BenchmarkGet_AllMisses(b *testing.B) {
	cache := New(c)
	total := 1000000

	// Insert 1 million items
	for i := 0; i < total; i++ {
		cache.Set(createEntry(strconv.Itoa(i), []int{i}))
	}

	b.ResetTimer()
	// Attempt to get items that are not in cache
	for i := 0; i < b.N; i++ {
		cache.Get(strconv.Itoa(total))
	}
}

func BenchmarkGet_AllHits(b *testing.B) {
	cache := New(c)
	total := 1000000

	// Insert 1 million items
	for i := 0; i < total; i++ {
		cache.Set(createEntry(strconv.Itoa(i), []int{i}))
	}

	b.ResetTimer()
	// Attempt to get items that are  in cache
	for i := 0; i < b.N; i++ {
		cache.Get(strconv.Itoa(i))
	}
}

func BenchmarkParallelGet(b *testing.B) {
	cache := New(c)
	total := 1000000

	// Insert 1 million items
	for i := 0; i < total; i++ {
		cache.Set(createEntry(strconv.Itoa(i), []int{i}))
	}

	b.ResetTimer()

	i := 0
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i++
			cache.Set(createEntry(strconv.Itoa(i), []byte("user:data:cached")))
		}
	})
}

// * tlru <https://github.com/jahnestacado/go-tlru>
// * Copyright (c) 2020 Ioannis Tzanellis
// * Licensed under the MIT License (MIT).
package tlru

import "time"

// TLRU cache public interface
type TLRU interface {
	Get(key string) *CacheEntry
	Set(entry Entry)
	Delete(key string)
	Keys() []string
	Entries() []CacheEntry
	Clear()
	GetState() State
	SetState(state State) error
}

// Config of tlru cache
type Config struct {
	// Max size of cache
	Size int
	// Time to live of cached entries
	TTL time.Duration
	// Channel to listen for evicted entries events
	EvictionChannel *chan EvictedEntry
	// Eviction policy of tlru. Default is LRA
	EvictionPolicy evictionPolicy
}

// Entry to be cached
type Entry struct {
	// The unique identifier of entry
	Key string `json:"key"`
	// The value to be cached
	Value interface{} `json:"value"`
	// Optional field. If provided TTL of entry will be checked against this field
	Timestamp *time.Time `json:"timestamp"`
}

// CacheEntry holds the cached value along with some additional information
type CacheEntry struct {
	// The cached value
	Value interface{} `json:"value"`
	// The number of times this entry has been inserted or accessed based on the EvictionPolicy
	Counter int64 `json:"counter"`
	// The time that this entry was last inserted or accessed based on the EvictionPolicy
	LastUpdatedAt time.Time `json:"last_updated_at"`
	// The time this entry was inserted to the cache
	CreatedAt time.Time `json:"created_at"`
}

// EvictedEntry is an entry that is removed from the cache due to an evictionReason
type EvictedEntry struct {
	// The unique identifier of entry
	Key string `json:"key"`
	// The cached value
	Value interface{} `json:"value"`
	// The number of times this entry has been inserted or accessed based on the EvictionPolicy
	Counter int64 `json:"counter"`
	// The time that this entry was last inserted or accessed based on the EvictionPolicy
	LastUpdatedAt time.Time `json:"last_updated_at"`
	// The time this entry was inserted to the cache
	CreatedAt time.Time `json:"created_at"`
	// The time this entry was evicted from the cache
	EvictedAt time.Time `json:"evicted_at"`
	// The reason this entry has been removed
	Reason evictionReason `json:"reason"`
}

// State is the internal representation of the cache. State can be retrieved/set via the
// GetState/SetState methods
type State struct {
	Entries        []stateEntry
	EvictionPolicy evictionPolicy
	ExtractedAt    time.Time
}

type stateEntry struct {
	Key           string      `json:"key"`
	Value         interface{} `json:"value"`
	Counter       int64       `json:"counter"`
	LastUpdatedAt time.Time   `json:"last_updated_at"`
	CreatedAt     time.Time   `json:"created_at"`
}

type doublyLinkedNode struct {
	Key           string
	Value         interface{}
	Counter       int64
	LastUpdatedAt time.Time
	CreatedAt     time.Time
	Previous      *doublyLinkedNode
	Next          *doublyLinkedNode
}

func (d *doublyLinkedNode) ToCacheEntry() CacheEntry {
	return CacheEntry{
		Value:         d.Value,
		Counter:       d.Counter,
		LastUpdatedAt: d.LastUpdatedAt,
		CreatedAt:     d.CreatedAt,
	}
}
func (d *doublyLinkedNode) ToEvictedEntry(reason evictionReason) EvictedEntry {
	return EvictedEntry{
		Key:           d.Key,
		Value:         d.Value,
		Counter:       d.Counter,
		LastUpdatedAt: d.LastUpdatedAt,
		CreatedAt:     d.CreatedAt,
		EvictedAt:     time.Now().UTC(),
		Reason:        reason,
	}
}

func (d *doublyLinkedNode) ToStateEntry() stateEntry {
	return stateEntry{
		Key:           d.Key,
		Value:         d.Value,
		Counter:       d.Counter,
		LastUpdatedAt: d.LastUpdatedAt,
		CreatedAt:     d.CreatedAt,
	}
}

type evictionReason int

type evictionPolicy int

func (p evictionPolicy) String() string {
	return [...]string{0: "LRA", 1: "LRI"}[p]
}

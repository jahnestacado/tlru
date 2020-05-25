// * tlru <https://github.com/jahnestacado/go-tlru>
// * Copyright (c) 2020 Ioannis Tzanellis
// * Licensed under the MIT License (MIT).

// Package tlru (Time aware Least Recently Used) cache
package tlru

import "time"

// TLRU cache public interface
type TLRU interface {
	// Get retrieves an entry from the cache by key
	// Get behaves differently depending on the EvictionPolicy used
	// * EvictionPolicy.LRA:
	//		- If the key entry exists then the entry is marked as the most recently used entry
	//		- If the key entry exists then the entrys Counter is incremented and the LastUpdatedAt property is updated
	//		- If an entry for the specified key doesn't exist then it returns nil
	//		- If an entry for the specified key exists but is expired it returns nil and an EvictedEntry will be emitted
	// 			to the EvictionChannel(if present) with EvictionReasonExpired
	//		 (if present) with EvictionReasonExpired
	// * EvictionPolicy.LRI:
	//		- If an entry for the specified key doesn't exist then it returns nil
	//		- If an entry for the specified key exists but is expired it returns nil and an EvictedEntry will be emitted
	// 			to the EvictionChannel(if present) with EvictionReasonExpired
	Get(key string) *CacheEntry

	// Set inserts/updates an entry in the cache
	// Set behaves differently depending on the EvictionPolicy used
	// * EvictionPolicy.LRA:
	//		- If the key entry doesn't exist then it inserts it as the most recently used entry
	//		- If the key entry already exists then it will replace the existing entry with the new one
	//			as the most recently used entry and an EvictedEntry will be emitted to the EvictionChannel(if present)
	//			with EvictionReasonReplaced. Replace means that the entry will be dropped and
	//			re-inserted with a new CreatedAt/LastUpdatedAt timestamp and a resetted Counter
	//		- If the cache is full (Config.Size) then the least recently accessed entry(the node before the tailNode)
	//			will be dropped and an EvictedEntry will be emitted to the EvictionChannel(if present) with EvictionReasonDropped
	// * EvictionPolicy.LRI:
	//		- If the key entry doesn't exist then it inserts it as the most recently used entry
	//		- If the key entry already exists then it will update the Value, Counter, LastUpdatedAt, CreatedAt properties of
	//		  the existing entry and mark it as the most recently used entry
	//		- If the cache is full (Config.Size) then the least recently inserted entry(the node before the tailNode)
	//			will be dropped and an EvictedEntry will be emitted to the EvictionChannel(if present) with EvictionReasonDropped
	Set(entry Entry)

	// Delete removes the entry that corresponds to the provided key from the cache
	// An EvictedEntry will be emitted to the EvictionChannel(if present) with EvictionReasonDeleted
	Delete(key string)

	// Keys returns an unordered slice of all available keys in the cache
	// The order of keys is not guaranteed
	// It will also evict expired entries based on the TTL of the cache
	Keys() []string

	// Entries returns an unordered slice of all available entries in the cache
	// The order of entries is not guaranteed
	// It will also evict expired entries based on the TTL of the cache
	Entries() []CacheEntry

	// Clear removes all entries from the cache
	Clear()

	// GetState returns the internal State of the cache
	// This State can be put in persistent storage and rehydrated at a later point
	// via the SetState method
	GetState() State

	// SetState sets the internal State of the cache
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

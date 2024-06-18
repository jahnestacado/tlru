// * tlru <https://github.com/jahnestacado/tlru>
// * Copyright (c) 2020 Ioannis Tzanellis
// * Licensed under the MIT License (MIT).

// Package tlru (Time aware Least Recently Used) cache
package tlru

import (
	"fmt"
	"sync"
	"time"
)

// TLRU cache public interface
type TLRU[K comparable, V any] interface {
	// Get retrieves an entry from the cache by key
	// Get behaves differently depending on the EvictionPolicy used
	// * EvictionPolicy.LRA - (Least Recenty Accessed):
	//		- If the key entry exists then the entry is marked as the
	//		  most recently used entry
	//		- If the key entry exists then the entrys Counter is incremented and the
	//		  LastUsedAt property is updated
	//		- If an entry for the specified key doesn't exist then it returns nil
	// * EvictionPolicy.LRI - (Least Recenty Inserted):
	//		- If an entry for the specified key doesn't exist then it returns nil
	Get(key K) *CacheEntry[K, V]

	// Set inserts/updates an entry in the cache
	// Set behaves differently depending on the EvictionPolicy used
	// * EvictionPolicy.LRA - (Least Recenty Accessed):
	//		- If the key entry doesn't exist then it inserts it as the most
	//		  recently used entry with Counter = 0
	//		- If the key entry already exists then it will return an error
	//		- If the cache is full (Config.MaxSize) then the least recently accessed
	//		  entry(the node before the tailNode) will be dropped and an
	//		  EvictedEntry will be emitted to the EvictionChannel(if present)
	//		  with EvictionReasonDropped
	// * EvictionPolicy.LRI - (Least Recenty Inserted):
	//		- If the key entry doesn't exist then it inserts it as the
	//		  most recently used entry with Counter = 1
	//		- If the key entry already exists then it will update
	//		  the Value, Counter and LastUsedAt properties of
	//		  the existing entry and mark it as the most recently used entry
	//		- If the cache is full (Config.MaxSize) then
	//		  the least recently inserted entry(the node before the tailNode)
	//		  will be dropped and an EvictedEntry will be emitted to
	//		  the EvictionChannel(if present) with EvictionReasonDropped
	Set(key K, value V) error
	SetWithTimestamp(key K, value V, timestamp time.Time) error

	// Delete removes the entry that corresponds to the provided key from cache
	// An EvictedEntry will be emitted to the EvictionChannel(if present)
	// with EvictionReasonDeleted
	Delete(key K)

	// Keys returns an unordered slice of all available keys in the cache
	// The order of keys is not guaranteed
	// It will also evict expired entries based on the TTL of the cache
	Keys() []K

	// Entries returns an unordered slice of all available entries in the cache
	// The order of entries is not guaranteed
	// It will also evict expired entries based on the TTL of the cache
	Entries() []CacheEntry[K, V]

	// Clear removes all entries from the cache
	Clear()

	// GetState returns the internal State of the cache
	// This State can be put in persistent storage and rehydrated at a later point
	// via the SetState method
	GetState() State[K, V]

	// SetState sets the internal State of the cache
	SetState(state State[K, V]) error

	// Has returns true if the provided keys exists in cache otherwise it returns false
	Has(key K) bool
}

// Config of tlru cache
type Config[K comparable, V any] struct {
	// Max size of cache
	MaxSize int
	// Time to live of cached entries
	TTL time.Duration
	// Channel to listen for evicted entries events
	EvictionChannel *chan EvictedEntry[K, V]
	// Eviction policy of tlru. Default is LRA
	EvictionPolicy evictionPolicy
	// GarbageCollectionInterval. If not set it defaults to 10 seconds
	GarbageCollectionInterval time.Duration
}

// Entry to be cached
type Entry[K comparable, V any] struct {
	// The unique identifier of entry
	Key K `json:"key"`
	// The value to be cached
	Value V `json:"value"`
	// Optional field. If provided TTL of entry will be checked against this field
	// Timestamp is in UTC
	Timestamp *time.Time `json:"timestamp"`
}

// CacheEntry holds the cached value along with some additional information
type CacheEntry[K comparable, V any] struct {
	// The unique identifier of entry
	Key K `json:"key"`
	// The cached value
	Value V `json:"value"`
	// The number of times this entry has been inserted or accessed based
	// on the EvictionPolicy
	Counter int64 `json:"counter"`
	// The time that this entry was last inserted or accessed based
	// on the EvictionPolicy
	LastUsedAt time.Time `json:"last_used_at"`
	// The time this entry was inserted to the cache
	CreatedAt time.Time `json:"created_at"`
}

// EvictedEntry is an entry that is removed from the cache due to
// an evictionReason
type EvictedEntry[K comparable, V any] struct {
	CacheEntry[K, V]
	// The time this entry was evicted from the cache
	EvictedAt time.Time `json:"evicted_at"`
	// The reason this entry has been removed
	Reason evictionReason `json:"reason"`
}

// State is the internal representation of the cache.
// State can be retrieved/set via the GetState/SetState methods respectively
type State[K comparable, V any] struct {
	Entries        []StateEntry[K, V] `json:"entries"`
	EvictionPolicy evictionPolicy     `json:"eviction_policy"`
	ExtractedAt    time.Time          `json:"extracted_at"`
}

// StateEntry is a representation of a doublyLinkedNode without pointer references
type StateEntry[K comparable, V any] struct {
	Key        K         `json:"key"`
	Value      V         `json:"value"`
	Counter    int64     `json:"counter"`
	LastUsedAt time.Time `json:"last_used_at"`
	CreatedAt  time.Time `json:"created_at"`
}

const (
	// LRA - Least Recenty Accessed
	LRA evictionPolicy = iota
	// LRI - Least Recenty Inserted
	LRI
)

const (
	// EvictionReasonDropped occurs when cache is full
	EvictionReasonDropped evictionReason = iota
	// EvictionReasonExpired occurs when the TTL of an entry is expired
	EvictionReasonExpired
	// EvictionReasonDeleted occurs when the Delete method is called for a key
	EvictionReasonDeleted
)

const (
	defaultGarbageCollectionInterval = 10 * time.Second
)

type tlru[K comparable, V any] struct {
	sync.RWMutex
	cache                     map[K]*doublyLinkedNode[K, V]
	config                    Config[K, V]
	headNode                  *doublyLinkedNode[K, V]
	tailNode                  *doublyLinkedNode[K, V]
	garbageCollectionInterval time.Duration
	garbageCollectionTimer    *time.Timer
}

// New returns a new instance of TLRU cache
func New[K comparable, V any](config Config[K, V]) TLRU[K, V] {
	var headNodeRef, tailNodeRef K
	headNode := &doublyLinkedNode[K, V]{key: headNodeRef}
	tailNode := &doublyLinkedNode[K, V]{key: tailNodeRef}
	headNode.next = tailNode
	tailNode.previous = headNode

	garbageCollectionInterval := defaultGarbageCollectionInterval
	if config.GarbageCollectionInterval > 0 {
		garbageCollectionInterval = config.GarbageCollectionInterval
	}

	cache := &tlru[K, V]{
		config:                    config,
		cache:                     make(map[K]*doublyLinkedNode[K, V]),
		garbageCollectionInterval: garbageCollectionInterval,
	}

	cache.initializeDoublyLinkedList()

	return cache
}

func (c *tlru[K, V]) Get(key K) *CacheEntry[K, V] {
	c.RLock()

	linkedNode, exists := c.cache[key]
	if !exists {
		c.RUnlock()
		return nil
	}

	if c.config.TTL < time.Since(linkedNode.lastUsedAt) {
		c.RUnlock()
		c.Lock()
		defer c.Unlock()
		c.evictEntry(linkedNode, EvictionReasonExpired)
		return nil
	}

	if c.config.EvictionPolicy == LRA {
		c.RUnlock()
		c.Lock()
		c.handleNodeState(Entry[K, V]{Key: key, Value: linkedNode.value})
		c.Unlock()
		c.RLock()
	}

	defer c.RUnlock()
	cacheEntry := linkedNode.ToCacheEntry()

	return &cacheEntry
}

func (c *tlru[K, V]) Set(key K, value V) error {
	return c.set(key, value, nil)
}

func (c *tlru[K, V]) SetWithTimestamp(key K, value V, timestamp time.Time) error {
	return c.set(key, value, &timestamp)
}

func (c *tlru[K, V]) set(key K, value V, timestamp *time.Time) error {
	defer c.Unlock()
	c.Lock()

	if c.garbageCollectionTimer == nil {
		c.garbageCollectionTimer = time.AfterFunc(c.garbageCollectionInterval, func() {
			c.Lock()
			c.evictExpiredEntries()
			c.Unlock()
		})
	}

	entry := Entry[K, V]{Key: key, Value: value, Timestamp: timestamp}
	_, exists := c.cache[entry.Key]
	if c.config.MaxSize != 0 && !exists && len(c.cache) == c.config.MaxSize {
		c.evictEntry(c.tailNode.previous, EvictionReasonDropped)
	}

	if exists && c.config.EvictionPolicy == LRA {
		return fmt.Errorf("tlru.Set: Key '%+v' already exist. Entry replacement is not allowed in LRA EvictionPolicy", entry.Key)
	}

	c.handleNodeState(entry)

	return nil
}

func (c *tlru[K, V]) Delete(key K) {
	defer c.Unlock()
	c.Lock()

	linkedNode, exists := c.cache[key]
	if exists {
		c.evictEntry(linkedNode, EvictionReasonDeleted)
	}
}

func (c *tlru[K, V]) Keys() []K {
	c.Lock()
	c.evictExpiredEntries()
	c.Unlock()

	defer c.RUnlock()
	c.RLock()

	keys := make([]K, 0, len(c.cache))
	for key := range c.cache {
		keys = append(keys, key)
	}

	return keys
}

func (c *tlru[K, V]) Entries() []CacheEntry[K, V] {
	c.Lock()
	c.evictExpiredEntries()
	c.Unlock()

	defer c.RUnlock()
	c.RLock()

	entries := make([]CacheEntry[K, V], 0, len(c.cache))
	for _, linkedNode := range c.cache {
		entries = append(entries, linkedNode.ToCacheEntry())
	}

	return entries
}

func (c *tlru[K, V]) Clear() {
	defer c.Unlock()
	c.Lock()

	c.clear()

	if c.garbageCollectionTimer != nil {
		c.garbageCollectionTimer.Stop()
		c.garbageCollectionTimer = nil
	}
}

func (c *tlru[K, V]) GetState() State[K, V] {
	defer c.RUnlock()
	c.RLock()

	state := State[K, V]{
		EvictionPolicy: c.config.EvictionPolicy,
		Entries:        make([]StateEntry[K, V], 0, len(c.cache)),
		ExtractedAt:    time.Now().UTC(),
	}

	nextNode := c.headNode.next
	for nextNode != nil && nextNode != c.tailNode {
		state.Entries = append(state.Entries, nextNode.ToStateEntry())
		nextNode = nextNode.next
	}

	return state
}

func (c *tlru[K, V]) SetState(state State[K, V]) error {
	defer c.Unlock()
	c.Lock()
	if state.EvictionPolicy != c.config.EvictionPolicy {
		return fmt.Errorf("tlru.SetState: Incompatible state EvictionPolicy %s", state.EvictionPolicy.String())
	}
	c.clear()

	previousNode := c.headNode
	cache := make(map[K]*doublyLinkedNode[K, V], 0)
	for _, StateEntry := range state.Entries {
		rehydratedNode := &doublyLinkedNode[K, V]{
			key:        StateEntry.Key,
			value:      StateEntry.Value,
			counter:    StateEntry.Counter,
			lastUsedAt: StateEntry.LastUsedAt,
			createdAt:  StateEntry.CreatedAt,
		}
		previousNode.next = rehydratedNode
		rehydratedNode.previous = previousNode
		previousNode = rehydratedNode
		cache[rehydratedNode.key] = rehydratedNode
	}
	previousNode.next = c.tailNode
	c.tailNode.previous = previousNode
	c.cache = cache

	return nil
}

func (c *tlru[K, V]) Has(key K) bool {
	defer c.RUnlock()
	c.RLock()
	_, exists := c.cache[key]

	return exists
}

type doublyLinkedNode[K comparable, V any] struct {
	key        K
	value      V
	counter    int64
	lastUsedAt time.Time
	createdAt  time.Time
	previous   *doublyLinkedNode[K, V]
	next       *doublyLinkedNode[K, V]
}

func (d *doublyLinkedNode[K, V]) ToCacheEntry() CacheEntry[K, V] {
	return CacheEntry[K, V]{
		Key:        d.key,
		Value:      d.value,
		Counter:    d.counter,
		LastUsedAt: d.lastUsedAt,
		CreatedAt:  d.createdAt,
	}
}
func (d *doublyLinkedNode[K, V]) ToEvictedEntry(reason evictionReason) EvictedEntry[K, V] {
	return EvictedEntry[K, V]{
		CacheEntry: CacheEntry[K, V]{
			Key:        d.key,
			Value:      d.value,
			Counter:    d.counter,
			LastUsedAt: d.lastUsedAt,
			CreatedAt:  d.createdAt,
		},
		EvictedAt: time.Now().UTC(),
		Reason:    reason,
	}
}

func (d *doublyLinkedNode[K, V]) ToStateEntry() StateEntry[K, V] {
	return StateEntry[K, V]{
		Key:        d.key,
		Value:      d.value,
		Counter:    d.counter,
		LastUsedAt: d.lastUsedAt,
		CreatedAt:  d.createdAt,
	}
}

type evictionReason int

func (e evictionReason) String() string {
	return [...]string{0: "Dropped", 1: "Expired", 2: "Deleted"}[e]
}

type evictionPolicy int

func (p evictionPolicy) String() string {
	return [...]string{0: "LRA", 1: "LRI"}[p]
}

func (c *tlru[K, V]) clear() {
	if len(c.cache) > 0 {
		c.cache = make(map[K]*doublyLinkedNode[K, V])
		c.initializeDoublyLinkedList()
	}
}

func (c *tlru[K, V]) initializeDoublyLinkedList() {
	var headNodeRef, tailNodeRef K
	headNode := &doublyLinkedNode[K, V]{key: headNodeRef}
	tailNode := &doublyLinkedNode[K, V]{key: tailNodeRef}
	headNode.next = tailNode
	tailNode.previous = headNode
	c.headNode = headNode
	c.tailNode = tailNode
}

func (c *tlru[K, V]) handleNodeState(e Entry[K, V]) {
	var counter int64
	if c.config.EvictionPolicy == LRI {
		counter++
	}

	lastUsedAt := time.Now().UTC()
	if e.Timestamp != nil {
		lastUsedAt = *e.Timestamp
	}
	linkedNode, exists := c.cache[e.Key]
	if exists {
		if c.config.TTL >= time.Since(linkedNode.lastUsedAt) {
			linkedNode.counter++
		}
		linkedNode.lastUsedAt = lastUsedAt

		// Re-wire siblings of linkedNode
		linkedNode.next.previous = linkedNode.previous
		linkedNode.previous.next = linkedNode.next
	} else {
		linkedNode = &doublyLinkedNode[K, V]{
			key:        e.Key,
			value:      e.Value,
			counter:    counter,
			lastUsedAt: lastUsedAt,
			previous:   c.headNode,
			next:       c.headNode.next,
			createdAt:  time.Now().UTC(),
		}

		c.cache[e.Key] = linkedNode
	}

	// Re-wire headNode
	linkedNode.previous = c.headNode
	linkedNode.next = c.headNode.next
	c.headNode.next.previous = linkedNode
	c.headNode.next = linkedNode
}

func (c *tlru[K, V]) evictEntry(evictedNode *doublyLinkedNode[K, V], reason evictionReason) {
	evictedNode.previous.next = evictedNode.next
	evictedNode.next.previous = evictedNode.previous
	delete(c.cache, evictedNode.key)

	if c.config.EvictionChannel != nil {
		*c.config.EvictionChannel <- evictedNode.ToEvictedEntry(reason)
	}
}

func (c *tlru[K, V]) evictExpiredEntries() {
	previousNode := c.tailNode.previous
	for previousNode != nil && previousNode != c.headNode {
		if c.config.TTL < time.Since(previousNode.lastUsedAt) {
			c.evictEntry(previousNode, EvictionReasonExpired)
		}
		previousNode = previousNode.previous
	}
}

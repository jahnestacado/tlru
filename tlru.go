// * tlru <https://github.com/jahnestacado/tlru>
// * Copyright (c) 2020 Ioannis Tzanellis
// * Licensed under the MIT License (MIT).

// Package tlru (Time aware Least Recently Used) cache
package tlru

import (
	"fmt"
	"sync"
	"time"
	"context"
)

// TLRU cache public interface
type TLRU interface {
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
	Get(key string) *CacheEntry

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
	Set(entry Entry) error

	// Delete removes the entry that corresponds to the provided key from cache
	// An EvictedEntry will be emitted to the EvictionChannel(if present)
	// with EvictionReasonDeleted
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

	// Has returns true if the provided keys exists in cache otherwise it returns false
	Has(key string) bool

	// Start starts the garabage collection daemon
	Start(context.Context)

	// Shutdown stops the garabage collection daemon
	Shutdown()
}

// Config of tlru cache
type Config struct {
	// Max size of cache
	MaxSize int
	// Time to live of cached entries
	TTL time.Duration
	// Channel to listen for evicted entries events
	EvictionChannel chan EvictedEntry
	// Eviction policy of tlru. Default is LRA
	EvictionPolicy evictionPolicy
	// GarbageCollectionInterval. If not set it defaults to 10 seconds
	GarbageCollectionInterval time.Duration
}

// Entry to be cached
type Entry struct {
	// The unique identifier of entry
	Key string `json:"key"`
	// The value to be cached
	Value interface{} `json:"value"`
	// Optional field. If provided TTL of entry will be checked against this field
	// Timestamp is in UTC
	Timestamp *time.Time `json:"timestamp"`
}

// CacheEntry holds the cached value along with some additional information
type CacheEntry struct {
	// The cached value
	Value interface{} `json:"value"`
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
type EvictedEntry struct {
	// The unique identifier of entry
	Key string `json:"key"`
	// The cached value
	Value interface{} `json:"value"`
	// The number of times this entry has been inserted or accessed based on
	// the EvictionPolicy
	Counter int64 `json:"counter"`
	// The time that this entry was last inserted or accessed based on
	// the EvictionPolicy
	LastUsedAt time.Time `json:"last_used_at"`
	// The time this entry was inserted to the cache
	CreatedAt time.Time `json:"created_at"`
	// The time this entry was evicted from the cache
	EvictedAt time.Time `json:"evicted_at"`
	// The reason this entry has been removed
	Reason evictionReason `json:"reason"`
}

// State is the internal representation of the cache.
// State can be retrieved/set via the GetState/SetState methods respectively
type State struct {
	Entries        []StateEntry   `json:"entries"`
	EvictionPolicy evictionPolicy `json:"eviction_policy"`
	ExtractedAt    time.Time      `json:"extracted_at"`
}

// StateEntry is a representation of a doublyLinkedNode without pointer references
type StateEntry struct {
	Key        string      `json:"key"`
	Value      interface{} `json:"value"`
	Counter    int64       `json:"counter"`
	LastUsedAt time.Time   `json:"last_used_at"`
	CreatedAt  time.Time   `json:"created_at"`
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

type tlru struct {
	sync.RWMutex
	cache                     map[string]*doublyLinkedNode
	config                    Config
	headNode                  *doublyLinkedNode
	tailNode                  *doublyLinkedNode
	garbageCollectionInterval time.Duration
	ctx                       context.Context
	cancelFunc                context.CancelFunc
}

// New returns a new instance of TLRU cache
func New(config Config) TLRU {
	headNode := &doublyLinkedNode{key: "head_node"}
	tailNode := &doublyLinkedNode{key: "tail_node"}
	headNode.next = tailNode
	tailNode.previous = headNode

	garbageCollectionInterval := defaultGarbageCollectionInterval
	if config.GarbageCollectionInterval != 0 {
		garbageCollectionInterval = config.GarbageCollectionInterval
	}

	cache := &tlru{
		config:                    config,
		cache:                     make(map[string]*doublyLinkedNode, 0),
		garbageCollectionInterval: garbageCollectionInterval,
	}

	cache.initializeDoublyLinkedList()

	return cache
}

// Start TTL eviction daemon
// Use Shutdown to stop daemon
func Start(ctx context.Context) {
	cache.ctx, cache.cancelFunc = context.WithCancel(config.Context)
	go cache.startTTLEvictionDaemon()
}

func (c *tlru) Get(key string) *CacheEntry {
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
		c.handleNodeState(Entry{Key: key, Value: linkedNode.value})
		c.Unlock()
		c.RLock()
	}

	defer c.RUnlock()
	cacheEntry := linkedNode.ToCacheEntry()

	return &cacheEntry
}

func (c *tlru) Set(entry Entry) error {
	defer c.Unlock()
	c.Lock()

	_, exists := c.cache[entry.Key]
	if c.config.MaxSize != 0 && !exists && len(c.cache) == c.config.MaxSize {
		c.evictEntry(c.tailNode.previous, EvictionReasonDropped)
	}

	if exists && c.config.EvictionPolicy == LRA {
		return fmt.Errorf("tlru.Set: Key '%s' already exist. Entry replacement is not allowed in LRA EvictionPolicy", entry.Key)
	}

	c.handleNodeState(entry)

	return nil
}

func (c *tlru) Delete(key string) {
	defer c.Unlock()
	c.Lock()

	linkedNode, exists := c.cache[key]
	if exists {
		c.evictEntry(linkedNode, EvictionReasonDeleted)
	}
}

func (c *tlru) Keys() []string {
	c.Lock()
	c.evictExpiredEntries()
	c.Unlock()

	defer c.RUnlock()
	c.RLock()

	keys := make([]string, 0, len(c.cache))
	for key := range c.cache {
		keys = append(keys, key)
	}

	return keys
}

func (c *tlru) Entries() []CacheEntry {
	c.Lock()
	c.evictExpiredEntries()
	c.Unlock()

	defer c.RUnlock()
	c.RLock()

	entries := make([]CacheEntry, 0, len(c.cache))
	for _, linkedNode := range c.cache {
		entries = append(entries, linkedNode.ToCacheEntry())
	}

	return entries
}

func (c *tlru) Clear() {
	defer c.Unlock()
	c.Lock()

	c.clear()
}

func (c *tlru) GetState() State {
	defer c.RUnlock()
	c.RLock()

	state := State{
		EvictionPolicy: c.config.EvictionPolicy,
		Entries:        make([]StateEntry, 0, len(c.cache)),
		ExtractedAt:    time.Now().UTC(),
	}

	nextNode := c.headNode.next
	for nextNode != nil && nextNode != c.tailNode {
		state.Entries = append(state.Entries, nextNode.ToStateEntry())
		nextNode = nextNode.next
	}

	return state
}

func (c *tlru) SetState(state State) error {
	defer c.Unlock()
	c.Lock()
	if state.EvictionPolicy != c.config.EvictionPolicy {
		return fmt.Errorf("tlru.SetState: Incompatible state EvictionPolicy %s", state.EvictionPolicy.String())
	}
	c.clear()

	previousNode := c.headNode
	cache := make(map[string]*doublyLinkedNode, 0)
	for _, StateEntry := range state.Entries {
		rehydratedNode := &doublyLinkedNode{
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

func (c *tlru) Has(key string) bool {
	defer c.RUnlock()
	c.RLock()
	_, exists := c.cache[key]

	return exists
}

type doublyLinkedNode struct {
	key        string
	value      interface{}
	counter    int64
	lastUsedAt time.Time
	createdAt  time.Time
	previous   *doublyLinkedNode
	next       *doublyLinkedNode
}

func (d *doublyLinkedNode) ToCacheEntry() CacheEntry {
	return CacheEntry{
		Value:      d.value,
		Counter:    d.counter,
		LastUsedAt: d.lastUsedAt,
		CreatedAt:  d.createdAt,
	}
}
func (d *doublyLinkedNode) ToEvictedEntry(reason evictionReason) EvictedEntry {
	return EvictedEntry{
		Key:        d.key,
		Value:      d.value,
		Counter:    d.counter,
		LastUsedAt: d.lastUsedAt,
		CreatedAt:  d.createdAt,
		EvictedAt:  time.Now().UTC(),
		Reason:     reason,
	}
}

func (d *doublyLinkedNode) ToStateEntry() StateEntry {
	return StateEntry{
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

func (c *tlru) clear() {
	if len(c.cache) > 0 {
		c.cache = make(map[string]*doublyLinkedNode, 0)
		c.initializeDoublyLinkedList()
	}
}

func (c *tlru) initializeDoublyLinkedList() {
	headNode := &doublyLinkedNode{key: "head_node"}
	tailNode := &doublyLinkedNode{key: "tail_node"}
	headNode.next = tailNode
	tailNode.previous = headNode
	c.headNode = headNode
	c.tailNode = tailNode
}

func (c *tlru) handleNodeState(entry Entry) {
	var counter int64
	if c.config.EvictionPolicy == LRI {
		counter++
	}

	lastUsedAt := time.Now().UTC()
	if entry.Timestamp != nil {
		lastUsedAt = *entry.Timestamp
	}
	linkedNode, exists := c.cache[entry.Key]
	if exists {
		if c.config.TTL >= time.Since(linkedNode.lastUsedAt) {
			linkedNode.counter++
		}
		linkedNode.lastUsedAt = lastUsedAt

		// Re-wire siblings of linkedNode
		linkedNode.next.previous = linkedNode.previous
		linkedNode.previous.next = linkedNode.next
	} else {
		linkedNode = &doublyLinkedNode{
			key:        entry.Key,
			value:      entry.Value,
			counter:    counter,
			lastUsedAt: lastUsedAt,
			previous:   c.headNode,
			next:       c.headNode.next,
			createdAt:  time.Now().UTC(),
		}

		c.cache[entry.Key] = linkedNode
	}

	// Re-wire headNode
	linkedNode.previous = c.headNode
	linkedNode.next = c.headNode.next
	c.headNode.next.previous = linkedNode
	c.headNode.next = linkedNode
}

func (c *tlru) evictEntry(evictedNode *doublyLinkedNode, reason evictionReason) {
	evictedNode.previous.next = evictedNode.next
	evictedNode.next.previous = evictedNode.previous
	delete(c.cache, evictedNode.key)

	if c.config.EvictionChannel != nil {
		*c.config.EvictionChannel <- evictedNode.ToEvictedEntry(reason)
	}
}

func (c *tlru) evictExpiredEntries() {
	previousNode := c.tailNode.previous
	for previousNode != nil && previousNode != c.headNode {
		if c.config.TTL < time.Since(previousNode.lastUsedAt) {
			c.evictEntry(previousNode, EvictionReasonExpired)
		}
		previousNode = previousNode.previous
	}
}

func (c *tlru) startTTLEvictionDaemon() { 
	for {
		timer := time.NewTimer(c.garbageCollectionInterval)
		select {
		case <-c.ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			close(*c.config.EvictionChannel)
			return
		case <-timer.C:
			c.Lock()
			c.evictExpiredEntries()
			c.Unlock()
		}
	}
}

func (c *tlru) Shutdown() {
	c.cancelFunc()
}

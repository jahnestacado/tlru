// * tlru <https://github.com/jahnestacado/go-tlru>
// * Copyright (c) 2020 Ioannis Tzanellis
// * Licensed under the MIT License (MIT).
package tlru

import (
	"fmt"
	"sync"
	"time"
)

const (
	// Least Recenty Accessed
	LRA evictionPolicy = iota
	// Least Recenty Inserted
	LRI
)

const (
	// EvictionReasonDropped occurs when cache is full
	EvictionReasonDropped evictionReason = iota
	// EvictionReasonExpired occurs when the TTL of an entry is expired
	EvictionReasonExpired
	// EvictionReasonReplaced occurs when the corresponding CacheEntry of a key is replaced.
	// Can happen only when the EvictionPolicy is LRA
	EvictionReasonReplaced
	// EvictionReasonDeleted occurs when the Delete method is called for a key
	EvictionReasonDeleted
)

type tlru struct {
	sync.RWMutex
	cache          map[string]*doublyLinkedNode
	config         Config
	headNode       *doublyLinkedNode
	tailNode       *doublyLinkedNode
	evictionPolicy evictionPolicy
}

// New returns a new instance of TLRU cache
func New(config Config) TLRU {
	headNode := &doublyLinkedNode{Key: "head_node"}
	tailNode := &doublyLinkedNode{Key: "tail_node"}
	headNode.Next = tailNode
	tailNode.Previous = headNode

	cache := &tlru{
		config: config,
		cache:  make(map[string]*doublyLinkedNode, 0),
	}

	cache.initializeDoublyLinkedList()

	return cache
}

// Get retrieves an entry from the cache by key
// Get behaves differently depending on the EvictionPolicy used
// * EvictionPolicy.LRA:
//		- If the key entry exists then the entry is marked as the most recently used entry
//		- If the key entry exists then the entrys Counter is incremented and the LastUpdatedAt property is updated
//		- If an entry for the specified key doesn't exist then it returns nil
//		- If an entry for the specified key exists but is expired it returns nil and an EvictedEntry will be emmited
// 			to the EvictionChannel(if present) with EvictionReasonExpired
//		 (if present) with EvictionReasonExpired
// * EvictionPolicy.LRI:
//		- If an entry for the specified key doesn't exist then it returns nil
//		- If an entry for the specified key exists but is expired it returns nil and an EvictedEntry will be emmited
// 			to the EvictionChannel(if present) with EvictionReasonExpired
func (c *tlru) Get(key string) *CacheEntry {
	defer c.Unlock()
	c.Lock()

	linkedNode, exists := c.cache[key]
	if !exists {
		return nil
	}

	if c.config.TTL < time.Since(linkedNode.LastUpdatedAt) {
		c.evictEntry(linkedNode, EvictionReasonExpired)
		return nil
	}

	if c.config.EvictionPolicy == LRA {
		c.setMRUNode(Entry{Key: key, Value: linkedNode.Value})
	}

	cacheEntry := linkedNode.ToCacheEntry()

	return &cacheEntry
}

// Set inserts/updates an entry in the cache
// Set behaves differently depending on the EvictionPolicy used
// * EvictionPolicy.LRA:
//		- If the key entry doesn't exist then it inserts it as the most recently used entry
//		- If the key entry already exists then it will replace the existing entry with the new one
//			as the most recently used entry and an EvictedEntry will be emmited to the EvictionChannel(if present)
//			with EvictionReasonReplaced. Replace means that the entry will be dropped and
//			re-inserted with a new CreatedAt/LastUpdatedAt timestamp and a resetted Counter
//		- If the cache is full (Config.Size) then the least recently accessed entry(the node before the tailNode)
//			will be dropped and an EvictedEntry will be emmited to the EvictionChannel(if present) with EvictionReasonDropped
// * EvictionPolicy.LRI:
//		- If the key entry doesn't exist then it inserts it as the most recently used entry
//		- If the key entry already exists then it will update the Value, Counter, LastUpdatedAt, CreatedAt properties of
//		  the existing entry and mark it as the most recently used entry
//		- If the cache is full (Config.Size) then the least recently inserted entry(the node before the tailNode)
//			will be dropped and an EvictedEntry will be emmited to the EvictionChannel(if present) with EvictionReasonDropped
func (c *tlru) Set(entry Entry) {
	defer c.Unlock()
	c.Lock()

	linkedEntry, exists := c.cache[entry.Key]
	if exists && c.config.EvictionPolicy == LRA {
		c.evictEntry(linkedEntry, EvictionReasonReplaced)
	}

	if !exists && len(c.cache) == c.config.Size {
		c.evictEntry(c.tailNode.Previous, EvictionReasonDropped)
	}

	c.setMRUNode(entry)
}

// Delete removes the entry that corresponds to the provided key from the cache
// An EvictedEntry will be emmited to the EvictionChannel(if present) with EvictionReasonDeleted
func (c *tlru) Delete(key string) {
	defer c.Unlock()
	c.Lock()

	linkedNode, exists := c.cache[key]
	if exists {
		c.evictEntry(linkedNode, EvictionReasonDeleted)
	}
}

// Keys returns an unordered slice of all available keys in the cache
// The order of keys is not guaranteed
// It will also evict expired entries based on the TTL of the cache
func (c *tlru) Keys() []string {
	defer c.Unlock()
	c.Lock()
	c.evictExpiredEntries()

	keys := make([]string, 0, len(c.cache))
	for key := range c.cache {
		keys = append(keys, key)
	}

	return keys
}

// Entries returns an unordered slice of all available entries in the cache
// The order of entries is not guaranteed
// It will also evict expired entries based on the TTL of the cache
func (c *tlru) Entries() []CacheEntry {
	defer c.Unlock()
	c.Lock()
	c.evictExpiredEntries()

	entries := make([]CacheEntry, 0, len(c.cache))
	for _, linkedNode := range c.cache {
		entries = append(entries, linkedNode.ToCacheEntry())
	}

	return entries
}

// Clear removes all entries from the cache
func (c *tlru) Clear() {
	defer c.Unlock()
	c.Lock()

	c.clear()
}

// GetState returns the internal State of the cache
// This State can be put in persistent storage and rehydrated at a later point
// via the SetState method
func (c *tlru) GetState() State {
	defer c.RUnlock()
	c.RLock()

	state := State{
		EvictionPolicy: c.config.EvictionPolicy,
		Entries:        make([]stateEntry, 0, len(c.cache)),
		ExtractedAt:    time.Now().UTC(),
	}

	nextNode := c.headNode.Next
	for nextNode != nil && nextNode != c.tailNode {
		state.Entries = append(state.Entries, nextNode.ToStateEntry())
		nextNode = nextNode.Next
	}

	return state
}

// SetState sets the internal State of the cache
func (c *tlru) SetState(state State) error {
	defer c.Unlock()
	c.Lock()
	if state.EvictionPolicy != c.config.EvictionPolicy {
		return fmt.Errorf("tlru.SetState: Incompatible state EvictionPolicy %s", state.EvictionPolicy.String())
	}
	c.clear()

	previousNode := c.headNode
	cache := make(map[string]*doublyLinkedNode, 0)
	for _, stateEntry := range state.Entries {
		rehydratedNode := &doublyLinkedNode{
			Key:           stateEntry.Key,
			Value:         stateEntry.Value,
			Counter:       stateEntry.Counter,
			LastUpdatedAt: stateEntry.LastUpdatedAt,
			CreatedAt:     stateEntry.CreatedAt,
		}
		previousNode.Next = rehydratedNode
		rehydratedNode.Previous = previousNode
		previousNode = rehydratedNode
		cache[rehydratedNode.Key] = rehydratedNode
	}
	previousNode.Next = c.tailNode
	c.tailNode.Previous = previousNode
	c.cache = cache

	return nil
}

func (c *tlru) clear() {
	if len(c.cache) > 0 {
		c.cache = make(map[string]*doublyLinkedNode, 0)
		c.initializeDoublyLinkedList()
	}
}

func (c *tlru) initializeDoublyLinkedList() {
	headNode := &doublyLinkedNode{Key: "head_node"}
	tailNode := &doublyLinkedNode{Key: "tail_node"}
	headNode.Next = tailNode
	tailNode.Previous = headNode
	c.headNode = headNode
	c.tailNode = tailNode
}

func (c *tlru) setMRUNode(entry Entry) {
	var counter int64
	if c.config.EvictionPolicy == LRI {
		counter++
	}

	lastUpdatedAt := time.Now().UTC()
	if entry.Timestamp != nil {
		lastUpdatedAt = *entry.Timestamp
	}
	linkedNode, exists := c.cache[entry.Key]
	if exists {
		if c.config.TTL >= time.Since(linkedNode.LastUpdatedAt) {
			linkedNode.Counter++
		}
		linkedNode.LastUpdatedAt = lastUpdatedAt

		// Re-wire siblings of linkedNode
		linkedNode.Next.Previous = linkedNode.Previous
		linkedNode.Previous.Next = linkedNode.Next
	} else {
		linkedNode = &doublyLinkedNode{
			Key:           entry.Key,
			Value:         entry.Value,
			Counter:       counter,
			LastUpdatedAt: lastUpdatedAt,
			Previous:      c.headNode,
			Next:          c.headNode.Next,
			CreatedAt:     time.Now().UTC(),
		}

		c.cache[entry.Key] = linkedNode
	}

	// Re-wire headNode
	linkedNode.Previous = c.headNode
	linkedNode.Next = c.headNode.Next
	c.headNode.Next.Previous = linkedNode
	c.headNode.Next = linkedNode
}

func (c *tlru) evictEntry(evictedNode *doublyLinkedNode, reason evictionReason) {
	evictedNode.Previous.Next = evictedNode.Next
	evictedNode.Next.Previous = evictedNode.Previous
	delete(c.cache, evictedNode.Key)

	if c.config.EvictionChannel != nil {
		*c.config.EvictionChannel <- evictedNode.ToEvictedEntry(reason)
	}
}

func (c *tlru) evictExpiredEntries() {
	previousNode := c.tailNode.Previous
	for previousNode != nil && previousNode != c.headNode {
		if c.config.TTL >= time.Since(previousNode.LastUpdatedAt) {
			break
		}
		c.evictEntry(previousNode, EvictionReasonExpired)
		previousNode = previousNode.Previous
	}
}

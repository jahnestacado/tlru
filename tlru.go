// * cable <https://github.com/jahnestacado/go-tlru>
// * Copyright (c) 2020 Ioannis Tzanellis
// * Licensed under the MIT License (MIT).
package tlru

import (
	"fmt"
	"sync"
	"time"
)

const (
	// Eviction reasons
	EvictionReasonDropped evictionReason = iota
	EvictionReasonExpired
	EvictionReasonReplaced
	EvictionReasonDeleted
)

const (
	// Flavors
	Read flavor = iota
	Write
)

type tlru struct {
	sync.RWMutex
	cache    map[string]*doublyLinkedNode
	config   Config
	headNode *doublyLinkedNode
	tailNode *doublyLinkedNode
	flavor   flavor
}

func New(config Config) Cache {
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

func (c *tlru) Set(entry Entry) {
	defer c.Unlock()
	c.Lock()

	linkedEntry, exists := c.cache[entry.Key]
	if exists && c.config.Flavor == Read {
		c.evictEntry(linkedEntry, EvictionReasonReplaced)
	}

	if !exists && len(c.cache) == c.config.Size {
		c.evictEntry(c.tailNode.Previous, EvictionReasonDropped)
	}

	c.insertHeadNode(entry)
}

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

	if c.config.Flavor == Read {
		c.insertHeadNode(Entry{Key: key, Value: linkedNode.Value})
	}

	cacheEntry := linkedNode.ToCacheEntry()

	return &cacheEntry
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
	defer c.Unlock()
	c.Lock()
	c.evictExpiredEntries()

	keys := make([]string, 0, len(c.cache))
	for key := range c.cache {
		keys = append(keys, key)
	}

	return keys
}

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

func (c *tlru) Clear() {
	defer c.Unlock()
	c.Lock()

	if len(c.cache) > 0 {
		c.cache = make(map[string]*doublyLinkedNode, 0)
		c.initializeDoublyLinkedList()
	}
}

func (c *tlru) GetState() State {
	defer c.RUnlock()
	c.RLock()

	state := State{
		Flavor:      c.config.Flavor,
		Entries:     make([]stateEntry, 0, len(c.cache)),
		ExtractedAt: time.Now().UTC(),
	}

	nextNode := c.headNode.Next
	for nextNode != nil && nextNode != c.tailNode {
		state.Entries = append(state.Entries, nextNode.ToStateEntry())
		nextNode = nextNode.Next
	}

	return state
}

func (c *tlru) SetState(state State) error {
	defer c.Unlock()
	c.Lock()
	if state.Flavor != c.config.Flavor {
		return fmt.Errorf("tlru.SetState: Incompatible state flavor %s", state.Flavor.String())
	}
	c.Unlock()
	c.Clear()
	c.Lock()
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

func (c *tlru) initializeDoublyLinkedList() {
	headNode := &doublyLinkedNode{Key: "head_node"}
	tailNode := &doublyLinkedNode{Key: "tail_node"}
	headNode.Next = tailNode
	tailNode.Previous = headNode
	c.headNode = headNode
	c.tailNode = tailNode
}

func (c *tlru) insertHeadNode(entry Entry) {
	var counter int64
	if c.config.Flavor == Write {
		counter++
	}

	lastUpdatedAt := entry.LastUpdatedAt
	if time.Time.IsZero(entry.LastUpdatedAt) {
		lastUpdatedAt = time.Now().UTC()
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

package tlru

import (
	"sync"
	"time"
)

type Cache interface {
	Get(key string) interface{}
	Set(entry Entry)
	Keys() []string
	Values() []interface{}
}

type Entry struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

type CacheEntry struct {
	Key         string      `json:"key"`
	Value       interface{} `json:"value"`
	Counter     int64       `json:"counter"`
	LastUpdated time.Time   `json:"last_updated"`
}

type doublyLinkedNode struct {
	Key         string
	Value       interface{}
	Counter     int64
	LastUpdated time.Time
	Previous    *doublyLinkedNode
	Next        *doublyLinkedNode
}

type Config struct {
	Size            int
	TTL             time.Duration
	EvictionChannel *chan CacheEntry
}

type lfu struct {
	cache    map[string]*doublyLinkedNode
	config   Config
	mutex    *sync.RWMutex
	headNode *doublyLinkedNode
	tailNode *doublyLinkedNode
}

func New(config Config) Cache {
	headNode := &doublyLinkedNode{Key: "head_node"}
	tailNode := &doublyLinkedNode{Key: "tail_node"}
	headNode.Next = tailNode
	tailNode.Previous = headNode

	return &lfu{
		config:   config,
		cache:    make(map[string]*doublyLinkedNode, 0),
		mutex:    &sync.RWMutex{},
		headNode: headNode,
		tailNode: tailNode,
	}
}

// func (c *lfu) print() {
// 	next := c.headNode
// 	for next != nil {
// 		fmt.Printf("{%p} [%s - %p](%d) {%p} -> ", next.Previous, next.Key, next, next.Counter, next.Next)
// 		next = next.Next
// 	}
// 	fmt.Println("----")
// }

func (c *lfu) Set(entry Entry) {
	defer c.mutex.Unlock()
	c.mutex.Lock()

	_, exists := c.cache[entry.Key]
	if !exists && len(c.cache) == c.config.Size {
		c.evictEntry(c.tailNode.Previous)
	}

	c.insertHeadNode(entry)
}

func (c *lfu) Get(key string) interface{} {
	defer c.mutex.RUnlock()
	c.mutex.RLock()

	linkedNode, exists := c.cache[key]
	if !exists {
		return nil
	}

	if c.config.TTL < time.Since(linkedNode.LastUpdated) {
		c.evictEntry(linkedNode)
		return nil
	}

	return linkedNode.Value
}

func (c *lfu) Keys() []string {
	defer c.mutex.RUnlock()
	c.mutex.RLock()
	c.evictExpiredEntries()

	keys := make([]string, 0, c.config.Size)
	for key := range c.cache {
		keys = append(keys, key)
	}

	return keys
}

func (c *lfu) Values() []interface{} {
	defer c.mutex.RUnlock()
	c.mutex.RLock()
	c.evictExpiredEntries()

	values := make([]interface{}, 0, c.config.Size)
	for _, linkedNode := range c.cache {
		values = append(values, linkedNode.Value)
	}

	return values
}

func (c *lfu) insertHeadNode(entry Entry) {
	counter := int64(1)
	lastUpdated := time.Now().UTC()
	linkedNode, exists := c.cache[entry.Key]
	if exists {
		if c.config.TTL >= time.Since(linkedNode.LastUpdated) {
			counter++
		}
		linkedNode.Counter = counter
		linkedNode.LastUpdated = lastUpdated

		// Re-wire siblings of linkedNode
		linkedNode.Next.Previous = linkedNode.Previous
		linkedNode.Previous.Next = linkedNode.Next
	} else {
		linkedNode = &doublyLinkedNode{
			Key:         entry.Key,
			Value:       entry.Value,
			Counter:     counter,
			LastUpdated: lastUpdated,
			Previous:    c.headNode,
			Next:        c.headNode.Next,
		}

		c.cache[entry.Key] = linkedNode
	}

	// // Re-wire headNode
	linkedNode.Previous = c.headNode
	linkedNode.Next = c.headNode.Next
	c.headNode.Next.Previous = linkedNode
	c.headNode.Next = linkedNode
}

func (c *lfu) evictEntry(evictedNode *doublyLinkedNode) {
	evictedNode.Previous.Next = c.tailNode
	c.tailNode.Previous = evictedNode.Previous
	delete(c.cache, evictedNode.Key)

	if c.config.EvictionChannel != nil {
		*c.config.EvictionChannel <- CacheEntry{
			Key:         evictedNode.Key,
			Value:       evictedNode.Value,
			Counter:     evictedNode.Counter,
			LastUpdated: evictedNode.LastUpdated,
		}
	}
}

func (c *lfu) evictExpiredEntries() {
	previousNode := c.tailNode.Previous
	for previousNode != nil && previousNode != c.headNode {
		if c.config.TTL >= time.Since(previousNode.LastUpdated) {
			break
		}
		c.evictEntry(previousNode)
		previousNode = previousNode.Previous
	}
}

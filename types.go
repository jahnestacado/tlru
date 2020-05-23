// * cable <https://github.com/jahnestacado/go-tlru>
// * Copyright (c) 2020 Ioannis Tzanellis
// * Licensed under the MIT License (MIT).
package tlru

import "time"

type Cache interface {
	Get(key string) *CacheEntry
	Set(entry Entry)
	Keys() []string
	Entries() []CacheEntry
	Delete(key string)
	Clear()
	GetState() State
	SetState(state State) error
}

type Config struct {
	Size            int
	TTL             time.Duration
	EvictionChannel *chan EvictedEntry
	EvictionPolicy  evictionPolicy
}

type Entry struct {
	Key       string      `json:"key"`
	Value     interface{} `json:"value"`
	Timestamp *time.Time  `json:"timestamp"`
}

type CacheEntry struct {
	Value         interface{} `json:"value"`
	Counter       int64       `json:"counter"`
	LastUpdatedAt time.Time   `json:"last_updated_at"`
	CreatedAt     time.Time   `json:"created_at"`
}

type EvictedEntry struct {
	Key           string         `json:"key"`
	Value         interface{}    `json:"value"`
	Counter       int64          `json:"counter"`
	LastUpdatedAt time.Time      `json:"last_updated_at"`
	CreatedAt     time.Time      `json:"created_at"`
	Reason        evictionReason `json:"reason"`
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

func (f evictionPolicy) String() string {
	return [...]string{"LRA", "LRI"}[f]
}

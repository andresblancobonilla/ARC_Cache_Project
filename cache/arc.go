package cache

import (
	"container/list"
)

// An ARC is a fixed-size in-memory cache with adaptive replacement eviction
type ARC struct {
	cache          map[string]Value
	t1List         *LRU
	t2List         *LRU
	b1List         *LRU
	b2List         *LRU
	totalUsedBytes int
	limit          int
	stats          Stats
}

type Value struct {
	bytes   []byte
	element *list.Element
}

type CacheList struct {
	list  *list.List
	cache map[string]Value
}

func NewCacheList() *CacheList {
	var cacheList CacheList
	cacheList.list = list.New()
	cacheList.cache = make(map[string]Value)
	return &cacheList
}

// NewLRU returns a pointer to a new LRU with a capacity to store limit bytes
func NewARC(limit int) *ARC {
	var arc ARC
	arc.cache = make(map[string]Value)
	arc.t1List = NewCacheList()
	arc.t2List = NewCacheList()
	arc.b1List = NewCacheList()
	arc.bList = NewCacheList()
	arc.usedBytes = 0
	arc.limit = limit
	arc.stats = Stats{0, 0}
	return &arc
}

// func NewVal(bytes []byte, element *list.Element) *Value {
// 	var value Value
// 	value.bytes = bytes
// 	value.element = element
// 	return &value
// }

// MaxStorage returns the maximum number of bytes this LRU can store
func (lru *LRU) MaxStorage() int {
	return lru.limit
}

// RemainingStorage returns the number of unused bytes available in this LRU
func (lru *LRU) RemainingStorage() int {
	return (lru.limit - lru.usedBytes)
}

// Get returns the value associated with the given key, if it exists.
// This operation counts as a "use" for that key-value pair
// ok is true if a value was found and false otherwise.
func (lru *LRU) Get(key string) (value []byte, ok bool) {

	if value, found := lru.cache[key]; found {
		lru.stats.Hits++

		// testing, print front
		// front := lru.nodes.Front()
		// fmt.Sprint(front.Value)
		// elm1 := new(list.Element)
		// elm1.Value = "hello"
		// elm2 := new(list.Element)
		// elm2.Value = "hello"

		// fmt.Println(elm1 == elm2)

		lru.nodes.MoveToFront(value.element)
		// testing
		// front = lru.nodes.Front()
		// fmt.Sprint(front.Value)
		return value.bytes, true
	} else {
		lru.stats.Misses++
		return nil, false
	}
}

// Remove removes and returns the value associated with the given key, if it exists.
// ok is true if a value was found and false otherwise
func (lru *LRU) Remove(key string) (value []byte, ok bool) {
	if value, found := lru.cache[key]; found {
		delete(lru.cache, key)
		lru.usedBytes = lru.usedBytes - len(value.bytes) - len(key)
		// traverse linked list to remove the given key
		lru.nodes.Remove(value.element)
		return value.bytes, true
	} else {
		return nil, false
	}
}

// Set associates the given value with the given key, possibly evicting values
// to make room. Returns true if the binding was added successfully, else false.
func (lru *LRU) Set(key string, value []byte) bool {
	if old_val, found := lru.cache[key]; found {
		lru.usedBytes = lru.usedBytes - len(old_val.bytes) + len(value)
		var new_val Value
		new_val.bytes = value
		new_val.element = old_val.element
		lru.cache[key] = new_val
		lru.nodes.MoveToFront(new_val.element)
		return true
	}

	itemSize := len(value) + len(key)

	if itemSize > lru.limit {
		return false
	}

	for itemSize > lru.RemainingStorage() {
		back := lru.nodes.Back()
		// testing
		// fmt.Sprint(back.Value)
		// fmt.Println(lru.nodes.Len())
		// evictedKey := fmt.Sprintf("%v", back.Value)
		evictedKey := back.Value.(string)

		// fmt.Println(evictedKey)

		lru.usedBytes = lru.usedBytes - len(lru.cache[evictedKey].bytes) - len(evictedKey)
		lru.nodes.Remove(back)
		delete(lru.cache, evictedKey)
	}

	// element := new(list.Element)
	// element.Value = key
	// fmt.Println("element Value: ", element.Value)
	element := lru.nodes.PushFront(key)
	lru.usedBytes += itemSize
	var new_val Value
	new_val.bytes = value
	new_val.element = element
	// new_value := NewVal(value, element)
	lru.cache[key] = new_val

	return true
}

// Len returns the number of bindings in the LRU.
func (lru *LRU) Len() int {
	return len(lru.cache)
}

// Stats returns statistics about how many search hits and misses have occurred.
func (lru *LRU) Stats() *Stats {
	return &lru.stats
}

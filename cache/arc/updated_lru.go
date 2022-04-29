package cache

import (
	"container/list"
)

// An LRU is a fixed-size in-memory cache with least-recently-used eviction
type LRU struct {
	cache       map[string]Value
	nodes       *list.List
	usedEntries int
	limit       int
	stats       Stats
}

type Value struct {
	bytes   []byte
	element *list.Element
}

// NewLRU returns a pointer to a new LRU with a capacity to store limit bytes
func NewLRU(limit int) *LRU {
	var lru LRU
	lru.cache = make(map[string]Value)
	lru.nodes = new(list.List)
	lru.usedEntries = 0
	lru.limit = limit
	lru.stats = Stats{0, 0}
	return &lru
}

// func NewVal(bytes []byte, element *list.Element) *Value {
// 	var value Value
// 	value.bytes = bytes
// 	value.element = element
// 	return &value
// }

// MaxStorage returns the maximum number of entries this LRU can store
func (lru *LRU) MaxEntries() int {
	return lru.limit
}

// RemainingSpaces returns the number of unused spaces for entries available in this LRU
func (lru *LRU) RemainingSpaces() int {
	return (lru.limit - lru.usedEntries)
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

// Check returns the value associated with the given key, if it exists.
// This operation DOES NOT counts as a "use" for that key-value pair
// ok is true if a value was found and false otherwise.
func (lru *LRU) Check(key string) (value []byte, ok bool) {

	if value, found := lru.cache[key]; found {
		//lru.stats.Hits++

		// testing, print front
		// front := lru.nodes.Front()
		// fmt.Sprint(front.Value)
		// elm1 := new(list.Element)
		// elm1.Value = "hello"
		// elm2 := new(list.Element)
		// elm2.Value = "hello"

		// fmt.Println(elm1 == elm2)

		//lru.nodes.MoveToFront(value.element)
		// testing
		// front = lru.nodes.Front()
		// fmt.Sprint(front.Value)
		return value.bytes, true
	} else {
		//lru.stats.Misses++
		return nil, false
	}
}

// Remove removes and returns the value associated with the given key, if it exists.
// ok is true if a value was found and false otherwise
func (lru *LRU) Remove(key string) (value []byte, ok bool) {
	if value, found := lru.cache[key]; found {
		delete(lru.cache, key)
		lru.usedEntries--
		// traverse linked list to remove the given key
		lru.nodes.Remove(value.element)
		return value.bytes, true
	} else {
		return nil, false
	}
}

// Evict removes the least recently used binding from the LRU
// and returns the key associated with it.
func (lru *LRU) Evict() (key string) {
	back := lru.nodes.Back()
	evictedKey := back.Value.(string)
	lru.usedEntries--
	lru.nodes.Remove(back)
	delete(lru.cache, evictedKey)
	return evictedKey
}

// Set associates the given value with the given key, possibly evicting values
// to make room. Returns true if the binding was added successfully, else false.
func (lru *LRU) Set(key string, value []byte) bool {
	if old_val, found := lru.cache[key]; found {
		var new_val Value
		new_val.bytes = value
		new_val.element = old_val.element
		lru.cache[key] = new_val
		lru.nodes.MoveToFront(new_val.element)
		return true
	}
	if lru.RemainingSpaces() == 0 {

		back := lru.nodes.Back()
		// testing
		// fmt.Sprint(back.Value)
		// fmt.Println(lru.nodes.Len())
		// evictedKey := fmt.Sprintf("%v", back.Value)
		evictedKey := back.Value.(string)

		// fmt.Println(evictedKey)

		lru.usedEntries--
		lru.nodes.Remove(back)
		delete(lru.cache, evictedKey)
	}

	// element := new(list.Element)
	// element.Value = key
	// fmt.Println("element Value: ", element.Value)
	element := lru.nodes.PushFront(key)
	lru.usedEntries++
	var new_val Value
	new_val.bytes = value
	new_val.element = element
	// new_value := NewVal(value, element)
	lru.cache[key] = new_val

	return true
}

// Len returns the number of bindings in the LRU.
func (lru *LRU) Len() int {
	return lru.usedEntries
}

// Stats returns statistics about how many search hits and misses have occurred.
func (lru *LRU) Stats() *Stats {
	return &lru.stats
}

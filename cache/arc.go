package cache

// An ARC is a fixed-size in-memory cache with adaptive replacement eviction
type ARC struct {
	t1List         *LRU
	t2List         *LRU
	b1List         *LRU
	b2List         *LRU
	cacheDirectory map[string]Value
	targetMarker   int
	totalUsedBytes int
	limit          int
	stats          Stats
}

// NewARC returns a pointer to a new LRU with a capacity to store limit bytes
func NewARC(limit int) *ARC {
	var arc ARC
	arc.t1List = NewLRU(limit)
	arc.t2List = NewLRU(limit)
	arc.b1List = NewLRU(limit)
	arc.b2List = NewLRU(limit)
	arc.cacheDirectory = make(map[string]Value)
	arc.targetMarker = 0
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
func (arc *ARC) MaxStorage() int {
	return arc.limit
}

// RemainingStorage returns the number of unused bytes available in this LRU
func (arc *ARC) RemainingStorage() int {
	return (arc.limit - (arc.t1List.usedBytes + arc.t2List.usedBytes))
}

// Get returns the value associated with the given key, if it exists.
// This operation counts as a "use" for that key-value pair
// ok is true if a value was found and false otherwise.
func (arc *ARC) Get(key string) (value []byte, ok bool) {
	val, found := arc.cacheDirectory[key]
	if found {
		hit := arc.Access(key)
		if hit {
			arc.stats.Hits++
			return val.bytes, true
		}
	} else {
		arc.stats.Misses++
		return nil, false
	}

	return nil, false

	// if value, found := arc.t1List.Check(key); found {
	// 	arc.t1List.Remove(key)
	// 	arc.t2List.Set(key, value)
	// 	arc.stats.Hits++
	// 	return value, true
	// }
	// if value, found := arc.t2List.Get(key); found {
	// 	arc.stats.Hits++
	// 	return value, true
	// }

	// arc.stats.Misses++
	// return nil, false

}

// Remove removes and returns the value associated with the given key, if it exists.
// ok is true if a value was found and false otherwise
func (arc *ARC) Remove(key string) (value []byte, ok bool) {
	if _, found := arc.cacheDirectory[key]; found {

		if value, found := arc.t1List.Check(key); found {
			arc.t1List.Remove(key)
			return value, true
		}

		if value, found := arc.t2List.Check(key); found {
			arc.t2List.Remove(key)
			return value, true
		}

		if value, found := arc.b1List.Check(key); found {
			arc.b1List.Remove(key)
			return value, true
		}

		if value, found := arc.b2List.Check(key); found {
			arc.b2List.Remove(key)
			return value, true
		}
		delete(arc.cacheDirectory, key)

	}
	return nil, false

}

// Evict evicts an entry adaptably from either T1 or T2 depending on the
// location of the target marker in order to add a new entry.
func (arc *ARC) Evict(key string) {
	_, b2Hit := arc.b2List.Check(key)
	var evictedKey string
	//value, b2Hit := arc.b1List.Check(key)
	if (arc.t1List.Len() >= 0) && (b2Hit && arc.t1List.Len() == arc.targetMarker) || arc.t1List.Len() > arc.targetMarker {
		evictedKey = arc.t1List.Evict()
		arc.b1List.Set(evictedKey, nil)
	} else {
		evictedKey = arc.t2List.Evict()
		arc.b2List.Set(evictedKey, nil)
	}
	delete(arc.cacheDirectory, evictedKey)
}

// Access adapts the cache and the target marker based on if the access hit T1, T2, B1, or B2.
// Returns true if the the access hit in T1 or T2, false otherwise.
func (arc *ARC) Access(key string) (hit bool) {
	b1Len := (arc.b1List.Len())
	b2Len := (arc.b2List.Len())

	// Case I: key is found in either t1 or t2
	if value, found := arc.t1List.Check(key); found {
		arc.t1List.Remove(key)
		arc.t2List.Set(key, value)
		return true
	}

	if value, found := arc.t2List.Check(key); found {
		arc.t2List.Set(key, value)
		return true
	}

	// Case II: key is found in b1
	if _, found := arc.b1List.Check(key); found {
		ratio := b2Len / b1Len
		arc.targetMarker = min(arc.limit, arc.targetMarker+max(ratio, 1))
		arc.Evict(key)
		return false
	}
	// Case III: key is found in b2
	if _, found := arc.b2List.Check(key); found {
		ratio := b1Len / b2Len
		arc.targetMarker = max(0, arc.targetMarker-max(ratio, 1))
		arc.Evict(key)
		return false
	}
	return false

}

// Set associates the given value with the given key, possibly evicting values
// to make room. Returns true if the binding was added successfully, else false.
func (arc *ARC) Set(key string, value []byte) bool {

	t1Len := (arc.t1List.usedBytes)
	b1Len := (arc.b1List.usedBytes)
	t2Len := (arc.t2List.usedBytes)
	b2Len := (arc.b2List.usedBytes)
	l1Len := t1Len + b1Len
	l2Len := t2Len + b2Len
	totalLen := l1Len + l2Len

	arc.Access(key)

	_, found := arc.cacheDirectory[key]

	if found {
		if _, found := arc.t1List.Check(key); found {
			arc.t1List.Set(key, value)
			return true
		}
		if _, found := arc.t2List.Check(key); found {
			arc.t2List.Set(key, value)
			return true
		}
		if _, found := arc.b1List.Check(key); found {
			arc.b1List.Remove(key)
			arc.t2List.Set(key, value)
			return true
		}
		if _, found := arc.b2List.Check(key); found {
			arc.b2List.Remove(key)
			arc.t2List.Set(key, value)
			return true
		}
	}

	// Case IV: key is not found
	if !found {
		// case (i)
		var evictedKey string
		if l1Len == arc.limit {
			if t1Len < arc.limit {
				arc.b1List.Evict()
				delete(arc.cacheDirectory, evictedKey)
				arc.Evict(key)
			} else {
				evictedKey := arc.t1List.Evict()
				arc.b1List.Set(evictedKey, nil)
			}
		}

		// case (ii)
		if l1Len < arc.limit && totalLen >= arc.limit {
			if totalLen == 2*arc.limit {
				arc.b2List.Evict()
				delete(arc.cacheDirectory, evictedKey)
			}
			arc.Evict(key)
		}
		arc.t1List.Set(key, value)
		arc.cacheDirectory[key] = arc.t1List.cache[key]
	}

	return true

	// t1Len := (arc.t1List.Len())
	// b1Len := (arc.b1List.Len())
	// t2Len := (arc.t2List.Len())
	// b2Len := (arc.b2List.Len())
	// l1Len := t1Len + b1Len
	// l2Len := t2Len + b2Len
	// totalLen := l1Len + l2Len

	// // Case I: key is found in either t1 or t2
	// if _, found := arc.t1List.Check(key); found {
	// 	arc.t1List.Remove(key)
	// 	arc.t2List.Set(key, value)
	// 	return true
	// }

	// if _, found := arc.t2List.Check(key); found {
	// 	arc.t2List.Set(key, value)
	// 	return true
	// }

	// // Case II: key is found in b1
	// if _, found := arc.b1List.Check(key); found {
	// 	ratio := b2Len / b1Len
	// 	arc.targetMarker = min(arc.limit, arc.targetMarker+max(ratio, 1))
	// 	arc.Evict(key)
	// 	arc.b1List.Remove(key)
	// 	arc.t2List.Set(key, value)
	// }
	// // Case III: key is found in b2
	// if value, found := arc.b2List.Check(key); found {
	// 	ratio := b1Len / b2Len
	// 	arc.targetMarker = max(0, arc.targetMarker-max(ratio, 1))
	// 	arc.Evict(key)
	// 	arc.b2List.Remove(key)
	// 	arc.t2List.Set(key, value)
	// }
	// Case IV: key is not found

	// case (i)
	// if l1Len == arc.limit {
	// 	if t1Len < arc.limit {
	// 		arc.b1List.Evict()
	// 		arc.Evict(key)
	// 	} else {
	// 		evictedKey := arc.t1List.Evict()
	// 		arc.b1List.Set(evictedKey, nil)
	// 	}
	// }

	// // case (ii)
	// if l1Len < arc.limit && totalLen >= arc.limit {
	// 	if totalLen == 2*arc.limit {
	// 		arc.b2List.Evict()
	// 	}
	// 	arc.Evict(key)
	// }

	// arc.t1List.Set(key, value)
	// return true
}

// Len returns the number of bindings in the ARC.
func (arc *ARC) Len() int {
	return arc.t1List.Len() + arc.t2List.Len()
}

// Stats returns statistics about how many search hits and misses have occurred.
func (arc *ARC) Stats() *Stats {
	return &arc.stats
}

// returns the lesser of ints x and y.
func min(x int, y int) int {
	if x < y {
		return x
	} else {
		return y
	}
}

// returns the greater of ints x and y.
func max(x int, y int) int {
	if x > y {
		return x
	} else {
		return y
	}
}

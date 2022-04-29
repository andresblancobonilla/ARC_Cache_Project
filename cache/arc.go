package cache

import (
	"os"
	"path/filepath"
)

// An ARC is a fixed-size in-memory cache with adaptive replacement eviction
type ARC struct {
	t1List         *LRU
	t2List         *LRU
	b1List         *LRU
	b2List         *LRU
	cacheDirectory string
	targetMarker   int
	totalUsedBytes int
	limit          int
	stats          Stats
}

// NewARC returns a pointer to a new ARC with a capacity to store limited entries
func NewARC(limit int) (*ARC, error) {
	var arc ARC
	arc.t1List = NewLRU(limit)
	arc.t2List = NewLRU(limit)
	arc.b1List = NewLRU(limit)
	arc.b2List = NewLRU(limit)
	arc.cacheDirectory = "Cache_Directory"
	os.Mkdir(arc.cacheDirectory, 0777)
	arc.targetMarker = 0
	arc.limit = limit
	arc.stats = Stats{0, 0}
	return &arc, nil
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

// RemainingStorage returns the number of unused spaces available for entries in this LRU
func (arc *ARC) RemainingSpaces() int {
	return (arc.limit - (arc.t1List.usedEntries + arc.t2List.usedEntries))
}

// Get returns the value associated with the given key, if it exists.
// This operation counts as a "use" for that key-value pair
// ok is true if a value was found and false otherwise.
func (arc *ARC) Get(key string) (value []byte, ok bool) {
	value, inCacheDirectory := arc.CheckCacheDirectory(key)
	if inCacheDirectory {
		_, inCache := arc.CheckCache(key)
		arc.Access(key)
		if inCache {
			arc.stats.Hits++
			ok = inCache
			return value, ok
		}
	} else {
		ok = false
		arc.stats.Misses++
		return nil, ok
	}
	return nil, false

}

// CheckCache returns the value associated with the given key, if it exists.
// This operation DOES NOT counts as a "use" for that key-value pair
// ok is true if a value was found and false otherwise.
func (arc *ARC) CheckCache(key string) (value []byte, okarc bool) {
	if val, found := arc.t1List.Check(key); found {
		okarc = found
		value = val
	}

	if val, found := arc.t2List.Check(key); found {
		okarc = found
		value = val
	}
	return value, okarc
}

// Check returns the value associated with the given key, if it exists.
// This operation DOES NOT counts as a "use" for that key-value pair
// ok is true if a value was found and false otherwise.
func (arc *ARC) CheckCacheDirectory(key string) (value []byte, okcd bool) {

	if val, found := arc.t1List.Check(key); found {
		okcd = found
		value = val
	}

	if val, found := arc.t2List.Check(key); found {
		okcd = found
		value = val
	}

	if _, found := arc.b1List.Check(key); found {
		okcd = found
		okcd = !found
		value = nil
	}
	if _, found := arc.b2List.Check(key); found {
		okcd = found
		okcd = !found
		value = nil
	}

	return value, okcd
}

// Remove removes and returns the value associated with the given key, if it exists.
// ok is true if a value was found and false otherwise
func (arc *ARC) Remove(key string) (value []byte, ok bool) {
	value, found := arc.CheckCacheDirectory(key)

	if !found {
		ok = false
	} else {
		ok = true
		if _, found := arc.t1List.Check(key); found {
			arc.t1List.Remove(key)
		}

		if _, found := arc.t2List.Check(key); found {
			arc.t2List.Remove(key)
		}

		if _, found := arc.b1List.Check(key); found {
			arc.b1List.Remove(key)
		}

		if _, found := arc.b2List.Check(key); found {
			arc.b2List.Remove(key)
		}
		//arc.RemoveFromDisk(key)
	}
	return value, ok

}

// Evict evicts an entry adaptably from either T1 or T2 depending on the
// location of the target marker in order to add a new entry.
func (arc *ARC) Evict(key string) {
	t1Len := arc.t1List.Len()
	_, b2Hit := arc.b2List.Check(key)
	var evictedKey string
	//value, b2Hit := arc.b1List.Check(key)
	if (arc.t1List.Len() > 0) && ((b2Hit && (t1Len == arc.targetMarker)) || (t1Len > arc.targetMarker)) {
		evictedKey = arc.t1List.Evict()
		arc.b1List.Set(evictedKey, nil)
	} else {
		evictedKey = arc.t2List.Evict()
		arc.b2List.Set(evictedKey, nil)
	}
}

// Access accesses the cache directory in search of key,
// and adapts the cache depending which list key was found in.
func (arc *ARC) Access(key string) {

	// Case I: key is found in either t1 or t2
	if value, found := arc.t1List.Check(key); found {
		arc.t1List.Remove(key)
		arc.t2List.Set(key, value)
		return
	}

	if value, found := arc.t2List.Check(key); found {
		arc.t2List.Set(key, value)
		return
	}

	b1Len := arc.b1List.Len()
	b2Len := arc.b2List.Len()

	// Case II: key is found in b1
	if value, found := arc.b1List.Check(key); found {
		ratio := b2Len / b1Len
		arc.targetMarker = min(arc.limit, arc.targetMarker+max(ratio, 1))
		arc.Evict(key)
		arc.b1List.Remove(key)
		value = arc.ReadFromDisk(key)
		arc.t2List.Set(key, value)
		return
	}
	// Case III: key is found in b2
	if value, found := arc.b2List.Check(key); found {
		ratio := b1Len / b2Len
		arc.targetMarker = max(0, arc.targetMarker-max(ratio, 1))
		arc.Evict(key)
		arc.b2List.Remove(key)
		value = arc.ReadFromDisk(key)
		arc.t2List.Set(key, value)
		return
	}

}

// Set associates the given value with the given key, possibly evicting values
// to make room. Returns true if the binding was added successfully, else false.
func (arc *ARC) Set(key string, value []byte) bool {

	t1Len := arc.t1List.Len()
	b1Len := arc.b1List.Len()
	t2Len := arc.t2List.Len()
	b2Len := arc.b2List.Len()
	l1Len := t1Len + b1Len
	l2Len := t2Len + b2Len
	totalLen := l1Len + l2Len
	value, inCacheDirectory := arc.CheckCacheDirectory(key)

	if inCacheDirectory {
		arc.Access(key)
		arc.t2List.Set(key, value)
		return true
	}

	// Case IV: key is not found
	if !inCacheDirectory {
		// Case (A)
		var evictedKey string
		if l1Len == arc.limit {
			if t1Len < arc.limit {
				evictedKey = arc.b1List.Evict()
				//arc.RemoveFromDisk(evictedKey)
				arc.Evict(key)
			} else {
				evictedKey = arc.t1List.Evict()
				arc.b1List.Set(evictedKey, nil)
			}
		}

		// Case (B)
		if l1Len < arc.limit && totalLen >= arc.limit {
			if totalLen == 2*arc.limit {
				arc.b2List.Evict()
				//arc.RemoveFromDisk(evictedKey)
			}
			arc.Evict(key)
		}
		arc.t1List.Set(key, value)
		arc.WriteToDisk(key, value)
	}

	return true

}

// WriteToDisk writes the key/value pair to a new file on disk.
// The key is the name of the file and the contents are the value.
func (arc *ARC) WriteToDisk(key string, value []byte) {
	path := filepath.Join(arc.cacheDirectory, key)
	file, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	file.Write(value)
}

// ReadFromDisk returns the value associated with a key.
// The value is stored on disk in a file named the same as the key.
func (arc *ARC) ReadFromDisk(key string) (value []byte) {
	path := filepath.Join(arc.cacheDirectory, key)
	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	file.Read(value)
	return value
}

// RemoveFromDisk deletes the file associated with a key on disk.
func (arc *ARC) RemoveFromDisk(key string) {
	path := filepath.Join(arc.cacheDirectory, key)
	err := os.Remove(path)
	if err != nil {
		panic(err)
	}
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

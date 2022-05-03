package arc

import (
	"fmt"
	"os"
	"path/filepath"
	"errors"
)

// An ARC is a fixed-size in-memory cache with adaptive replacement eviction.
// In the comments, "cache" refers to the collection of the 2 LRUs T1 and T2/
// "Cache directory" refers to 4 LRUs T1, T2, and the 2 ghost lists B1, and B2.
// L1 means T1 + B1, L2 means T2 + B2.
// L1 stores single-referenced entries, L2 stores frequently referenced entries.
type ARC struct {
	// t1List and t2List have values associated with their keys.
	t1List *LRU
	t2List *LRU
	// b1List and b2List have nil associated with their keys.
	// They are ghost lists meant for keeping track
	// of the recently evicted keys only.
	b1List *LRU
	b2List *LRU
	// The name of the directory in which the entire cache directory
	// is stored on disk, both keys and values.
	// Keys are filenames and values are file contents.
	// This enables the algorithm to properly fetch B1 and B2's values
	// if they are hit and need to be moved back into the cache.
	cacheDirectory string
	// Target size of T1, which adapts depending on ghost list hits.
	targetMarker int
	// The maximum number of entries that can be added to the cache.
	// Also, T1 + T2 <= limit, B1 + B2 <= limit, L1 <= limit, L1 + L2 <= 2*limit
	limit int
	stats Stats
	// map was used for testing
	//cache map[string][]byte
}

// NewARC returns a pointer to a new ARC with a capacity to store limited entries
func NewARC(limit int) (*ARC, error) {
	if limit <= 0 {
		return nil, errors.New("Capacity must be greater than zero")
	}
	var arc ARC
	arc.t1List = NewLRU(limit)
	arc.t2List = NewLRU(limit)
	arc.b1List = NewLRU(limit)
	arc.b2List = NewLRU(limit)
	//arc.cache = make(map[string][]byte)
	arc.cacheDirectory = "cache_directory"
	// Make a new directory on disk that everyone can read/write to
	os.Mkdir(arc.cacheDirectory, 0777)
	arc.targetMarker = 0
	arc.limit = limit
	arc.stats = Stats{0, 0}
	return &arc, nil
}

// MaxEntries returns the maximum number of entries this ARC cache can store
func (arc *ARC) MaxEntries() int {
	return arc.limit
}

// RemainingSpaces returns the number of unused spaces available for entries in the ARC cache
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
// This operation DOES NOT counts as a "use" for that key-value pair,
// it just "checks" if the pair is in the cache.
// ok is true if a value was found and false otherwise.
func (arc *ARC) CheckCache(key string) (value []byte, okc bool) {
	if val, found := arc.t1List.Check(key); found {
		okc = found
		value = val
	}

	if val, found := arc.t2List.Check(key); found {
		okc = found
		value = val
	}
	return value, okc
}

// CheckCacheDirectory returns the value associated with the given key, if it exists.
// This operation DOES NOT counts as a "use" for that key-value pair,
// it just "checks" if the pair is in the cache directory.
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
		value = nil
	}
	if _, found := arc.b2List.Check(key); found {
		okcd = found
		value = nil
	}
	return value, okcd
}

// Remove removes and returns the value associated with the given key, if it exists.
// This erases the key-value pair from both the cache lists and the on-disk cache directory
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
		arc.RemoveFromDisk(key)
		//delete(arc.cache, key)
	}
	return value, ok

}

// Evict evicts an entry adaptively from either T1 or T2 (into B1 or B2),
// depending on the location of the target marker, in order to add a new entry.
func (arc *ARC) Evict(key string) {
	t1Len := arc.t1List.Len()
	bLen := arc.b1List.Len() + arc.b2List.Len()
	_, b2Hit := arc.b2List.Check(key)

	// Evict from T1
	if (arc.t1List.Len() > 0) && ((b2Hit && (t1Len == arc.targetMarker)) || (t1Len > arc.targetMarker)) {
		evictedKey, ok := arc.t1List.Evict()
		if ok {
			// If adding an entry will violate B1 + B2 <= limit, Evict() clears a space
			// from the appropriate ghost list.
			if bLen == arc.limit {
				ghostEvictedKey, gok := arc.b1List.Evict()
				if !gok {
					ghostEvictedKey, _ = arc.b2List.Evict()
				}
				arc.RemoveFromDisk(ghostEvictedKey)
				//delete(arc.cache, ghostEvictedKey)
			}
			arc.b1List.Set(evictedKey, nil)
		}
		// Evict from T2
	} else {
		evictedKey, ok := arc.t2List.Evict()
		if ok {
			// If adding an entry will violate B1 + B2 <= c, Evict() clears a space
			// from the appropriate ghost list.
			if bLen == arc.limit {
				ghostEvictedKey, gok := arc.b2List.Evict()
				if !gok {
					ghostEvictedKey, _ = arc.b1List.Evict()
				}
				arc.RemoveFromDisk(ghostEvictedKey)
				//delete(arc.cache, ghostEvictedKey)
			}
			arc.b2List.Set(evictedKey, nil)
		}
	}
}

// Access accesses the cache directory in search of the key,
// and adapts the cache depending which list key was found in.
func (arc *ARC) Access(key string) {

	// Case I: key is found in either T1 or T2
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

	// Case II: key is found in B1
	if _, found := arc.b1List.Check(key); found {
		// Adapt the target marker.
		ratio := b2Len / b1Len
		arc.targetMarker = min(arc.limit, arc.targetMarker+max(ratio, 1))
		// Fetch B1's value from the on-disk cache directory.
		value := arc.ReadFromDisk(key)
		// Corner case: Evict might end up deleting key from the on-disk cache directory,
		// if it is the least recently used entry in B1.
		// Move key to the front of B1 to prevent this from happening.
		arc.b1List.Set(key, nil)
		arc.Evict(key)
		arc.b1List.Remove(key)
		// Add B1 back to the cache.
		arc.t2List.Set(key, value)
		return
	}
	// Case III: key is found in B2
	if _, found := arc.b2List.Check(key); found {
		// Adapt the target marker.
		ratio := b1Len / b2Len
		arc.targetMarker = max(0, arc.targetMarker-max(ratio, 1))
		// Fetch B2's value from the on-disk cache directory.
		value := arc.ReadFromDisk(key)
		// Corner case: Evict might end up deleting key from the on-disk cache directory,
		// if it is the least recently used entry in B2.
		// Move key to the front of B2 to prevent this from happening.
		arc.b2List.Set(key, nil)
		arc.Evict(key)
		arc.b2List.Remove(key)
		// Add B2 back to the cache.
		arc.t2List.Set(key, value)
		return
	}

}

// Set associates the given value with the given key, possibly evicting values
// to make room. Returns true if the binding was added successfully, else false.
func (arc *ARC) Set(key string, value []byte) (ok bool) {

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
		// If key is in the cache directory, accessing it will move
		// it to the front of T2 no matter what.
		arc.t2List.Set(key, value)
		return true
	}

	// Case IV: key is not found
	if !inCacheDirectory {
		// Case (A): when L1 has exactly arc.limit number of pages
		if l1Len == arc.limit {
			if t1Len < arc.limit {
				evictedKey, _ := arc.b1List.Evict()
				arc.RemoveFromDisk(evictedKey)
				//delete(arc.cache, evictedKey)
				arc.Evict(key)
			} else {
				evictedKey, _ := arc.t1List.Evict()
				arc.RemoveFromDisk(evictedKey)
				//delete(arc.cache, evictedKey)
			}
		}

		// Case (B): when L1 has less than arc.limit number of pages
		if l1Len < arc.limit && totalLen >= arc.limit {
			if totalLen == 2*arc.limit {
				evictedKey, _ := arc.b2List.Evict()
				arc.RemoveFromDisk(evictedKey)
				//delete(arc.cache, evictedKey)
			}
			arc.Evict(key)
		}

		arc.t1List.Set(key, value)
		// Add the key-value to the on-disk cache directory.
		arc.WriteToDisk(key, value)

		// testing
		// t1Len = arc.t1List.Len()
		// b1Len = arc.b1List.Len()
		// t2Len = arc.t2List.Len()
		// b2Len = arc.b2List.Len()
		// l1Len = t1Len + b1Len
		// l2Len = t2Len + b2Len
		// totalLen = l1Len + l2Len
		//[key] = value
		// t1Len = arc.t1List.Len()
		// b1Len = arc.b1List.Len()
		// t2Len = arc.t2List.Len()
		// b2Len = arc.b2List.Len()
		// l1Len = t1Len + b1Len
		// l2Len = t2Len + b2Len
		// totalLen = l1Len + l2Len
		return true
	}

	return false

}

// WriteToDisk writes the key-value pair to a new file on disk.
// The key is the name of the file and the file's contents are the value.
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
// The value is stored in the on-disk cache directory
// in a file named the same as the key.
func (arc *ARC) ReadFromDisk(key string) (value []byte) {
	_, found := arc.CheckCacheDirectory(key)
	if found {
		path := filepath.Join(arc.cacheDirectory, key)
		file, err := os.Open(path)
		if err != nil {
			panic(err)
		}
		defer file.Close()
		file.Read(value)
	} else {
		fmt.Println("not in cache directory")
	}
	return value
}

// RemoveFromDisk deletes the file associated with a key
// from the on-disk cache directory.
func (arc *ARC) RemoveFromDisk(key string) {
	path := filepath.Join(arc.cacheDirectory, key)
	err := os.Remove(path)
	if err != nil {
		panic(err)
	}
}

// Len returns the number of bindings in the ARC cache.
func (arc *ARC) Len() int {
	return arc.t1List.Len() + arc.t2List.Len()
}

// Stats returns statistics about how many search hits and misses have occurred.
func (arc *ARC) Stats() *Stats {
	return &arc.stats
}

// min returns the lesser of ints x and y.
func min(x int, y int) int {
	if x < y {
		return x
	} else {
		return y
	}
}

// max returns the greater of ints x and y.
func max(x int, y int) int {
	if x > y {
		return x
	} else {
		return y
	}
}

package arc

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func init() {
	rand.Seed(time.Now().Unix())
}

// Computes hit ratio for accessing random entries in an ARC
func BenchmarkARC_Rand(b *testing.B) {
	l, err := NewARC(8192)
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	trace := make([]int64, b.N*2)
	for i := 0; i < b.N*2; i++ {
		trace[i] = rand.Int63() % 32768
	}

	b.ResetTimer()

	// var hit, miss int
	for i := 0; i < 2*b.N; i++ {
		s := fmt.Sprintf("%v", trace[i])
		if i%2 == 0 {
			b := make([]byte, 8)
			binary.LittleEndian.PutUint64(b, uint64(trace[i]))

			l.Set(s, b)
		} else {
			// _, ok := l.Get(s)
			// if ok {
			// 	hit++
			// } else {
			// 	miss++
			// }
			l.Get(s)
		}
	}
	b.Logf(fmt.Sprintf("%v", l.Stats()))
	hits := l.stats.Hits
	misses := l.stats.Misses
	// b.Logf(fmt.Sprintf("%f", l.Stats()[0] ))
	// b.Logf("hit: %d miss: %d ratio: %f", hit, miss, float64(hit)/float64(miss))
	b.Logf("hit: %d miss: %d ratio: %f", hits, misses, float64(hits)/float64(misses))
	// absolutePath, _ := filepath.Abs("./" + l.cacheDirectory)
	// os.RemoveAll(absolutePath)
}

// Compute hit ratio for a linear sequence of accesses
func BenchmarkARC_Freq(b *testing.B) {
	l, err := NewARC(8192)
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	trace := make([]int64, b.N*2)
	for i := 0; i < b.N*2; i++ {
		if i%2 == 0 {
			trace[i] = rand.Int63() % 16384
		} else {
			trace[i] = rand.Int63() % 32768
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b := make([]byte, 8)
		binary.LittleEndian.PutUint64(b, uint64(trace[i]))
		s := fmt.Sprintf("%v", trace[i])

		l.Set(s, b)
	}
	// var hit, miss int
	for i := 0; i < b.N; i++ {
		s := fmt.Sprintf("%v", trace[i])
		// _, ok := l.Get(s)
		// if ok {
		// 	hit++
		// } else {
		// 	miss++
		// }
		l.Get(s)
	}
	b.Logf(fmt.Sprintf("%v", l.Stats()))
	hits := l.stats.Hits
	misses := l.stats.Misses
	b.Logf("hit: %d miss: %d ratio: %f", hits, misses, float64(hits)/float64(misses))
	// b.Logf("hit: %d miss: %d ratio: %f", hit, miss, float64(hit)/float64(miss))
	// absolutePath, _ := filepath.Abs("./" + l.cacheDirectory)
	// os.RemoveAll(absolutePath)
}

func TestARC_RandomOps(t *testing.T) {
	size := 3
	l, err := NewARC(size)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	n := 20000
	// gfull := false
	for i := 0; i < n; i++ {
		key := rand.Int63() % 512
		s := fmt.Sprintf("%v", key)
		r := rand.Int63()
		t1 := l.t1List.Len()
		t2 := l.t2List.Len()
		b1 := l.b1List.Len()
		b2 := l.b2List.Len()
		// fmt.Print("t1")
		// fmt.Println(l.t1List.cache)
		// fmt.Print("t2")
		// fmt.Println(l.t2List.cache)
		// fmt.Print("b1")
		// fmt.Println(l.b1List.cache)
		// fmt.Print("b2")
		// fmt.Println(l.b2List.cache)
		//target := l.targetMarker
		switch r % 3 {
		case 0:
			// if gfull {
			// 	fmt.Println("add when full")
			// }
			b := make([]byte, 8)
			binary.LittleEndian.PutUint64(b, uint64(key))

			l.Set(s, b)
		case 1:
			l.Get(s)
		case 2:
			l.Remove(s)
		}

		// Printing cases for the purposes of debugging

		if l.t1List.Len()+l.b1List.Len() > size {
			rem := r % 3
			fmt.Print("case: ")
			fmt.Println(rem)
			fmt.Print("prev: ")
			fmt.Print(t1)
			fmt.Print(t2)
			fmt.Print(b1)
			fmt.Print(b2)
			t.Fatalf("bad l1: t1: %d t2: %d b1: %d b2: %d p: %d",
				l.t1List.Len(), l.t2List.Len(), l.b1List.Len(), l.b2List.Len(), l.targetMarker)
		}


		if l.t1List.Len()+l.t2List.Len() > size {
			rem := r % 3
			fmt.Print("case: ")
			fmt.Println(rem)
			fmt.Print("prev: ")
			fmt.Print(t1)
			fmt.Print(t2)
			fmt.Print(b1)
			fmt.Print(b2)
			t.Fatalf("bad t: t1: %d t2: %d b1: %d b2: %d p: %d",
				l.t1List.Len(), l.t2List.Len(), l.b1List.Len(), l.b2List.Len(), l.targetMarker)
		}

		if l.b1List.Len()+l.b2List.Len() > size {
			rem := r % 3
			fmt.Print("case: ")
			fmt.Println(rem)
			fmt.Print("prev: ")
			fmt.Print("t1")
			fmt.Println(l.t1List.cache)
			fmt.Print("t2")
			fmt.Println(l.t2List.cache)
			fmt.Print("b1")
			fmt.Println(l.b1List.cache)
			fmt.Print("b2")
			fmt.Println(l.b2List.cache)
			t.Fatalf("bad b: t1: %d t2: %d b1: %d b2: %d p: %d",
				l.t1List.Len(), l.t2List.Len(), l.b1List.Len(), l.b2List.Len(), l.targetMarker)
		}

		

		// cache list appends all keys into a string in sorted order, and compares the two list and checks that nothing was deleted that wasn't supposed to be
		// basically comparing the in-memory cache (lru list) and the disk-based cache (file path)
		// added elements of in-memory cache to disk-based cache (cannot fetch page from the disk)


		// var cachelist sort.StringSlice
		// var cachelists sort.StringSlice
		// for k, _ := range l.cache {
		// 	cachelist = append(cachelist, k)
		// }
		// cachelist.Sort()
		// for k, _ := range l.t1List.cache {
		// 	cachelists = append(cachelists, k)
		// }
		// for k, _ := range l.t2List.cache {
		// 	cachelists = append(cachelists, k)
		// }
		// for k, _ := range l.b1List.cache {
		// 	cachelists = append(cachelists, k)
		// }
		// for k, _ := range l.b2List.cache {
		// 	cachelists = append(cachelists, k)
		// }
		// cachelists.Sort()
		// same := true
		// for i, v := range cachelist {
		// 	if v != cachelists[i] {
		// 		same = false
		// 	}
		// }
		// if !same {
		// 	rem := r % 3
		// 	fmt.Print("case: ")
		// 	fmt.Println(rem)
		// 	fmt.Print("get: ")
		// 	fmt.Println(s)
		// 	fmt.Print("target")
		// 	fmt.Println(target)
		// 	fmt.Print("prev: ")
		// 	fmt.Print("t1")
		// 	fmt.Println(l.t1List.cache)
		// 	fmt.Print("t2")
		// 	fmt.Println(l.t2List.cache)
		// 	fmt.Print("b1")
		// 	fmt.Println(l.b1List.cache)
		// 	fmt.Print("b2")
		// 	fmt.Println(l.b2List.cache)
		// 	fmt.Print("cd")
		// 	fmt.Println(cachelist)
		// 	t.Fatalf("bad cache directory")
		// }

		// if l.t1List.Len() == 2 && l.t2List.Len() == 0 && l.t2List.Len() == 0 && l.b2List.Len() == 3 && l.targetMarker == 1 {
		// 	gfull = true
		// }
	}
	absolutePath, _ := filepath.Abs("./" + l.cacheDirectory)
	os.RemoveAll(absolutePath)
}

// Tests max entries and remaining spaces
func TestARC_Values(t *testing.T) {
	size := 3
	l, err := NewARC(size)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	
	
	if n := l.MaxEntries(); n != 3 {
		t.Fatalf("Max entries incorrect: %d", n)
	}

	// Fill t1
	for i := 0; i < 2; i++ {
		b := make([]byte, 8)
		binary.LittleEndian.PutUint64(b, uint64(i))
		s := fmt.Sprintf("%v", i)

		l.Set(s, b)
	}

	if m := l.RemainingSpaces(); m != 1 {
		t.Fatalf("Remaining incorrect: %d", m)
	}

}

// Tests Get() by accessing entries, then getting from t1 and t2
func TestARC_Get_RecentToFrequent(t *testing.T) {
	l, err := NewARC(128)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Touch all the entries, should be in t1
	for i := 0; i < 128; i++ {
		b := make([]byte, 8)
		binary.LittleEndian.PutUint64(b, uint64(i))
		s := fmt.Sprintf("%v", i)

		l.Set(s, b)
	}
	if n := l.t1List.Len(); n != 128 {
		t.Fatalf("Wrong length for t1: %d", n)
	}
	if n := l.t2List.Len(); n != 0 {
		t.Fatalf("Wrong length for t2: %d", n)
	}

	// Get should upgrade to t2
	for i := 0; i < 128; i++ {
		s := fmt.Sprintf("%v", i)
		_, ok := l.Get(s)
		if !ok {
			t.Fatalf("missing: %d", i)
		}
	}
	if n := l.t1List.Len(); n != 0 {
		t.Fatalf("Wrong length for t1: %d", n)
	}
	if n := l.t2List.Len(); n != 128 {
		t.Fatalf("Wrong length for t2: %d", n)
	}

	// Get the entry you just added from t2
	for i := 0; i < 128; i++ {
		s := fmt.Sprintf("%v", i)
		_, ok := l.Get(s)
		if !ok {
			t.Fatalf("missing: %d", i)
		}
	}
	if n := l.t1List.Len(); n != 0 {
		t.Fatalf("Wrong length for t1: %d", n)
	}
	if n := l.t2List.Len(); n != 128 {
		t.Fatalf("Wrong length for t2: %d", n)
	}
	absolutePath, _ := filepath.Abs("./" + l.cacheDirectory)
	os.RemoveAll(absolutePath)
}

// Tests remove by inserting entries then checking length after removal
func TestARC_Remove(t *testing.T) {
	size := 3
	l, err := NewARC(size)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	// Fill t1
	for i := 0; i < 3; i++ {
		b := make([]byte, 8)
		binary.LittleEndian.PutUint64(b, uint64(i))
		s := fmt.Sprintf("%v", i)

		l.Set(s, b)
	}

	for i := 0; i < 2; i++ {
		s := fmt.Sprintf("%v", i)

		l.Remove(s)
	}
	
	if n := l.t1List.Len(); n != 1 {
		t.Fatalf("Incorrect length after removal: %d", n)
	}

}

// Tests Set() by setting value of entries in t1 then t2
func TestARC_Set_RecentToFrequent(t *testing.T) {
	l, err := NewARC(128)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Set initially to t1
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(1))
	s := fmt.Sprintf("%v", 1)

	l.Set(s, b)
	if n := l.t1List.Len(); n != 1 {
		t.Fatalf("bad: %d", n)
	}
	if n := l.t2List.Len(); n != 0 {
		t.Fatalf("bad: %d", n)
	}

	// Set should upgrade to t2
	b = make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(1))
	l.Set(s, b)
	if n := l.t1List.Len(); n != 0 {
		t.Fatalf("bad: %d", n)
	}
	if n := l.t2List.Len(); n != 1 {
		t.Fatalf("bad: %d", n)
	}

	// Set should remain in t2
	b = make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(1))
	l.Set(s, b)
	if n := l.t1List.Len(); n != 0 {
		t.Fatalf("bad: %d", n)
	}
	if n := l.t2List.Len(); n != 1 {
		t.Fatalf("bad: %d", n)
	}
	absolutePath, _ := filepath.Abs("./" + l.cacheDirectory)
	os.RemoveAll(absolutePath)
}


// Tests migration entries between t1, t2, b1 and b2 based on ARC policy
func TestARC_Adaptive(t *testing.T) {
	l, err := NewARC(4)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Fill t1
	for i := 0; i < 4; i++ {
		b := make([]byte, 8)
		binary.LittleEndian.PutUint64(b, uint64(i))
		s := fmt.Sprintf("%v", i)

		l.Set(s, b)
	}
	if n := l.t1List.Len(); n != 4 {
		t.Fatalf("bad: %d", n)
	}

	// Move to t2
	s := fmt.Sprintf("%v", 0)
	l.Get(s)
	s = fmt.Sprintf("%v", 1)
	l.Get(s)
	if n := l.t2List.Len(); n != 2 {
		t.Fatalf("bad: %d", n)
	}

	// Evicts from t1
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(4))
	s = fmt.Sprintf("%v", 4)
	l.Set(s, b)
	if n := l.b1List.Len(); n != 1 {
		t.Fatalf("bad: %d", n)
	}

	// Current state
	// t1 : (MRU) [4, 3] (LRU)
	// t2 : (MRU) [1, 0] (LRU)
	// b1 : (MRU) [2] (LRU)
	// b2 : (MRU) [] (LRU)

	// Set 2, should cause hit on b1
	b = make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(2))
	s = fmt.Sprintf("%v", 2)
	l.Set(s, b)
	if n := l.b1List.Len(); n != 1 {
		t.Fatalf("bad: %d", n)
	}
	if l.targetMarker != 1 {
		t.Fatalf("bad: %d", l.targetMarker)
	}
	if n := l.t2List.Len(); n != 3 {
		t.Fatalf("bad: %d", n)
	}

	// Current state
	// t1 : (MRU) [4] (LRU)
	// t2 : (MRU) [2, 1, 0] (LRU)
	// b1 : (MRU) [3] (LRU)
	// b2 : (MRU) [] (LRU)

	// Set 4, should migrate to t2
	b = make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(4))
	s = fmt.Sprintf("%v", 4)
	l.Set(s, b)
	if n := l.t1List.Len(); n != 0 {
		t.Fatalf("bad: %d", n)
	}
	if n := l.t2List.Len(); n != 4 {
		t.Fatalf("bad: %d", n)
	}

	// Current state
	// t1 : (MRU) [] (LRU)
	// t2 : (MRU) [4, 2, 1, 0] (LRU)
	// b1 : (MRU) [3] (LRU)
	// b2 : (MRU) [] (LRU)

	// Set 4, should evict to b2
	b = make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(5))
	s = fmt.Sprintf("%v", 5)
	l.Set(s, b)
	if n := l.t1List.Len(); n != 1 {
		t.Fatalf("bad: %d", n)
	}
	if n := l.t2List.Len(); n != 3 {
		t.Fatalf("bad: %d", n)
	}
	if n := l.b2List.Len(); n != 1 {
		t.Fatalf("bad: %d", n)
	}

	// Current state
	// t1 : (MRU) [5] (LRU)
	// t2 : (MRU) [4, 2, 1] (LRU)
	// b1 : (MRU) [3] (LRU)
	// b2 : (MRU) [0] (LRU)

	// Set 0, should decrease p
	b = make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(0))
	s = fmt.Sprintf("%v", 0)
	l.Set(s, b)
	if n := l.t1List.Len(); n != 0 {
		t.Fatalf("bad: %d", n)
	}
	if n := l.t2List.Len(); n != 4 {
		t.Fatalf("bad: %d", n)
	}
	if n := l.b1List.Len(); n != 2 {
		t.Fatalf("bad: %d", n)
	}
	if n := l.b2List.Len(); n != 0 {
		t.Fatalf("bad: %d", n)
	}
	if l.targetMarker != 0 {
		t.Fatalf("bad: %d", l.targetMarker)
	}

	fmt.Println(l.Stats())

	// Current state
	// t1 : (MRU) [] (LRU)
	// t2 : (MRU) [0, 4, 2, 1] (LRU)
	// b1 : (MRU) [5, 3] (LRU)
	// b2 : (MRU) [0] (LRU)
	absolutePath, _ := filepath.Abs("./" + l.cacheDirectory)
	os.RemoveAll(absolutePath)
}

// Test Set(), Get(), and Remove() in sequence
func TestARC_Sequence(t *testing.T) {
	l, err := NewARC(128)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	for i := 0; i < 256; i++ {
		b := make([]byte, 8)
		binary.LittleEndian.PutUint64(b, uint64(i))
		s := fmt.Sprintf("%v", i)
		l.Set(s, b)
	}
	if l.Len() != 128 {
		t.Fatalf("bad len: %v", l.Len())
	}

	for i := 0; i < 128; i++ {
		s := fmt.Sprintf("%v", i)
		_, ok := l.Get(s)
		if ok {
			t.Fatalf("should be evicted")
		}
	}
	for i := 128; i < 256; i++ {
		s := fmt.Sprintf("%v", i)
		_, ok := l.Get(s)
		if !ok {
			t.Fatalf("should not be evicted")
		}
	}
	for i := 128; i < 192; i++ {
		s := fmt.Sprintf("%v", i)
		l.Remove(s)
		_, ok := l.Get(s)
		if ok {
			t.Fatalf("should be deleted")
		}
	}
	absolutePath, _ := filepath.Abs("./" + l.cacheDirectory)
	os.RemoveAll(absolutePath)
}


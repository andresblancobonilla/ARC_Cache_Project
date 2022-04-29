package cache

import (
	"encoding/binary"
	"fmt"
	"math/big"
	"math/rand"
	"testing"
	"time"
)

func init() {
	rand.Seed(time.Now().Unix())
}

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

	var hit, miss int
	for i := 0; i < 2*b.N; i++ {
		s := fmt.Sprintf("%v", i)
		if i%2 == 0 {
			b := make([]byte, 8)
			binary.LittleEndian.PutUint64(b, uint64(i))

			l.Set(s, b)
		} else {
			_, ok := l.Get(s)
			if ok {
				hit++
			} else {
				miss++
			}
		}
	}
	b.Logf("hit: %d miss: %d ratio: %f", hit, miss, float64(hit)/float64(miss))
}

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
		binary.LittleEndian.PutUint64(b, uint64(i))
		s := fmt.Sprintf("%v", i)

		l.Set(s, b)
	}
	var hit, miss int
	for i := 0; i < b.N; i++ {
		s := fmt.Sprintf("%v", i)
		_, ok := l.Get(s)
		if ok {
			hit++
		} else {
			miss++
		}
	}
	b.Logf("hit: %d miss: %d ratio: %f", hit, miss, float64(hit)/float64(miss))
}

func TestARC_RandomOps(t *testing.T) {
	size := 128
	l, err := NewARC(128)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	n := 200000
	for i := 0; i < n; i++ {
		key := rand.Int63() % 512
		s := fmt.Sprintf("%v", key)
		r := rand.Int63()
		switch r % 3 {
		case 0:
			b := make([]byte, 8)
			binary.LittleEndian.PutUint64(b, uint64(key))
			l.Set(s, b)
		case 1:
			l.Get(s)
		case 2:
			l.Remove(s)
		}

		if l.t1List.Len()+l.t2List.Len() > size {
			t.Fatalf("bad t: t1: %d t2: %d b1: %d b2: %d p: %d",
				l.t1List.Len(), l.t2List.Len(), l.b1List.Len(), l.b2List.Len(), l.targetMarker)
		}
		if l.b1List.Len()+l.b2List.Len() > size {
			t.Fatalf("bad b: t1: %d t2: %d b1: %d b2: %d p: %d",
				l.t1List.Len(), l.t2List.Len(), l.b1List.Len(), l.b2List.Len(), l.targetMarker)
		}
	}
}

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
		t.Fatalf("bad: %d", n)
	}
	if n := l.t2List.Len(); n != 0 {
		t.Fatalf("bad: %d", n)
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
		t.Fatalf("bad: %d", n)
	}
	if n := l.t2List.Len(); n != 128 {
		t.Fatalf("bad: %d", n)
	}

	// Get be from t2
	for i := 0; i < 128; i++ {
		s := fmt.Sprintf("%v", i)
		_, ok := l.Get(s)
		if !ok {
			t.Fatalf("missing: %d", i)
		}
	}
	if n := l.t1List.Len(); n != 0 {
		t.Fatalf("bad: %d", n)
	}
	if n := l.t2List.Len(); n != 128 {
		t.Fatalf("bad: %d", n)
	}
}

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
}

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

	// Evict from t1
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

	// Current state
	// t1 : (MRU) [] (LRU)
	// t2 : (MRU) [0, 4, 2, 1] (LRU)
	// b1 : (MRU) [5, 3] (LRU)
	// b2 : (MRU) [0] (LRU)
}

func TestARC(t *testing.T) {
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

	for i, k := range l.cacheDirectory {
		kv := int(big.NewInt(0).SetBytes(k.bytes).Uint64())
		v, ok := l.Get(i)
		vv := int(big.NewInt(0).SetBytes(v).Uint64())
		if !ok || vv != kv {
			t.Fatalf("bad key: %v", k)
		}
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
}

package arc

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func init() {
	rand.Seed(time.Now().Unix())
}

// Computes hit ratio for accessing random entries in an ARC
func BenchmarkLRU_Rand(b *testing.B) {
	l := NewLRU(8192)

	trace := make([]int64, b.N*2)
	for i := 0; i < b.N*2; i++ {
		trace[i] = rand.Int63() % 32768
	}

	b.ResetTimer()

	for i := 0; i < 2*b.N; i++ {
		s := fmt.Sprintf("%v", trace[i])
		if i%2 == 0 {
			b := make([]byte, 8)
			binary.LittleEndian.PutUint64(b, uint64(trace[i]))

			l.Set(s, b)
		} else {
			l.Get(s)
		}
	}
	hits := l.stats.Hits
	misses := l.stats.Misses
	b.Logf("hit: %d miss: %d ratio: %f", hits, misses, float64(hits)/float64(misses))
	
}

// Compute hit ratio for a linear sequence of accesses
func BenchmarkLRU_Freq(b *testing.B) {
	l := NewLRU(8192)

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
	for i := 0; i < b.N; i++ {
		s := fmt.Sprintf("%v", trace[i])
		l.Get(s)
	}
	hits := l.stats.Hits
	misses := l.stats.Misses
	b.Logf("hit: %d miss: %d ratio: %f", hits, misses, float64(hits)/float64(misses))
	
}

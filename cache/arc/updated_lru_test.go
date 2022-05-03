package arc

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	// "os"
	// "path/filepath"
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

	var hit, miss int
	for i := 0; i < 2*b.N; i++ {
		s := fmt.Sprintf("%v", trace[i])
		if i%2 == 0 {
			b := make([]byte, 8)
			binary.LittleEndian.PutUint64(b, uint64(trace[i]))

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
	b.Logf(fmt.Sprintf("%v", l.Stats()))
	b.Logf("hit: %d miss: %d ratio: %f", hit, miss, float64(hit)/float64(miss))
	// absolutePath, _ := filepath.Abs("./" + l.cacheDirectory)
	// os.RemoveAll(absolutePath)
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
	var hit, miss int
	for i := 0; i < b.N; i++ {
		s := fmt.Sprintf("%v", trace[i])
		_, ok := l.Get(s)
		if ok {
			hit++
		} else {
			miss++
		}
	}
	b.Logf(fmt.Sprintf("%v", l.Stats()))
	b.Logf("hit: %d miss: %d ratio: %f", hit, miss, float64(hit)/float64(miss))
	// absolutePath, _ := filepath.Abs("./" + l.cacheDirectory)
	// os.RemoveAll(absolutePath)
}

package bbuf

import (
	"math/bits"
	"math/rand"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLenNs(t *testing.T) {
	for l := 0; l < 1000; l++ {
		n := lenN(l)
		size := nSize(n)
		if size < l {
			t.Errorf("Not enough size for l=%d, n=%d, size=%d", l, n, size)
		}
	}
	for n := 0; n < poolsCount; n++ {
		size := nSize(n)
		lenN := lenN(size)
		if lenN != n {
			t.Errorf("For bucket %d size is %d, but for this size bucket is %d", n, size, lenN)
		}
	}
}

func Benchmark_lenN(b *testing.B) {
	n := 0
	_ = n
	b.Run("simple 2x", func(b *testing.B) {
		for i := 0; i < b.N/10000; i++ {
			for l := 0; l < 10000; l++ {
				if l == 0 {
					n = 0
					continue
				}
				n = bits.Len(uint(l) - 1)
			}
		}
	})
	b.Run("flatter 1.4x", func(b *testing.B) {
		for i := 0; i < b.N/10000; i++ {
			for l := 0; l < 10000; l++ {
				if l < 5 {
					n = 0
					continue
				}
				m := uint(l) - 1
				bits := bits.Len(m)
				if m>>(uint(bits)-2) == 2 {
					n = 2*bits - 5
				} else {
					n = 2*bits - 4
				}
			}
		}
	})
}

func Benchmark_NSize(b *testing.B) {
	size := 0
	_ = size
	b.Run("flat", func(b *testing.B) {
		for i := 0; i < b.N/32; i++ {
			for l := 0; l < 32; l++ {
				size = int(1 << uint(l))
			}
		}
	})
	b.Run("flatter x1.4", func(b *testing.B) {
		for i := 0; i < b.N/32; i++ {
			for l := 0; l < 32; l++ {
				if l%2 == 0 {
					size = 2 << uint(l/2)
				} else {
					size = 3 << uint(l/2)
				}
			}
		}
	})
}

func Benchmark_Buffer(b *testing.B) {
	bp := NewBuffersPool()
	rnds := make([]int, 1000)
	for i := range rnds {
		rnds[i] = rand.Intn(10000) + 1
	}
	b.ResetTimer()
	b.Run("use buffer pool", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			size := rnds[i%1000]
			buf := bp.Get(size)
			buf[0] = 1
			bp.Put(&buf)
		}
	})
	b.Run("use buffer pool concurrently", func(b *testing.B) {
		wg := sync.WaitGroup{}
		const goN = 4
		wg.Add(goN)
		start := make(chan struct{})
		for g := 0; g < goN; g++ {
			g := g
			go func() {
				<-start
				for i := 0; i < b.N/goN; i++ {
					size := rnds[(i+g*100)%1000]
					buf := bp.Get(size)
					buf[0] = 1
					bp.Put(&buf)
				}
				wg.Done()
			}()
		}
		b.ResetTimer()
		close(start)
		wg.Wait()
	})
	b.Run("return part bad", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			size := rnds[i%1000]
			buf := bp.Get(size)
			subpart := buf[9:]
			bp.Put(&subpart)
		}
	})
	b.Run("copy part in other", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			size := 1025
			buf := bp.Get(size)
			subpart := bp.Get(size - 9)
			copy(subpart, buf[9:])
			bp.Put(&buf)
			bp.Put(&subpart)
		}
	})
	b.Run("use standard make", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			size := rand.Intn(10000) + 1
			buf := make([]byte, size)
			buf[0] = 1
		}
	})
}

func TestBytesBuf(t *testing.T) {
	got := Make(1024)
	require.Len(t, got, 1024)
	require.Equal(t, 1024, cap(got))
	back := got[:5]
	Put(&back)
	check := Make(1024)
	require.Len(t, check, 1024)
	require.Equal(t, 1024, cap(check))
}

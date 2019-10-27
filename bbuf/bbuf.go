package bbuf

import (
	"math/bits"
	"sync"
)

var defaultPool *BuffersPool

func init() {
	defaultPool = NewBuffersPool()
}

func Make(size int) []byte {
	return defaultPool.Get(size)
}

func Put(b *[]byte) {
	defaultPool.Put(b)
}

func Copy(b []byte) (cpy []byte) {
	return defaultPool.Copy(b)
}

const poolsCount = 64

type BuffersPool struct {
	pools [poolsCount]sync.Pool
}

func NewBuffersPool() *BuffersPool {
	bp := new(BuffersPool)
	for i := range bp.pools {
		size := nSize(i)
		bp.pools[i] = sync.Pool{New: func() interface{} {
			buf := make([]byte, size)
			return &buf
		}}
	}
	return bp
}

//faster version with bigger spread. 60ns vs 72ns for full get-put
//wider spread, so poolCount=32 is enough

//func nSize(n int) int{
//	return int(1<<uint(n))
//}
//
//func lenN(l int) int{
//	if l==0{
//		return 0
//	}
//	return bits.Len(uint(l)-1)
//}

func nSize(n int) int {
	if n == 0 {
		return 4
	} else if n%2 == 0 {
		return 2 << uint(n/2+1)
	} else {
		return 3 << uint(n/2+1)
	}
}

func lenN(l int) int {
	if l < 5 {
		return 0
	}
	m := uint(l) - 1
	bits := bits.Len(m)
	if m>>(uint(bits)-2) == 2 {
		return 2*bits - 5
	} else {
		return 2*bits - 4
	}
}

func (bp *BuffersPool) Get(size int) []byte {
	n := lenN(size)
	if n > poolsCount-1 {
		return make([]byte, size)
	}
	b := bp.pools[n].Get().(*[]byte)
	return (*b)[:size]
}

func (bp *BuffersPool) Put(b *[]byte) {
	if b == nil {
		return
	}
	c := cap(*b)
	if c == 0 {
		return
	}
	n := lenN(c)
	if n > poolsCount-1 {
		n = poolsCount - 1
	}
	if c < nSize(n) {
		if n == 0 {
			return
		}
		n--
	}
	bp.pools[n].Put(b)
}

func (bp *BuffersPool) Copy(b []byte) (cpy []byte) {
	cpy = bp.Get(len(b))
	copy(cpy, b)
	return cpy
}

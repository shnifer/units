package juggler

import (
	"sync"
	"testing"
)

type emptyStorage struct{}

func (e emptyStorage) Get(uint64) ([]byte, error) {
	return nil, nil
}

func (e emptyStorage) Put(uint64, []byte) error {
	return nil
}

func (e emptyStorage) Delete(uint64) error {
	return nil
}

func BenchmarkJuggler_no_conflicts(b *testing.B) {
	const threads = 8
	wg := &sync.WaitGroup{}
	wg.Add(threads)
	J := New(emptyStorage{})
	for fN := 0; fN < threads; fN++ {
		go func(fn int) {
			for i := 0; i < b.N/threads; i++ {
				id := uint64(fn)
				T := J.NewTx()
				_, err := T.Get(id)
				noErr(err)
				noErr(T.SetLimited())
				noErr(T.Put(id, nil))
				noErr(T.Finish())
			}
			wg.Done()
		}(fN)
	}
	wg.Wait()
}

func BenchmarkJuggler_all_conflicts(b *testing.B) {
	const threads = 8
	wg := &sync.WaitGroup{}
	wg.Add(threads)
	J := New(emptyStorage{})
	for fN := 0; fN < threads; fN++ {
		go func(fn int) {
			for i := 0; i < b.N/threads; i++ {
				id := uint64(1)
				T := J.NewTx()
				_, _ = T.Get(id)
				_ = T.SetLimited()
				_ = T.Put(id, nil)
				_ = T.Finish()
			}
			wg.Done()
		}(fN)
	}
	wg.Wait()
}

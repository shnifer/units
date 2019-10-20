package juggler

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type TestStorage struct {
	sync.Mutex
	data map[uint64][]byte
}

func NewTestStorage() *TestStorage {
	return &TestStorage{
		Mutex: sync.Mutex{},
		data:  make(map[uint64][]byte),
	}
}

func (s *TestStorage) Get(uid uint64) ([]byte, error) {
	s.Lock()
	defer s.Unlock()
	data, ok := s.data[uid]
	if !ok {
		return nil, errors.New("NOT FOUND")
	}
	ret := make([]byte, 8)
	copy(ret[:], data[len(data)-8:])
	return ret, nil
}

func (s *TestStorage) Put(uid uint64, data []byte) error {
	s.Lock()
	s.data[uid] = append(s.data[uid], data...)
	s.Unlock()
	return nil
}

func (s *TestStorage) Delete(uid uint64) error {
	s.Lock()
	delete(s.data, uid)
	s.Unlock()
	return nil
}

func TestFuzz_Juggler(t *testing.T) {
	Data := NewTestStorage()
	var zero = make([]byte, 16)
	noErr(Data.Put(0, zero))
	noErr(Data.Put(1, zero))
	noErr(Data.Put(2, zero))
	noErr(Data.Put(3, zero))

	wg := &sync.WaitGroup{}
	wg.Add(255)

	J := New(Data)
	for fN := 1; fN <= 255; fN++ {
		go testFunc(fN, J, wg, false)
	}
	wg.Wait()
	fmt.Println("Retry count", RetryCount)

	for id := 0; id < 4; id++ {
		rawData := Data.data[uint64(id)]
		i := 8
		cur := uint64(0)
		for i+16 <= len(rawData) {
			was := binary.BigEndian.Uint64(rawData[i : i+8])
			new := binary.BigEndian.Uint64(rawData[i+8 : i+16])
			if new != was {
				panic("broken chain")
			}
			if new < cur {
				panic("reverse chain")
			}
			cur = new
			i += 16
		}
	}
}

var RetryCount int32

func testFunc(fN int, J *Juggler, wg *sync.WaitGroup, withLimited bool) {
	defer wg.Done()
	for {
		err := testFuncErr(fN, J, withLimited)
		if err == nil {
			return
		}
		if err != ErrDropped {
			panic(err)
		}
		atomic.AddInt32(&RetryCount, 1)
	}
}

func testFuncErr(fN int, J *Juggler, withLimited bool) error {
	dat := make(map[uint64][]byte)
	var ids []uint64
	for i := 0; i < 4; i++ {
		if fN&(1<<i) > 0 {
			ids = append(ids, uint64(i))
		}
	}
	rand.Shuffle(len(ids), func(i, j int) {
		ids[i], ids[j] = ids[j], ids[i]
	})
	T := J.NewTx()
	for _, id := range ids {
		data, err := T.Get(id)
		if err != nil {
			return err
		}
		dat[id] = data
		randWait()
	}
	if withLimited {
		err := T.SetLimited()
		if err != nil {
			return err
		}
	}
	for k, v := range dat {
		sig := make([]byte, 8)
		binary.BigEndian.PutUint64(sig, T.proc.procN)
		dat[k] = append(v, sig...)
	}
	rand.Shuffle(len(ids), func(i, j int) {
		ids[i], ids[j] = ids[j], ids[i]
	})
	for _, id := range ids {
		err := T.Put(id, dat[id])
		if err != nil {
			return err
		}
		randWait()
	}
	err := T.Finish()
	if err != nil {
		return err
	}
	return nil
}

func randWait() {
	t := rand.Intn(10)
	if t == 0 {
		time.Sleep(time.Millisecond * time.Duration(rand.Intn(100)))
	} else if t < 2 {
		time.Sleep(time.Millisecond * time.Duration(rand.Intn(10)))
	} else {
		time.Sleep(time.Microsecond * time.Duration(rand.Intn(1000)))
	}
}

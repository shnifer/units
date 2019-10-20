package juggler

import (
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

func noErr(err error) {
	if err != nil {
		panic(err)
	}
}

func TestJuggler_NewProcess(t *testing.T) {
	J := newSequencer()
	var Ps []*process
	for i := 0; i < 10; i++ {
		Ps = append(Ps, J.NewProcess())
	}
	require.Len(t, J.procs, 10)
	for i := 0; i < 5; i++ {
		noErr(Ps[i].Finish())
	}
	require.Len(t, J.procs, 5)
}

func TestProcess(t *testing.T) {
	t.Run("get 2, save 1", func(t *testing.T) {
		J := newSequencer()
		P := J.NewProcess()
		noErr(P.Get(100))
		noErr(P.Get(200))
		require.Len(t, J.procs, 1)
		require.Len(t, J.procs[0].got, 2)
		require.False(t, J.procs[0].inCharge)

		noErr(P.Pretend())
		require.True(t, J.procs[0].inCharge)

		noErr(P.Change(200))
		noErr(P.Finish())
	})
	t.Run("get one, save another", func(t *testing.T) {
		J := newSequencer()
		P := J.NewProcess()
		noErr(P.Get(100))
		noErr(P.Pretend())
		noErr(P.Change(200))
		noErr(P.Finish())
	})
	t.Run("limited can't Get and Change new", func(t *testing.T) {
		J := newSequencer()
		P := J.NewProcess()
		noErr(P.Get(100))
		require.False(t, J.procs[0].limited)
		noErr(P.SetLimited())
		require.True(t, J.procs[0].limited)
		noErr(P.Get(100))
		err := P.Get(200)
		require.Equal(t, err, ErrLimited)
		noErr(P.Pretend())
		noErr(P.Change(100))
		err = P.Change(200)
		require.Equal(t, err, ErrLimited)
		noErr(P.Finish())
	})
	t.Run("finished", func(t *testing.T) {
		J := newSequencer()
		P := J.NewProcess()
		noErr(P.Finish())
		noErr(P.Finish())
		err := P.Get(1)
		require.Equal(t, err, ErrFinished)
		err = P.Pretend()
		require.Equal(t, err, ErrFinished)
		err = P.SetLimited()
		require.Equal(t, err, ErrFinished)
		err = P.Change(1)
		require.Equal(t, err, ErrFinished)
	})
	t.Run("not in charge", func(t *testing.T) {
		J := newSequencer()
		P := J.NewProcess()
		noErr(P.Get(1))
		err := P.Change(1)
		require.Equal(t, err, ErrNotInCharge)
	})
	t.Run("Dropped", func(t *testing.T) {
		J := newSequencer()
		P1 := J.NewProcess()
		P2 := J.NewProcess()
		noErr(P1.Get(1))
		noErr(P2.Get(1))
		noErr(P1.Pretend())
		noErr(P1.Change(1))
		err := P2.Get(2)
		require.Equal(t, err, ErrDropped)
		err = P2.Pretend()
		require.Equal(t, err, ErrDropped)
		err = P2.Change(1)
		require.Equal(t, err, ErrDropped)
		err = P2.SetLimited()
		require.Equal(t, err, ErrDropped)
		err = P2.Finish()
		require.Equal(t, err, nil) //we could finish dropped
	})
	t.Run("pretend waits", func(t *testing.T) {
		J := newSequencer()
		P1 := J.NewProcess()
		P2 := J.NewProcess()

		logDone := make(chan struct{})
		log := make(chan string, 1)
		var logs []string
		go func() {
			for l := range log {
				logs = append(logs, l)
			}
			close(logDone)
		}()

		wg := &sync.WaitGroup{}
		wg.Add(2)

		noErr(P1.Get(1))
		log <- "P1.Get"
		noErr(P2.Get(2))
		log <- "P2.Get"
		go func() {
			time.Sleep(time.Second / 10)
			log <- "P1.Finish"
			noErr(P1.Finish())
			wg.Done()
		}()
		go func() {
			log <- "P2.Pretend run"
			noErr(P2.Pretend())
			log <- "P2.Pretend done"
			wg.Done()
		}()
		wg.Wait()
		close(log)
		<-logDone
		require.Equal(t, logs, []string{
			"P1.Get",
			"P2.Get",
			"P2.Pretend run",
			"P1.Finish",
			"P2.Pretend done",
		})
	})
}

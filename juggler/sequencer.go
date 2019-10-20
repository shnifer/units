package juggler

import (
	"errors"
	"sync"
)

var ErrLimited = errors.New("limited process try to get new")
var ErrNotInCharge = errors.New("try to change while not in charge")
var ErrDropped = errors.New("process was dropped")
var ErrFinished = errors.New("process was finished")

type sequencer struct {
	cond     *sync.Cond
	curProcN uint64
	procs    procs
}

type procs []proc

type proc struct {
	procN    uint64
	limited  bool
	inCharge bool
	got      []uint64
}

func newSequencer() *sequencer {
	return &sequencer{
		cond: sync.NewCond(&sync.Mutex{}),
	}
}

//is not concurrent safe, one process for one goroutine or sync wrap
type process struct {
	juggler  *sequencer
	procN    uint64
	finished bool
}

func (j *sequencer) NewProcess() *process {
	j.cond.L.Lock()
	j.curProcN++
	procN := j.curProcN
	j.procs = append(j.procs, proc{
		procN: procN,
		got:   make([]uint64, 0, 2),
	})
	j.cond.L.Unlock()
	return &process{
		juggler: j,
		procN:   procN}
}

func (p *process) Get(uid uint64) error {
	if p.finished {
		return ErrFinished
	}
	return p.juggler.get(p.procN, uid)
}

func (j *sequencer) get(procN uint64, uid uint64) error {
	j.cond.L.Lock()
	defer j.cond.L.Unlock()
	iP, exist := j.procs.contains(procN)
	if !exist {
		return ErrDropped
	}
	proc := j.procs[iP]
	iG, ok := search(proc.got, uid)
	if ok {
		return nil
	}
	if proc.limited {
		return ErrLimited
	}
	insert(&j.procs[iP].got, iG, uid)
	return nil
}

func (p *process) SetLimited() error {
	if p.finished {
		return ErrFinished
	}
	return p.juggler.setLimited(p.procN)
}

func (j *sequencer) setLimited(procN uint64) error {
	j.cond.L.Lock()
	defer j.cond.L.Unlock()
	iP, exist := j.procs.contains(procN)
	if !exist {
		return ErrDropped
	}
	j.procs[iP].limited = true
	j.cond.Broadcast()
	return nil
}

func (p *process) Pretend() error {
	if p.finished {
		return ErrFinished
	}
	return p.juggler.pretend(p.procN)
}

func (j *sequencer) pretend(procN uint64) error {
	j.cond.L.Lock()
	defer j.cond.L.Unlock()
	for {
		iP, exist := j.procs.contains(procN)
		if !exist {
			return ErrDropped
		}
		if iP == 0 || j.procs[iP].inCharge || j.couldCharge(iP) {
			j.procs[iP].inCharge = true
			j.cond.Broadcast()
			return nil
		}
		j.cond.Wait()
	}
}

func (j *sequencer) couldCharge(iP int) bool {
	if !j.procs[iP].limited {
		return false
	}
	for i := 0; i < iP; i++ {
		if !j.procs[i].limited {
			return false
		}
		if hasIntersect(j.procs[iP].got, j.procs[i].got) {
			return false
		}
	}
	return true
}

func (p *process) Change(uid uint64) error {
	if p.finished {
		return ErrFinished
	}
	return p.juggler.change(p.procN, uid)
}

func (j *sequencer) change(procN uint64, uid uint64) error {
	j.cond.L.Lock()
	defer j.cond.L.Unlock()
	iP, exist := j.procs.contains(procN)
	if !exist {
		return ErrDropped
	}
	proc := j.procs[iP]
	if !proc.inCharge {
		return ErrNotInCharge
	}
	iG, ok := search(proc.got, uid)
	if !ok {
		if proc.limited {
			return ErrLimited
		}
		insert(&j.procs[iP].got, iG, uid)
	}
	for i := len(j.procs) - 1; i > iP; i-- {
		if hasIntersect(j.procs[iP].got, j.procs[i].got) {
			j.procs.delete(i)
		}
	}
	j.cond.Broadcast()
	return nil
}

func (p *process) Finish() error {
	if p.finished {
		return nil
	}
	p.finished = true

	return p.juggler.finish(p.procN)
}

func (j *sequencer) finish(procN uint64) error {
	j.cond.L.Lock()
	defer j.cond.L.Unlock()
	indP, exist := j.procs.contains(procN)
	if !exist {
		//if we come dropped here - means we never tried to change something
		return nil
	}
	j.procs.delete(indP)
	j.cond.Broadcast()
	return nil
}

func insert(xs *[]uint64, ind int, v uint64) {
	*xs = append(*xs, 0)
	copy((*xs)[ind+1:], (*xs)[ind:])
	(*xs)[ind] = v
}

func search(xs []uint64, v uint64) (int, bool) {
	i, j := 0, len(xs)
	for i < j {
		h := int(uint(i+j) >> 1)
		if xs[h] < v {
			i = h + 1
		} else {
			j = h
		}
	}
	return i, i < len(xs) && xs[i] == v
}

func hasIntersect(xs, ys []uint64) bool {
	i, j := 0, 0
	lx, ly := len(xs), len(ys)
	for i < lx && j < ly {
		if xs[i] == ys[j] {
			return true
		} else if xs[i] < ys[j] {
			i++
		} else {
			j++
		}
	}
	return false
}

func (ps procs) contains(procN uint64) (n int, ok bool) {
	for i := range ps {
		if ps[i].procN == procN {
			return i, true
		} else if ps[i].procN > procN {
			return 0, false
		}
	}
	return 0, false
}

func (ps *procs) delete(ind int) {
	copy((*ps)[ind:], (*ps)[ind+1:])
	*ps = (*ps)[:len(*ps)-1]
}

package juggler

type GetPutDeleter interface {
	Get(uint64) ([]byte, error)
	Put(uint64, []byte) error
	Delete(uint64) error
}

type Juggler struct {
	resources GetPutDeleter
	sequencer *sequencer
}

type Tx struct {
	resources GetPutDeleter
	proc      *process
}

func New(resources GetPutDeleter) *Juggler {
	return &Juggler{
		resources: resources,
		sequencer: newSequencer(),
	}
}

func (J *Juggler) NewTx() *Tx {
	return &Tx{
		resources: J.resources,
		proc:      J.sequencer.NewProcess(),
	}
}

func (T *Tx) Get(uid uint64) ([]byte, error) {
	err := T.proc.Get(uid)
	if err != nil {
		return nil, err
	}
	return T.resources.Get(uid)
}

func (T *Tx) Put(uid uint64, data []byte) error {
	err := T.proc.Pretend()
	if err != nil {
		return err
	}
	err = T.resources.Put(uid, data)
	if err != nil {
		return err
	}
	return T.proc.Change(uid)
}

func (T *Tx) Delete(uid uint64) error {
	err := T.proc.Pretend()
	if err != nil {
		return err
	}
	err = T.resources.Delete(uid)
	if err != nil {
		return err
	}
	return T.proc.Change(uid)
}

func (T *Tx) SetLimited() error {
	return T.proc.SetLimited()
}

func (T *Tx) Finish() error {
	return T.proc.Finish()
}

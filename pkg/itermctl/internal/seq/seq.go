package seq

import "sync"

// MessageId maintains the sequence of message IDs assigned to outgoing messages.
var MessageId = newSeq()

type seq struct {
	mx *sync.Mutex
	i  int64
}

func newSeq() *seq {
	return &seq{mx: &sync.Mutex{}, i: 0}
}

func (m *seq) Next() *int64 {
	m.mx.Lock()
	defer m.mx.Unlock()
	i := m.i
	i += 1
	m.i = i
	return &i
}

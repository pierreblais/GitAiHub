package activity

import "sync"

// Broker fans recorded changesets out to live subscribers (the SSE feed).
// A slow subscriber drops events rather than blocking ingestion; the browser
// reconnects and catches up from its last sequence.
type Broker struct {
	mu   sync.Mutex
	subs map[chan Changeset]struct{}
}

// NewBroker returns an empty broker.
func NewBroker() *Broker { return &Broker{subs: map[chan Changeset]struct{}{}} }

// Subscribe returns a channel of new changesets and a cancel function.
func (b *Broker) Subscribe() (<-chan Changeset, func()) {
	ch := make(chan Changeset, 64)
	b.mu.Lock()
	b.subs[ch] = struct{}{}
	b.mu.Unlock()
	return ch, func() {
		b.mu.Lock()
		if _, ok := b.subs[ch]; ok {
			delete(b.subs, ch)
			close(ch)
		}
		b.mu.Unlock()
	}
}

// Publish sends cs to every subscriber without blocking.
func (b *Broker) Publish(cs Changeset) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.subs {
		select {
		case ch <- cs:
		default:
		}
	}
}

// Subscribers returns the number of live subscribers.
func (b *Broker) Subscribers() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.subs)
}

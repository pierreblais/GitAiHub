package activity

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/pierreblais/gitaihub/api/internal/source"
)

type memStore struct {
	mu      sync.Mutex
	saved   []Changeset
	cursors map[string]uint64
}

func (m *memStore) Cursor(_ context.Context, repo string) (uint64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cursors[repo], nil
}
func (m *memStore) SaveChangeset(_ context.Context, cs *Changeset) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.saved = append(m.saved, *cs)
	if m.cursors == nil {
		m.cursors = map[string]uint64{}
	}
	m.cursors[cs.Repository] = cs.Sequence
	return nil
}
func (m *memStore) ListChangesets(context.Context, Filter) ([]Changeset, error) { return nil, nil }
func (m *memStore) Landings(context.Context, string, []string) (map[string]source.Landing, error) {
	return nil, nil
}
func (m *memStore) ListAgents(context.Context) ([]Agent, error)                   { return nil, nil }
func (m *memStore) GetAgent(context.Context, string) (*Agent, error)              { return nil, nil }
func (m *memStore) RepositoryStats(context.Context) (map[string]RepoStats, error) { return nil, nil }

type fakeUpstream struct {
	log   []*Changeset
	after []uint64
	mu    sync.Mutex
}

func (f *fakeUpstream) WatchChangesets(ctx context.Context, _ string, after uint64) (<-chan *Changeset, <-chan error) {
	f.mu.Lock()
	f.after = append(f.after, after)
	f.mu.Unlock()
	ch := make(chan *Changeset)
	errs := make(chan error, 1)
	go func() {
		defer close(ch)
		for _, cs := range f.log {
			if cs.Sequence <= after {
				continue
			}
			c := *cs
			select {
			case ch <- &c:
			case <-ctx.Done():
				return
			}
		}
		<-ctx.Done()
	}()
	return ch, errs
}

func (f *fakeUpstream) LandedCommits(_ context.Context, _, _, _, newOID string, _ int) ([]*source.Commit, error) {
	return []*source.Commit{{ID: newOID, Subject: "commit " + newOID}}, nil
}

func TestIngestorResumesFromCursor(t *testing.T) {
	store := &memStore{cursors: map[string]uint64{"demo": 1}}
	up := &fakeUpstream{log: []*Changeset{
		{Sequence: 1, Reference: "refs/heads/main", OldOID: "0", NewOID: "a", Operation: "land", Actor: "x", RecordTime: time.Now()},
		{Sequence: 2, Reference: "refs/heads/main", OldOID: "a", NewOID: "b", Operation: "land", Actor: "planner", RecordTime: time.Now()},
	}}
	broker := NewBroker()
	sub, cancelSub := broker.Subscribe()
	defer cancelSub()
	in := &Ingestor{Store: store, Upstream: up, Broker: broker}
	ctx, cancel := context.WithCancel(context.Background())
	in.Ensure(ctx, map[string]string{"demo": "demo.git"})
	in.Ensure(ctx, map[string]string{"demo": "demo.git"}) // idempotent
	select {
	case cs := <-sub:
		if cs.Sequence != 2 || cs.Commits[0].Agent != "planner" || cs.Kind != KindLand {
			t.Fatalf("unexpected changeset %+v", cs)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no changeset published")
	}
	cancel()
	in.Wait()
	if len(store.saved) != 1 || up.after[0] != 1 || len(up.after) != 1 {
		t.Fatalf("saved=%d after=%v", len(store.saved), up.after)
	}
}

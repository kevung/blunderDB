package direction

import (
	"context"
	"fmt"
	"sort"
	"time"
)

// memStore is a Store held in memory, for the package's own tests. It enforces the one rule the
// real backends must also enforce: a sequence number is written once and never overwritten.
type memStore struct {
	recs   map[int64]Record
	events map[int64][]StoredEvent
	// failAt makes AppendEvent fail at a chosen sequence number, so a test can cut the power
	// between two decisions.
	failAt int
}

func newMemStore() *memStore {
	return &memStore{recs: map[int64]Record{}, events: map[int64][]StoredEvent{}, failAt: -1}
}

func (m *memStore) GetDirection(_ context.Context, id int64) (Record, error) {
	r, ok := m.recs[id]
	if !ok {
		return Record{}, ErrNoDirection
	}
	return r, nil
}

func (m *memStore) ListDirections(context.Context) ([]Record, error) {
	out := make([]Record, 0, len(m.recs))
	for _, r := range m.recs {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TournamentID < out[j].TournamentID })
	return out, nil
}

func (m *memStore) CreateDirection(_ context.Context, rec Record) error {
	if _, ok := m.recs[rec.TournamentID]; ok {
		return fmt.Errorf("already directed")
	}
	rec.CreatedAt = time.Now()
	m.recs[rec.TournamentID] = rec
	return nil
}

func (m *memStore) UpdateDirection(_ context.Context, rec Record) error {
	if _, ok := m.recs[rec.TournamentID]; !ok {
		return ErrNoDirection
	}
	m.recs[rec.TournamentID] = rec
	return nil
}

func (m *memStore) DeleteDirection(_ context.Context, id int64) error {
	delete(m.recs, id)
	delete(m.events, id)
	return nil
}

func (m *memStore) AppendEvent(_ context.Context, id int64, se StoredEvent) error {
	if m.failAt >= 0 && se.Seq >= m.failAt {
		return fmt.Errorf("simulated write failure at sequence %d", se.Seq)
	}
	for _, e := range m.events[id] {
		if e.Seq == se.Seq {
			// The log is append-only: a second write at the same sequence number is a bug,
			// not an update.
			return fmt.Errorf("sequence %d already written", se.Seq)
		}
	}
	m.events[id] = append(m.events[id], se)
	return nil
}

func (m *memStore) LoadEvents(_ context.Context, id int64) ([]StoredEvent, error) {
	out := append([]StoredEvent(nil), m.events[id]...)
	sort.Slice(out, func(i, j int) bool { return out[i].Seq < out[j].Seq })
	return out, nil
}

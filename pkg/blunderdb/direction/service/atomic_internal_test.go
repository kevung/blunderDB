package service

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// failingAppends fails the failAt-th event a transaction appends: a disk that fills up in the
// middle of a batch.
type failingAppends struct {
	storage.Storage
	n, failAt *int
}

func (f failingAppends) BeginTx(ctx context.Context) (storage.Tx, error) {
	tx, err := f.Storage.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	return failingTx{Tx: tx, n: f.n, failAt: f.failAt}, nil
}

type failingTx struct {
	storage.Tx
	n, failAt *int
}

func (t failingTx) Directions() storage.DirectionStore {
	return failingDirections{DirectionStore: t.Tx.Directions(), n: t.n, failAt: t.failAt}
}

type failingDirections struct {
	storage.DirectionStore
	n, failAt *int
}

func (d failingDirections) AppendEvent(ctx context.Context, scope string, tournamentID int64, ev direction.StoredEvent) error {
	*d.n++
	if *d.n == *d.failAt {
		return errors.New("disk full")
	}
	return d.DirectionStore.AppendEvent(ctx, scope, tournamentID, ev)
}

// TestConfirmAllProposals_AllOrNothing: a batch that fails on its second match writes none —
// not its first match either.
func TestConfirmAllProposals_AllOrNothing(t *testing.T) {
	ctx := context.Background()
	raw, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "d.db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = raw.Close() })
	tid, err := raw.Tournaments().Create(ctx, "", "Open", "2026-10-01", "")
	if err != nil {
		t.Fatal(err)
	}
	setup := New(raw, "", nil)
	if err := setup.CreateDirection(ctx, tid, `{"name":"Open","tables":{"count":4},"phases":[
		{"kind":"swiss_lives","length":7,"lives":2,"mode":"continuous"}]}`, 7); err != nil {
		t.Fatal(err)
	}
	if err := setup.EnterParticipants(ctx, tid, `[{"id":"a","name":"A"},{"id":"b","name":"B"},
		{"id":"c","name":"C"},{"id":"d","name":"D"},{"id":"e","name":"E"},{"id":"f","name":"F"}]`); err != nil {
		t.Fatal(err)
	}
	before, _, err := raw.Directions().EventsHead(ctx, "", tid)
	if err != nil {
		t.Fatal(err)
	}
	n, failAt := 0, 2
	svc := New(failingAppends{Storage: raw, n: &n, failAt: &failAt}, "", nil)
	if _, err := svc.ConfirmAllProposals(ctx, tid); err == nil {
		t.Fatalf("the batch succeeded although its second event failed (%d appended)", n)
	}
	if n < 2 {
		t.Fatalf("%d events appended; the batch needs at least two for the test to mean anything", n)
	}
	after, _, err := raw.Directions().EventsHead(ctx, "", tid)
	if err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Errorf("the log went from %d to %d events: the failed batch left part of itself", before, after)
	}
}

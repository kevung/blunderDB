package service

import (
	"context"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/events"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// heard records what is published; wants is what Wants answers.
type heard struct {
	got   []events.Event
	wants bool
	asked int
}

func (h *heard) Publish(ev events.Event) { h.got = append(h.got, ev) }

func (h *heard) Wants(string) bool { h.asked++; return h.wants }

// failingCommit refuses every commit: a connection lost at the last moment.
type failingCommit struct{ storage.Storage }

func (f failingCommit) BeginTx(ctx context.Context) (storage.Tx, error) {
	tx, err := f.Storage.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	return commitFails{Tx: tx}, nil
}

type commitFails struct{ storage.Tx }

func (c commitFails) Commit() error {
	_ = c.Tx.Rollback()
	return errors.New("connection lost")
}

func directed(t *testing.T) (storage.Storage, int64) {
	t.Helper()
	ctx := context.Background()
	st, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	tid, err := st.Tournaments().Create(ctx, "", "Open", "2026-10-01", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := New(st, "", nil).CreateDirection(ctx, tid, `{"name":"Open","tables":{"count":4},"phases":[
		{"kind":"swiss_lives","length":7,"lives":2,"mode":"continuous"}]}`, 7); err != nil {
		t.Fatal(err)
	}
	return st, tid
}

// TestAbortedGestureIsNotPublished: a gesture whose nested transaction rolled back after a
// partial write commits nothing, and publishes nothing.
func TestAbortedGestureIsNotPublished(t *testing.T) {
	ctx := context.Background()
	st, tid := directed(t)
	h := &heard{wants: true}
	mem := &Memory{}
	mem.SetPublisher(h)
	g, end, _, err := New(st, "", mem).lockGesture(ctx, gestureTarget{tournamentID: tid}, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.AddDirectionNote(ctx, tid, "à moitié"); err != nil {
		t.Fatalf("partial write: %v", err)
	}
	nested, err := g.st.BeginTx(ctx)
	if err != nil {
		t.Fatal(err)
	}
	_ = nested.Rollback()
	var gerr error
	end(&gerr)
	if !errors.Is(gerr, errGestureAborted) {
		t.Fatalf("end: %v; want errGestureAborted", gerr)
	}
	if len(h.got) != 0 {
		t.Fatalf("an aborted gesture was published: %+v", h.got)
	}
}

// TestFailedCommitIsNotPublished: a commit that fails publishes nothing.
func TestFailedCommitIsNotPublished(t *testing.T) {
	ctx := context.Background()
	st, tid := directed(t)
	h := &heard{wants: true}
	mem := &Memory{}
	mem.SetPublisher(h)
	if _, err := New(failingCommit{st}, "", mem).AddDirectionNote(ctx, tid, "perdue"); err == nil {
		t.Fatal("the gesture succeeded although its commit failed")
	}
	if len(h.got) != 0 {
		t.Fatalf("a failed commit was published: %+v", h.got)
	}
}

// TestNoListenerNoPublication: when no one listens to the scope, the gesture reads nothing more
// and publishes nothing.
func TestNoListenerNoPublication(t *testing.T) {
	ctx := context.Background()
	st, tid := directed(t)
	h := &heard{}
	mem := &Memory{}
	mem.SetPublisher(h)
	if _, err := New(st, "", mem).AddDirectionNote(ctx, tid, "seule"); err != nil {
		t.Fatal(err)
	}
	if len(h.got) != 0 || h.asked == 0 {
		t.Fatalf("published %d events after %d questions; want none after asking", len(h.got), h.asked)
	}
}

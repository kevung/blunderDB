package database

import (
	"errors"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// The desktop lists the roll's plays as the daemon does: every legal play,
// ranked, each index the one select_candidate names.
func TestTranscriptionCandidates_EveryPlayRanked(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	state, err := db.CreateTranscription(transcript.Header{MatchLength: 7})
	if err != nil {
		t.Fatalf("CreateTranscription: %v", err)
	}
	if _, err := db.TranscriptionCandidates(state.ID); !errors.Is(err, storage.ErrInvalid) {
		t.Fatalf("no roll typed: err = %v, want ErrInvalid", err)
	}
	for _, die := range []int{1, 2} {
		if state, err = db.ApplyTranscriptionGesture(state.ID, transcript.Gesture{Kind: transcript.GestureEnterDie, Die: die}); err != nil {
			t.Fatalf("enter die: %v", err)
		}
	}
	got, err := db.TranscriptionCandidates(state.ID)
	if err != nil {
		t.Fatalf("TranscriptionCandidates: %v", err)
	}
	legal := 0
	db.transcriptService().WithEditor("", state.ID, func(ed *transcript.Editor) { legal = len(transcript.Candidates(ed.Doc)) })
	if got.Unranked || len(got.Candidates) != legal || got.Dice != [2]int{2, 1} {
		t.Fatalf("unranked=%v dice=%v, %d candidates, want ranked 2-1 and %d", got.Unranked, got.Dice, len(got.Candidates), legal)
	}
	for i, c := range got.Candidates {
		if c.Rank != i+1 || c.Analysis == nil || c.Index < 0 || c.Index >= legal {
			t.Fatalf("candidate %d = %+v, want rank %d with its analysis", i, c, i+1)
		}
	}
}

// A keystroke's ranking never queues behind a batch that holds every core:
// it is a serial 0-ply search of about a millisecond, run under d.mu.RLock,
// where waiting for a token would stall the panel for a whole deep position.
// Not parallel: it takes the whole process budget while it runs.
func TestTranscriptionCandidates_NeverWaitsForACore(t *testing.T) {
	db := newTestDB(t)
	state, err := db.CreateTranscription(transcript.Header{MatchLength: 7})
	if err != nil {
		t.Fatalf("CreateTranscription: %v", err)
	}
	for _, die := range []int{3, 1} {
		if _, err = db.ApplyTranscriptionGesture(state.ID, transcript.Gesture{Kind: transcript.GestureEnterDie, Die: die}); err != nil {
			t.Fatalf("enter die: %v", err)
		}
	}
	for range gammonnet.ProcessCores().Cap() {
		release := gammonnet.HoldCore()
		defer release()
	}
	done := make(chan error, 1)
	go func() {
		_, err := db.TranscriptionCandidates(state.ID)
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("TranscriptionCandidates: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("TranscriptionCandidates waited for a core while a batch held them all")
	}
}

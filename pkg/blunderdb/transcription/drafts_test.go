package transcription

import (
	"errors"
	"fmt"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// Only what the document's state refuses is the caller's error; a failure
// stays one, so a client can tell a gesture to correct from an engine down.
func TestRefusedGestureClassifiesOnlyRefusals(t *testing.T) {
	for _, err := range []error{transcript.ErrNoEntry, transcript.ErrNoDice, transcript.ErrNoCandidate,
		transcript.ErrNoAction, transcript.ErrMatchOver, &transcript.Refusal{Kind: "illegal", Detail: "no such play"},
		fmt.Errorf("apply: %w", transcript.ErrNoDice), applyError(t, transcript.Gesture{Kind: "nope"}),
		applyError(t, transcript.Gesture{Kind: transcript.GestureEnterDie, Die: 9}),
		applyError(t, transcript.Gesture{Kind: transcript.GestureEnterDie, Die: 0})} {
		got := refusedGesture(err)
		if !errors.Is(got, storage.ErrInvalid) || !errors.Is(got, err) {
			t.Errorf("refusedGesture(%v) = %v; want a storage.ErrInvalid that keeps its cause", err, got)
		}
	}
	for _, err := range []error{errors.New("disk on fire"), transcript.ErrNotPure} {
		if got := refusedGesture(err); errors.Is(got, storage.ErrInvalid) || !errors.Is(got, err) {
			t.Errorf("refusedGesture(%v) = %v; want it unchanged, a failure", err, got)
		}
	}
}

// applyError is the error transcript.Apply answers g with on an empty document.
func applyError(t *testing.T, g transcript.Gesture) error {
	t.Helper()
	_, err := transcript.Apply(transcript.Document{}, g)
	if err == nil {
		t.Fatalf("Apply(%+v) on an empty document succeeded; want an error to classify", g)
	}
	return err
}

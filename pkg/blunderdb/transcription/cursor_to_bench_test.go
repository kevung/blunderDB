package transcription

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// openedTournamentMatch is a session on a whole tournament match, the Cursor at
// the end of the document, where a click on an early cell starts.
func openedTournamentMatch(b *testing.B) (*Service, *State) {
	b.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "test.mat"))
	if err != nil {
		b.Fatal(err)
	}
	svc := New(newStore(&testing.T{}), Options{})
	st, err := svc.Import(context.Background(), "", string(raw))
	if err != nil {
		b.Fatal(err)
	}
	return svc, st
}

func apply(b *testing.B, svc *Service, st *State, g transcript.Gesture) *State {
	next, err := svc.Apply(context.Background(), "", st.ID, Expect{Session: st.SessionID, Revision: st.Revision}, g)
	if err != nil {
		b.Fatal(err)
	}
	return next
}

// BenchmarkClickFarCellWalk and BenchmarkClickFarCellJump measure a click from
// the end of a tournament match onto its first cell, as the panel's binding
// sends it: one Service.Apply per step of the walk, against one in all.
func BenchmarkClickFarCellWalk(b *testing.B) {
	svc, st := openedTournamentMatch(b)
	end := len(st.Annotated.Document.Actions)
	for b.Loop() {
		for st.Annotated.Cursor > 0 {
			st = apply(b, svc, st, transcript.Gesture{Kind: transcript.GestureCursorBack})
		}
		b.StopTimer()
		st = apply(b, svc, st, transcript.Gesture{Kind: transcript.GestureCursorTo, At: end})
		b.StartTimer()
	}
}

func BenchmarkClickFarCellJump(b *testing.B) {
	svc, st := openedTournamentMatch(b)
	end := len(st.Annotated.Document.Actions)
	for b.Loop() {
		st = apply(b, svc, st, transcript.Gesture{Kind: transcript.GestureCursorTo, At: 0})
		b.StopTimer()
		st = apply(b, svc, st, transcript.Gesture{Kind: transcript.GestureCursorTo, At: end})
		b.StartTimer()
	}
}

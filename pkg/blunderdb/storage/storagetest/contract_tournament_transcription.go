package storagetest

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcription"
)

// testTournamentFinishedTranscriptionIsFiled pins where a finished
// transcription lands: under the tournament its header names, else under its
// event's tournament as an import would be; never detached from a tournament
// written for it, and never beside an empty one named after its event.
func testTournamentFinishedTranscriptionIsFiled(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	svc := transcription.New(s, transcription.Options{})
	finish := func(h transcript.Header) int64 {
		t.Helper()
		st, err := svc.Create(ctx, "", h)
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		for _, g := range []transcript.Gesture{
			{Kind: transcript.GestureEnterDie, Die: 6},
			{Kind: transcript.GestureEnterDie, Die: 3},
			{Kind: transcript.GestureSelectCandidate, Candidate: 0},
			{Kind: transcript.GestureValidate},
		} {
			if st, err = svc.Apply(ctx, "", st.ID, transcription.Expect{Session: st.SessionID, Revision: st.Revision}, g); err != nil {
				t.Fatalf("Apply %s: %v", g.Kind, err)
			}
		}
		res, err := svc.Finish(ctx, "", st.ID, transcription.Expect{Session: st.SessionID, Revision: st.Revision})
		if err != nil {
			t.Fatalf("Finish: %v", err)
		}
		return res.MatchID
	}
	filedIn := func(matchID int64) int64 {
		t.Helper()
		tr, err := s.Tournaments().TournamentOf(ctx, "", matchID)
		if err != nil || tr == nil {
			return 0
		}
		return tr.ID
	}
	names := func() map[string]int64 {
		out := map[string]int64{}
		for tr, err := range s.Tournaments().List(ctx, "", storage.ListOpts{}) {
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			out[tr.Name] = tr.ID
		}
		return out
	}

	byEvent := finish(transcript.Header{MatchLength: 9, Player1: "A", Player2: "B", Event: "Open"})
	if got, want := filedIn(byEvent), names()["Open"]; got == 0 || got != want {
		t.Fatalf("no tournament chosen: match in %d, want the event's (%d)", got, want)
	}

	chosen := names()["Open"]
	picked := finish(transcript.Header{MatchLength: 9, Player1: "A", Player2: "C", Event: "Open 2026", TournamentID: &chosen})
	if got := filedIn(picked); got != chosen {
		t.Fatalf("chosen tournament: match in %d, want %d", got, chosen)
	}
	if n := names(); len(n) != 1 {
		t.Fatalf("tournaments %v, want only the chosen one", n)
	}
}

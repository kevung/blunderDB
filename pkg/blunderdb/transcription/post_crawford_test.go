package transcription

import (
	"context"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// TestPostCrawfordSurvivesAMatchRoundTrip: a draft declared post-Crawford at
// 4-2 is saved with a live cube in its first game, and reopened from the Match
// it declares it again.
func TestPostCrawfordSurvivesAMatchRoundTrip(t *testing.T) {
	ctx := context.Background()
	svc := New(newStore(t), Options{})
	st, err := svc.Create(ctx, "1", transcript.Header{MatchLength: 5, Player1: "A", Player2: "B", Date: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	score := [2]int{4, 2}
	for _, g := range []transcript.Gesture{
		{Kind: transcript.GestureEnterDie, Die: 6},
		{Kind: transcript.GestureEnterDie, Die: 3},
		{Kind: transcript.GestureSelectCandidate, Candidate: 0},
		{Kind: transcript.GestureValidate},
		{Kind: transcript.GestureSetScore, At: 0, Score: &score},
		{Kind: transcript.GestureSetPostCrawford, PostCrawford: true},
		{Kind: transcript.GestureDouble},
		{Kind: transcript.GestureTake},
	} {
		if st, err = svc.Apply(ctx, "1", st.ID, Expect{Session: st.SessionID, Revision: st.Revision}, g); err != nil {
			t.Fatalf("%s: %v", g.Kind, err)
		}
	}
	// The declared 4-2 differs from the 0-0 no game gives; the cube is what is checked.
	if cube := st.Annotated.Actions[1:]; len(cube[0].Inconsistencies)+len(cube[1].Inconsistencies) > 0 {
		t.Fatalf("the cube is marked: %+v %+v", cube[0].Inconsistencies, cube[1].Inconsistencies)
	}
	res, err := svc.Finish(ctx, "1", st.ID, Expect{Session: st.SessionID, Revision: st.Revision})
	if err != nil {
		t.Fatal(err)
	}
	st, _, err = svc.EditMatch(ctx, "1", res.MatchID)
	if err != nil {
		t.Fatal(err)
	}
	ann := st.Annotated
	if !ann.Document.Header.PostCrawford || ann.Games[0].Crawford || len(ann.Actions[1].Inconsistencies) > 0 {
		t.Fatalf("reopened: post-Crawford %v, games %+v", ann.Document.Header.PostCrawford, ann.Games)
	}
}

package transcription

import (
	"context"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

func sameTick(a, b *int64) bool { return (a == nil) == (b == nil) && (a == nil || *a == *b) }

// TestVideoAndRepèresSurviveAMatchRoundTrip: a Match saved with a video and its
// Repères, reopened as a draft and saved again, keeps both — and the durations
// the Repères give reach its Moves.
func TestVideoAndRepèresSurviveAMatchRoundTrip(t *testing.T) {
	ctx := context.Background()
	svc := New(newStore(t), Options{})
	header := transcript.Header{MatchLength: 5, Player1: "A", Player2: "B", Date: time.Now(), Transcriber: "T",
		VideoSource: "https://example.org/match.mp4"}
	_, actions := typedGame(t, svc, "1", header, 12)
	clock := int64(10_000)
	for i := range actions {
		a := &actions[i]
		if a.Kind == transcript.KindChecker || a.Kind == transcript.KindDance {
			r := clock
			a.RollTickMS = &r
			clock += 1000
		}
		tk := clock
		a.TickMS = &tk
		clock += 2000
	}
	res, err := svc.Materialize(ctx, "1", header, actions)
	if err != nil {
		t.Fatal(err)
	}
	m := matchHeader(t, svc, "1", res.MatchID)
	if m.VideoSource == nil || *m.VideoSource != header.VideoSource {
		t.Fatalf("saved source %v", m.VideoSource)
	}
	deduced := 0
	for mv, err := range svc.store.Matches().MovesByMatch(ctx, "1", res.MatchID) {
		if err != nil {
			t.Fatal(err)
		}
		if mv.TickMS == nil {
			t.Fatalf("move %d lost its Repère", mv.MoveNumber)
		}
		if mv.DecisionMS != nil {
			deduced++
		}
	}
	if deduced == 0 {
		t.Fatal("no duration was deduced from the Repères")
	}

	st, _, err := svc.EditMatch(ctx, "1", res.MatchID)
	if err != nil {
		t.Fatal(err)
	}
	doc := st.Annotated.Document
	if doc.Header.VideoSource != header.VideoSource {
		t.Fatalf("the draft lost the source: %q", doc.Header.VideoSource)
	}
	if len(doc.Actions) != len(actions) {
		t.Fatalf("%d Actions back, want %d", len(doc.Actions), len(actions))
	}
	for i, a := range doc.Actions {
		if !sameTick(a.RollTickMS, actions[i].RollTickMS) || !sameTick(a.TickMS, actions[i].TickMS) {
			t.Fatalf("Action %d came back with other Repères", i)
		}
		if a.DecisionMS != nil || a.CubeDecisionMS != nil {
			t.Fatalf("Action %d came back with a duration written on it", i)
		}
	}

	if _, err := svc.Finish(ctx, "1", st.ID, Expect{Session: st.SessionID}); err != nil {
		t.Fatal(err)
	}
	if m := matchHeader(t, svc, "1", res.MatchID); m.VideoSource == nil || *m.VideoSource != header.VideoSource {
		t.Fatalf("saving the draft again dropped the source: %v", m.VideoSource)
	}

	// Detached in the draft, it is gone from the Match.
	st, _, err = svc.EditMatch(ctx, "1", res.MatchID)
	if err != nil {
		t.Fatal(err)
	}
	if st, err = svc.Apply(ctx, "1", st.ID, Expect{Session: st.SessionID}, transcript.Gesture{Kind: transcript.GestureSetVideo}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Finish(ctx, "1", st.ID, Expect{Session: st.SessionID}); err != nil {
		t.Fatal(err)
	}
	if m := matchHeader(t, svc, "1", res.MatchID); m.VideoSource != nil && *m.VideoSource != "" {
		t.Fatalf("a detached source is still on the Match: %q", *m.VideoSource)
	}
}

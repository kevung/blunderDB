package database

import (
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/duel"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// A match given up in the transcription panel is saved won by the other side,
// for the points that bring it to the length — not the single a resignation of
// the same game is worth, which leaves the match open — and its .mat ends as a
// forfeited Duel's does.
func TestTranscriptionForfeit_SavesTheMatchWonAndExportsIt(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)

	state, err := db.CreateTranscription(transcript.Header{MatchLength: 5})
	if err != nil {
		t.Fatal(err)
	}
	id := state.ID
	for _, g := range []transcript.Gesture{
		{Kind: transcript.GestureEnterDie, Die: 6, HasSide: true, Side: domain.Black},
		{Kind: transcript.GestureEnterDie, Die: 3},
		{Kind: transcript.GestureSelectCandidate, Candidate: 0},
		{Kind: transcript.GestureValidate},
		{Kind: transcript.GestureForfeit, HasTick: true, TickMS: 42000},
	} {
		if state, err = db.ApplyTranscriptionGesture(id, g); err != nil {
			t.Fatalf("%s: %v", g.Kind, err)
		}
	}
	ann := state.Annotated
	last := ann.Document.Actions[len(ann.Document.Actions)-1]
	if last.Kind != transcript.KindForfeit || last.Side != domain.White || last.TickMS == nil || *last.TickMS != 42000 {
		t.Fatalf("last Action = %+v, want White's forfeit at 42000", last)
	}
	if !ann.Finished || ann.Winner != domain.Black || ann.Score != [2]int{5, 0} {
		t.Fatalf("finished %v, winner %d, score %v; want the match to player 1, 5-0", ann.Finished, ann.Winner, ann.Score)
	}
	if _, err := db.ApplyTranscriptionGesture(id, transcript.Gesture{Kind: transcript.GestureEnterDie, Die: 4}); err == nil {
		t.Error("a die was taken after the forfeit")
	}

	saved, err := db.FinishTranscription(id)
	if err != nil {
		t.Fatalf("FinishTranscription: %v", err)
	}
	var winner, points int
	if err := db.db.QueryRow(`SELECT winner, points_won FROM game WHERE match_id = ?`, saved.MatchID).Scan(&winner, &points); err != nil {
		t.Fatal(err)
	}
	if winner != 1 || points != 5 {
		t.Errorf("the saved game: winner %d, %d points; want player 1, 5", winner, points)
	}
	text, err := db.MatchMAT(saved.MatchID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Wins 5 points and the match") {
		t.Errorf("the .mat does not give player 1 the match:\n%s", text)
	}
	if got, want := matResult(text), matResult(forfeitedDuelMAT(t, db)); got != want {
		t.Errorf("the transcription's .mat ends %q, a forfeited Duel's %q", got, want)
	}
}

// forfeitedDuelMAT is the .mat of a 5-point Duel player 2 gives up after the
// opening play.
func forfeitedDuelMAT(t *testing.T, db *Database) string {
	t.Helper()
	sides := [2]duel.SideSpec{{Kind: duel.SideExternal, Name: "A"}, {Kind: duel.SideExternal, Name: "B"}}
	st, err := db.CreateDuel(duel.Settings{MatchLength: 5, Sides: sides})
	if err != nil {
		t.Fatal(err)
	}
	for st.State.Awaiting != nil && st.State.Awaiting.Kind != duel.DecideMove {
		a := st.State.Awaiting
		if st, err = db.PlayDuel(st.State.ID, st.State.Revision, duel.Play{Side: a.Side, Kind: duel.PlayRoll}); err != nil {
			t.Fatal(err)
		}
	}
	a := st.State.Awaiting
	p := duel.Play{Side: a.Side, Kind: duel.PlayMove, Steps: domain.LegalMoves(&a.Position)[0].Steps}
	if st, err = db.PlayDuel(st.State.ID, st.State.Revision, p); err != nil {
		t.Fatal(err)
	}
	ended, err := db.ForfeitDuel(st.State.ID, st.State.Revision, 1-a.Side)
	if err != nil || ended.State.Ended == nil {
		t.Fatalf("ForfeitDuel = %+v, %v", ended, err)
	}
	text, err := db.MatchMAT(ended.State.Ended.MatchID)
	if err != nil {
		t.Fatal(err)
	}
	return text
}

// matResult is the .mat's result line, the winner's column dropped.
func matResult(text string) string {
	for _, line := range strings.Split(text, "\n") {
		if i := strings.Index(line, "Wins "); i >= 0 {
			return strings.TrimSpace(line[i:])
		}
	}
	return ""
}

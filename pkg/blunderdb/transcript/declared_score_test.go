package transcript

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/kevung/gnubgparser"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

// ADR-0053: a game's score can be DECLARED on its first Action, because a score error
// made at the table is a fact of the match and the transcription owes it to the record.

// wonByPass is a game player 1 wins one point in: player 1 opens with 31, player 2
// plays 31 too, player 1 doubles, player 2 passes. Its four Actions start at 4×game.
func wonByPass(score *[2]int) []Action {
	return []Action{
		{Side: domain.Black, Kind: KindChecker, Dice: [2]int{3, 1}, Score: score,
			Steps: []domain.CheckerStep{{From: 8, To: 5}, {From: 6, To: 5}}},
		{Side: domain.White, Kind: KindChecker, Dice: [2]int{3, 1},
			Steps: []domain.CheckerStep{{From: 17, To: 20}, {From: 19, To: 20}}},
		{Side: domain.Black, Kind: KindDouble},
		{Side: domain.White, Kind: KindPass},
	}
}

func scoreOf(a, b int) *[2]int { return &[2]int{a, b} }

func concat(games ...[]Action) []Action {
	var out []Action
	for _, g := range games {
		out = append(out, g...)
	}
	return out
}

func TestDeclaredScoreEqualToTheDerivedOneIsSilent(t *testing.T) {
	doc := docOf(3, concat(wonByPass(nil), wonByPass(scoreOf(1, 0)))...)
	ann := Replay(doc, 0)
	if ann.Inconsistent() {
		t.Fatalf("a declared score equal to the derived one is marked: %+v", ann.Actions[4].Inconsistencies)
	}
	g := ann.Games[1]
	if !g.Declared || g.InitialScore != [2]int{1, 0} || g.DerivedScore != [2]int{1, 0} {
		t.Errorf("game 2 = %+v, want declared 1-0 derived 1-0", g)
	}
	if ann.Games[0].Declared || ann.Games[0].DerivedScore != ann.Games[0].InitialScore {
		t.Errorf("game 1 declares nothing, got %+v", ann.Games[0])
	}
}

// TestDeclaredScoreDiffersIsPlayedAndMarked: a 5-point match whose second game was
// played at 3-0 when the first gave 1-0. Its first play is marked, the game is played at
// 3-0, and everything after follows the declared score — the Crawford game, the end
// of the match, its winner.
func TestDeclaredScoreDiffersIsPlayedAndMarked(t *testing.T) {
	doc := docOf(5, concat(wonByPass(nil), wonByPass(scoreOf(3, 0)), wonByPass(nil), wonByPass(nil))...)
	ann := Replay(doc, 0)

	flags := ann.Actions[4].Inconsistencies
	if len(flags) != 1 || flags[0].Kind != ScoreMismatch || !strings.Contains(flags[0].Detail, "1-0") {
		t.Fatalf("game 2's first play carries %+v, want one score_mismatch naming the derived 1-0", flags)
	}
	g := ann.Games[1]
	if g.InitialScore != [2]int{3, 0} || g.DerivedScore != [2]int{1, 0} || !g.Declared {
		t.Errorf("game 2 = %+v, want played at 3-0, derived 1-0", g)
	}
	// The Positions of the game carry the away score of 3-0 in a 5-point match.
	if got := ann.Actions[6].Before.Score; got != [2]int{2, 5} {
		t.Errorf("the double of game 2 is played at away %v, want [2 5]", got)
	}
	// 4-0 after game 2: game 3 is the Crawford game, which 2-0 would not have made.
	if !ann.Games[2].Crawford || ann.Games[2].InitialScore != [2]int{4, 0} {
		t.Errorf("game 3 = %+v, want the Crawford game at 4-0", ann.Games[2])
	}
	if got := ann.Actions[10].Before.Score; got != [2]int{domain.Crawford, 5} {
		t.Errorf("the Crawford game's double is played at %v", got)
	}
	// Game 3 won: 5-0, the match is over, and game 4 is past its end.
	if !ann.Finished || ann.Winner != domain.Black || ann.Score != [2]int{6, 0} {
		t.Errorf("match finished=%v winner=%d score=%v", ann.Finished, ann.Winner, ann.Score)
	}
	if !hasInconsistency(ann.Actions[12], PastEnd) {
		t.Errorf("game 4 is played after the match was won and is not marked")
	}

	// What a save writes and what a .mat says: the score the game was played at.
	_, games, _ := MatchParts(doc)
	if games[1].InitialScore != [2]int32{3, 0} || games[2].InitialScore != [2]int32{4, 0} {
		t.Errorf("saved games start at %v and %v", games[1].InitialScore, games[2].InitialScore)
	}
}

func TestDeclaredScoreItCannotUseIsMarkedAndIgnored(t *testing.T) {
	// In a money session, which has no score.
	money := docOf(0, wonByPass(scoreOf(1, 0))...)
	if ann := Replay(money, 0); !hasInconsistency(ann.Actions[0], ScoreMismatch) || ann.Games[0].Declared {
		t.Errorf("a score in a money session is not marked: %+v", ann.Actions[0].Inconsistencies)
	}
	// Below zero.
	if ann := Replay(docOf(5, wonByPass(scoreOf(-1, 0))...), 0); !hasInconsistency(ann.Actions[0], ScoreMismatch) ||
		ann.Games[0].InitialScore != [2]int{} {
		t.Errorf("a negative score is not marked and ignored: %+v", ann.Games[0])
	}
}

func TestSetScoreGesture(t *testing.T) {
	doc := docOf(5, concat(wonByPass(nil), wonByPass(nil), wonByPass(nil)[:2])...)
	ed := NewEditor(doc)
	cursor := ed.Doc.Cursor

	// Refused: only what has no meaning.
	for name, g := range map[string]Gesture{
		"not a first play": {Kind: GestureSetScore, At: 1, Score: scoreOf(1, 0)},
		"out of range":     {Kind: GestureSetScore, At: 99, Score: scoreOf(1, 0)},
		"mid-game":         {Kind: GestureSetScore, At: 9, Score: scoreOf(1, 0)},
		"a negative":       {Kind: GestureSetScore, At: 4, Score: scoreOf(0, -1)},
		"a negative too":   {Kind: GestureSetScore, At: 4, Score: scoreOf(-2, 0)},
		"a negative index": {Kind: GestureSetScore, At: -1, Score: scoreOf(1, 0)},
	} {
		if err := ed.Apply(g); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if ed.CanUndo() {
		t.Fatal("a refused gesture reached the stack")
	}

	if err := ed.Apply(Gesture{Kind: GestureSetScore, At: 4, Score: scoreOf(3, 1)}); err != nil {
		t.Fatal(err)
	}
	ann := ed.Replay(ed.From())
	if ann.Cursor != cursor {
		t.Errorf("declaring a score moved the Cursor from %d to %d", cursor, ann.Cursor)
	}
	if ann.Games[1].InitialScore != [2]int{3, 1} || !hasInconsistency(ann.Actions[4], ScoreMismatch) {
		t.Errorf("game 2 = %+v, flags %+v", ann.Games[1], ann.Actions[4].Inconsistencies)
	}
	if ann.Games[2].InitialScore != [2]int{4, 1} {
		t.Errorf("game 3 does not follow the declared score: %+v", ann.Games[2])
	}

	if err := ed.Apply(Gesture{Kind: GestureSetScore, At: 8, Score: scoreOf(4, 2)}); err != nil {
		t.Fatal(err)
	}
	if got := ed.Replay(0).Games[2]; got.InitialScore != [2]int{4, 2} || got.DerivedScore != [2]int{4, 1} {
		t.Errorf("game 3 declared 4-2: %+v", got)
	}

	// A score at or past the length is declared, not refused.
	if err := ed.Apply(Gesture{Kind: GestureSetScore, At: 8, Score: scoreOf(5, 0)}); err != nil {
		t.Fatalf("a score reaching the length is refused: %v", err)
	}
	if ann := ed.Replay(0); !ann.Finished || !hasInconsistency(ann.Actions[9], PastEnd) {
		t.Errorf("a declared 5-0 in a 5-point match does not end it: finished=%v", ann.Finished)
	}

	// Cleared: back to the derived score.
	if err := ed.Apply(Gesture{Kind: GestureSetScore, At: 4}); err != nil {
		t.Fatal(err)
	}
	if ed.Doc.Actions[4].Score != nil {
		t.Fatal("clearing left the score")
	}
	if got := ed.Replay(0).Games[1]; got.InitialScore != [2]int{1, 0} || got.Declared {
		t.Errorf("game 2 after clearing = %+v, want the derived 1-0", got)
	}

	// Undo and redo walk it like any gesture.
	ed.Undo()
	if got := ed.Replay(0).Games[1].InitialScore; got != [2]int{3, 1} {
		t.Errorf("undoing the clear gives %v, want 3-1", got)
	}
	ed.Redo()
	if ed.Doc.Actions[4].Score != nil {
		t.Error("redoing the clear left the score")
	}
	for ed.Undo() {
	}
	if !reflect.DeepEqual(Replay(ed.Doc, 0), Replay(doc, 0)) {
		t.Error("undoing everything does not give the document back")
	}

	// A money session refuses a score and still lets one be cleared.
	money := docOf(0, wonByPass(scoreOf(1, 0))...)
	if _, err := Apply(money, Gesture{Kind: GestureSetScore, At: 0, Score: scoreOf(1, 0)}); err == nil {
		t.Error("a money session accepted a score")
	}
	if out, err := Apply(money, Gesture{Kind: GestureSetScore, At: 0}); err != nil || out.Actions[0].Score != nil {
		t.Errorf("a money session cannot clear a stray score: %v", err)
	}
}

// TestDeclaredScoreSurvivesTheOtherGestures: it belongs to the game, so the gestures
// that rewrite a game's first play carry it.
func TestDeclaredScoreSurvivesTheOtherGestures(t *testing.T) {
	doc := docOf(5, concat(wonByPass(nil), wonByPass(scoreOf(3, 1)))...)

	swapped, err := Apply(doc, Gesture{Kind: GestureSwapPlayers})
	if err != nil {
		t.Fatal(err)
	}
	if got := swapped.Actions[4].Score; got == nil || *got != [2]int{1, 3} {
		t.Errorf("swapping the players gives the score %v, want 1-3", got)
	}

	// Correcting the first play's roll keeps the score declared on it.
	corrected := seek(t, doc, 4)
	corrected = runSteps(t, corrected, []step{{"die 4", die(4), nil}, {"die 2", die(2), nil}, {"validate", confirm(), nil}})
	if a := corrected.Actions[4]; a.Dice != [2]int{4, 2} || a.Score == nil || *a.Score != [2]int{3, 1} {
		t.Errorf("the corrected first play is %+v", a)
	}

	// Deleting the first play hands the score to the Action after it, which now
	// opens the game.
	cut := doc.clone()
	cut.Cursor = 4
	out, err := Apply(cut, Gesture{Kind: GestureDelete})
	if err != nil {
		t.Fatal(err)
	}
	if got := out.Actions[4].Score; got == nil || *got != [2]int{3, 1} {
		t.Errorf("deleting the first play lost the score: %v", got)
	}

	// A play inserted in front of the first one opens the game in its place and
	// takes the score over.
	ins := doc.clone()
	ins.Cursor = 4
	ins = runSteps(t, ins, []step{
		{"insert", Gesture{Kind: GestureInsertBefore}, nil},
		{"die 6", die(6), nil}, {"die 5", die(5), nil},
		{"a play", candidate(0), nil}, {"validate", confirm(), nil},
	})
	if a, b := ins.Actions[4], ins.Actions[5]; a.Score == nil || *a.Score != [2]int{3, 1} || b.Score != nil {
		t.Errorf("after an insertion in front of the first play: %+v / %+v", a.Score, b.Score)
	}
}

// TestDeclaredScoreJSON: the field is optional, and a document that carries it
// round-trips.
func TestDeclaredScoreJSON(t *testing.T) {
	doc := docOf(5, wonByPass(nil)...)
	blob, _ := json.Marshal(doc)
	if strings.Contains(string(blob), `"score"`) {
		t.Fatalf("an undeclared score is written: %s", blob)
	}
	doc.Actions[0].Score = scoreOf(2, 1)
	blob, _ = json.Marshal(doc)
	if !strings.Contains(string(blob), `"score":[2,1]`) {
		t.Errorf("the score is not written: %s", blob)
	}
	var back Document
	if err := json.Unmarshal(blob, &back); err != nil || back.Actions[0].Score == nil || *back.Actions[0].Score != [2]int{2, 1} {
		t.Errorf("the score does not read back: %+v %v", back.Actions[0], err)
	}
}

// TestDeclaredScoreMATRoundTrip: the .mat says the score the game was played at, and
// reading that file back declares it again — the same games, the same mark.
func TestDeclaredScoreMATRoundTrip(t *testing.T) {
	// Player 1 plays the opening 31, player 2 doubles, player 1 passes — a point
	// for player 2.
	played := func(score *[2]int) []Action {
		return []Action{
			{Side: domain.Black, Kind: KindChecker, Dice: [2]int{3, 1}, Score: score,
				Steps: []domain.CheckerStep{{From: 8, To: 5}, {From: 6, To: 5}}},
			{Side: domain.White, Kind: KindDouble},
			{Side: domain.Black, Kind: KindPass},
		}
	}
	doc := docOf(5, concat(played(nil), played(scoreOf(3, 0)), played(nil))...)
	ann := Replay(doc, 0)
	if ann.Games[1].DerivedScore != [2]int{0, 1} || ann.Games[1].InitialScore != [2]int{3, 0} {
		t.Fatalf("fixture: game 2 = %+v", ann.Games[1])
	}

	text := ingest.RenderMAT(MatchParts(doc))
	parsed, err := gnubgparser.ParseMAT(strings.NewReader(text))
	if err != nil {
		t.Fatal(err)
	}
	if got := parsed.Games[1].Score; got != [2]int{3, 0} {
		t.Errorf("the .mat writes game 2 at %v, want the declared 3-0:\n%s", got, text)
	}
	if got := parsed.Games[2].Score; got != [2]int{3, 1} {
		t.Errorf("the .mat writes game 3 at %v, want 3-1", got)
	}

	back, err := FromMAT(text)
	if err != nil {
		t.Fatal(err)
	}
	if back.FormatVersion != FormatVersion {
		t.Errorf("read back at version %d", back.FormatVersion)
	}
	rt := Replay(back, 0)
	if !reflect.DeepEqual(rt.Games, ann.Games) {
		t.Errorf("games read back differ:\n%+v\n%+v", rt.Games, ann.Games)
	}
	marked := 0
	for _, info := range rt.Actions {
		if hasInconsistency(info, ScoreMismatch) {
			marked++
		}
	}
	if marked != 1 {
		t.Errorf("%d score mismatches read back, want 1", marked)
	}
}

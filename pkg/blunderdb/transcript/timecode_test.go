package transcript

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
)

// at stamps a gesture with the video's instant.
func at(g Gesture, ms int64) Gesture {
	g.TickMS, g.HasTick = ms, true
	return g
}

func ms(v int64) *int64 { return &v }

// sameMS compares two optional durations or Repères.
func sameMS(a, b *int64) bool {
	return (a == nil) == (b == nil) && (a == nil || *a == *b)
}

func show(p *int64) string {
	if p == nil {
		return "unknown"
	}
	return strconv.FormatInt(*p, 10)
}

// timedTurn types a roll and its first candidate, the first die at roll and the
// validation at done.
func timedTurn(d1, d2 int, roll, done int64) []step {
	return []step{
		{name: "first die", g: at(die(d1), roll)},
		{name: "second die", g: at(die(d2), roll+400)},
		{name: "play", g: candidate(0)},
		{name: "validate", g: at(confirm(), done)},
	}
}

// timedGame is a 7-point game: player 1 opens 6-3, player 2 plays 5-2 and could
// have doubled, player 1 doubles, player 2 takes, player 1 — the cube now
// player 2's — plays 4-1.
func timedGame(t *testing.T) Document {
	t.Helper()
	var steps []step
	steps = append(steps, timedTurn(6, 3, 1000, 4000)...)
	steps = append(steps, timedTurn(5, 2, 6000, 9000)...)
	steps = append(steps,
		step{name: "double", g: at(Gesture{Kind: GestureDouble}, 12000)},
		step{name: "take", g: at(Gesture{Kind: GestureTake}, 15000)})
	steps = append(steps, timedTurn(4, 1, 17000, 20000)...)
	return runSteps(t, New(7), steps)
}

type durations struct{ cube, decision *int64 }

func checkDurations(t *testing.T, ann Annotated, want []durations) {
	t.Helper()
	if len(ann.Actions) != len(want) {
		t.Fatalf("%d Actions, want %d", len(ann.Actions), len(want))
	}
	for i, w := range want {
		got := ann.Actions[i]
		if !sameMS(got.CubeDecisionMS, w.cube) || !sameMS(got.DecisionMS, w.decision) {
			t.Errorf("Action %d (%s): cube %s, decision %s; want %s, %s", i, got.Kind,
				show(got.CubeDecisionMS), show(got.DecisionMS), show(w.cube), show(w.decision))
		}
	}
}

// TestDurationsAreDeducedFromTheRepères: the Repères the gestures post give every
// duration of ADR-0082 rule 2, and none of them is written on an Action.
func TestDurationsAreDeducedFromTheRepères(t *testing.T) {
	doc := timedGame(t)
	wantTicks := [][2]*int64{{ms(1000), ms(4000)}, {ms(6000), ms(9000)}, {nil, ms(12000)}, {nil, ms(15000)}, {ms(17000), ms(20000)}}
	for i, w := range wantTicks {
		a := doc.Actions[i]
		if !sameMS(a.RollTickMS, w[0]) || !sameMS(a.TickMS, w[1]) {
			t.Errorf("Action %d Repères %s/%s, want %s/%s", i, show(a.RollTickMS), show(a.TickMS), show(w[0]), show(w[1]))
		}
		if a.DecisionMS != nil || a.CubeDecisionMS != nil {
			t.Errorf("Action %d carries a deduced duration", i)
		}
	}
	checkDurations(t, Replay(doc, 0), []durations{
		{nil, ms(3000)},      // a game's first play has no cube decision
		{ms(2000), ms(3000)}, // player 2 could double: from player 1's play to the roll
		{nil, ms(3000)},      // the double, from the play before it
		{nil, ms(3000)},      // the take
		{nil, ms(3000)},      // player 1 no longer holds the cube
	})

	// A dance has a cube decision and no checker decision; a missing Repère
	// leaves unknown what it would bound; a measured duration stands.
	doc.Actions = append(doc.Actions,
		Action{Side: 1, Kind: KindDance, Dice: [2]int{6, 6}, RollTickMS: ms(22000), TickMS: ms(23000)},
		Action{Side: 0, Kind: KindUnrecorded, Dice: [2]int{3, 1}, RollTickMS: ms(25000)},
		Action{Side: 1, Kind: KindUnrecorded, Dice: [2]int{2, 1}, RollTickMS: ms(30000), TickMS: ms(31000)},
		Action{Side: 0, Kind: KindUnrecorded, Dice: [2]int{2, 1}, RollTickMS: ms(32000), CubeDecisionMS: ms(7)},
	)
	ann := Replay(doc, 0)
	checkDurations(t, ann, []durations{
		{nil, ms(3000)}, {ms(2000), ms(3000)}, {nil, ms(3000)}, {nil, ms(3000)}, {nil, ms(3000)},
		{ms(2000), nil}, // the dance
		{nil, nil},      // player 1 cannot double, and the play has no Repère
		{nil, nil},      // the cube decision starts on a Repère nobody posted
		{ms(7), nil},    // the Arbiter's measure, not 32000-31000
	})
}

// TestFirstPlayOfAGameHasNoCubeDecision: after a pass the next game opens on a
// play nobody could double before.
func TestFirstPlayOfAGameHasNoCubeDecision(t *testing.T) {
	var steps []step
	steps = append(steps, timedTurn(6, 3, 1000, 4000)...)
	steps = append(steps, timedTurn(5, 2, 6000, 9000)...)
	steps = append(steps,
		step{name: "double", g: at(Gesture{Kind: GestureDouble}, 12000)},
		step{name: "pass", g: at(Gesture{Kind: GesturePass}, 13000)})
	steps = append(steps, timedTurn(2, 5, 40000, 41000)...)
	ann := Replay(runSteps(t, New(7), steps), 0)
	checkDurations(t, ann, []durations{
		{nil, ms(3000)}, {ms(2000), ms(3000)}, {nil, ms(3000)}, {nil, ms(1000)},
		{nil, ms(1000)},
	})
	if !ann.Actions[4].OpensGame {
		t.Fatal("the last play should open the second game")
	}
}

// TestTimecodeBackwardsIsMarkedNotCorrected: a Repère earlier than the one before
// it is kept, marked, and every duration that uses it is unknown.
func TestTimecodeBackwardsIsMarkedNotCorrected(t *testing.T) {
	doc := seek(t, timedGame(t), 1)
	doc, err := Apply(doc, Gesture{Kind: GestureSetTimecode, RollTickMS: 3000, HasRollTick: true})
	if err != nil {
		t.Fatal(err)
	}
	if !sameMS(doc.Actions[1].RollTickMS, ms(3000)) {
		t.Fatalf("roll Repère %s, want 3000 as posted", show(doc.Actions[1].RollTickMS))
	}
	ann := Replay(doc, 0)
	if !hasInconsistency(ann.Actions[1], TimecodeBackwards) {
		t.Fatalf("Action 1 is not marked: %+v", ann.Actions[1].Inconsistencies)
	}
	for i, info := range ann.Actions {
		if i != 1 && hasInconsistency(info, TimecodeBackwards) {
			t.Errorf("Action %d is marked too", i)
		}
	}
	checkDurations(t, ann, []durations{
		{nil, ms(3000)}, {nil, nil}, {nil, ms(3000)}, {nil, ms(3000)}, {nil, ms(3000)},
	})

	// The roll then the action of one Action are in order too.
	doc, err = Apply(doc, Gesture{Kind: GestureSetTimecode, RollTickMS: 5000, HasRollTick: true, TickMS: 4500, HasTick: true})
	if err != nil {
		t.Fatal(err)
	}
	ann = Replay(doc, 0)
	if !hasInconsistency(ann.Actions[1], TimecodeBackwards) || ann.Actions[1].DecisionMS != nil {
		t.Fatalf("an action timed before its roll: %+v, decision %s", ann.Actions[1].Inconsistencies, show(ann.Actions[1].DecisionMS))
	}

	// A negative instant clears; a cube action has no roll to time.
	doc, err = Apply(doc, Gesture{Kind: GestureSetTimecode, RollTickMS: -1, HasRollTick: true, TickMS: -1, HasTick: true})
	if err != nil {
		t.Fatal(err)
	}
	if doc.Actions[1].RollTickMS != nil || doc.Actions[1].TickMS != nil {
		t.Fatal("a negative instant should clear the Repère")
	}
	if hasInconsistency(Replay(doc, 0).Actions[1], TimecodeBackwards) {
		t.Fatal("a cleared Repère is still marked")
	}
	cube := seek(t, timedGame(t), 2)
	if _, err := Apply(cube, Gesture{Kind: GestureSetTimecode, RollTickMS: 1, HasRollTick: true}); err == nil {
		t.Fatal("a double was given a roll Repère")
	}
}

// TestEditingMovesNoRepère: deleting an Action, inserting one, correcting one in
// place leave every other Repère where it was, and the correction keeps its own.
func TestEditingMovesNoRepère(t *testing.T) {
	orig := timedGame(t)
	ticks := func(as []Action) []*int64 {
		var out []*int64
		for _, a := range as {
			out = append(out, a.RollTickMS, a.TickMS)
		}
		return out
	}
	same := func(name string, got, want []*int64) {
		t.Helper()
		if len(got) != len(want) {
			t.Fatalf("%s: %d Repères, want %d", name, len(got), len(want))
		}
		for i := range got {
			if !sameMS(got[i], want[i]) {
				t.Errorf("%s: Repère %d is %s, want %s", name, i, show(got[i]), show(want[i]))
			}
		}
	}

	// Corrected in place: another roll, another play, the same Repères.
	fixed := runSteps(t, seek(t, orig, 1), []step{
		{name: "die", g: at(die(4), 99000)}, {name: "die", g: at(die(4), 99100)},
		{name: "play", g: candidate(0)}, {name: "validate", g: at(confirm(), 99900)},
	})
	if fixed.Actions[1].Dice != [2]int{4, 4} {
		t.Fatalf("the roll was not corrected: %v", fixed.Actions[1].Dice)
	}
	same("correction", ticks(fixed.Actions), ticks(orig.Actions))

	// Deleted: the Repères of the others do not shift.
	gone, err := Apply(seek(t, orig, 2), Gesture{Kind: GestureDelete})
	if err != nil {
		t.Fatal(err)
	}
	want := append(append([]*int64{}, ticks(orig.Actions)[:4]...), ticks(orig.Actions)[6:]...)
	same("deletion", ticks(gone.Actions), want)

	// Inserted: the new Action gets the instants it was typed at, the others keep theirs.
	ins, err := Apply(seek(t, orig, 1), Gesture{Kind: GestureInsertBefore})
	if err != nil {
		t.Fatal(err)
	}
	ins = runSteps(t, ins, timedTurn(5, 2, 50000, 51000))
	want = append(append(append([]*int64{}, ticks(orig.Actions)[:2]...), ms(50000), ms(51000)), ticks(orig.Actions)[2:]...)
	same("insertion", ticks(ins.Actions), want)
}

// TestUntimedDocumentIsUnchanged: a draft typed without a video carries no Repère,
// no duration and no source, in its JSON as in its Replay.
func TestUntimedDocumentIsUnchanged(t *testing.T) {
	doc := typedMatch(t, 7)
	blob, err := json.Marshal(Replay(doc, 0))
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"tick_ms", "decision_ms", "video_source", string(TimecodeBackwards)} {
		if strings.Contains(string(blob), k) {
			t.Errorf("an untimed draft renders %q", k)
		}
	}
}

// TestVideoSourceGestures: set_video attaches and detaches; a header form that
// does not name the video keeps it.
func TestVideoSourceGestures(t *testing.T) {
	doc := runSteps(t, New(7), []step{
		{name: "attach", g: Gesture{Kind: GestureSetVideo, VideoSource: " https://example.org/m.mp4 "}},
		{name: "header", g: Gesture{Kind: GestureSetHeader, Header: Header{Player1: "A"}}},
	})
	if doc.Header.VideoSource != "https://example.org/m.mp4" || doc.Header.Player1 != "A" {
		t.Fatalf("header %+v", doc.Header)
	}
	doc = runSteps(t, doc, []step{{name: "header names one", g: Gesture{Kind: GestureSetHeader, Header: Header{VideoSource: "/v/m.mkv"}}}})
	if doc.Header.VideoSource != "/v/m.mkv" {
		t.Fatalf("set_header did not post the source: %q", doc.Header.VideoSource)
	}
	doc = runSteps(t, doc, []step{{name: "detach", g: Gesture{Kind: GestureSetVideo}}})
	if doc.Header.VideoSource != "" {
		t.Fatalf("set_video \"\" left %q", doc.Header.VideoSource)
	}
}

// TestBuildCarriesRepèresAndDurations: the saved Moves carry the Repères and the
// durations, measured or deduced; the Match carries the source.
func TestBuildCarriesRepèresAndDurations(t *testing.T) {
	doc := timedGame(t)
	doc.Header.VideoSource = "https://example.org/m.mp4"
	p := Build(doc)
	if p.Match.VideoSource == nil || *p.Match.VideoSource != doc.Header.VideoSource {
		t.Fatalf("match source %v", p.Match.VideoSource)
	}
	moves := p.Moves[p.Games[0].ID]
	if len(moves) != 5 {
		t.Fatalf("%d moves, want 5", len(moves))
	}
	if mv := moves[1]; !sameMS(mv.RollTickMS, ms(6000)) || !sameMS(mv.TickMS, ms(9000)) ||
		!sameMS(mv.CubeDecisionMS, ms(2000)) || !sameMS(mv.DecisionMS, ms(3000)) {
		t.Errorf("checker move %s/%s %s/%s", show(mv.RollTickMS), show(mv.TickMS), show(mv.CubeDecisionMS), show(mv.DecisionMS))
	}
	if mv := moves[2]; mv.RollTickMS != nil || !sameMS(mv.TickMS, ms(12000)) || !sameMS(mv.DecisionMS, ms(3000)) {
		t.Errorf("double %s/%s %s", show(mv.RollTickMS), show(mv.TickMS), show(mv.DecisionMS))
	}
	// Detached, the source is stated empty so a replaced Match loses it.
	doc.Header.VideoSource = ""
	if p := Build(doc); p.Match.VideoSource == nil || *p.Match.VideoSource != "" {
		t.Fatalf("a detached source should be stated empty, got %v", p.Match.VideoSource)
	}
}

// TestImplicitValidationPostsNoInstant: a play left selected and recorded by the
// gesture that follows it — a cube gesture — gets no action Repère: the instant
// is the cube's, and lending it to the play would be false (ADR-0082 rule 3).
func TestImplicitValidationPostsNoInstant(t *testing.T) {
	var steps []step
	steps = append(steps, timedTurn(6, 3, 1000, 4000)...)
	steps = append(steps,
		step{name: "first die", g: at(die(5), 6000)},
		step{name: "second die", g: at(die(2), 6400)},
		step{name: "play", g: candidate(0)},
		step{name: "double", g: at(Gesture{Kind: GestureDouble}, 9000)})
	doc := runSteps(t, New(7), steps)
	if len(doc.Actions) != 3 || doc.Actions[1].Kind != KindChecker || doc.Actions[2].Kind != KindDouble {
		t.Fatalf("want play then double, got %s", dumpActions(doc.Actions))
	}
	if p := doc.Actions[1]; !sameMS(p.RollTickMS, ms(6000)) || p.TickMS != nil {
		t.Errorf("implicitly validated play: roll %s, action %s; want 6000, unknown", show(p.RollTickMS), show(p.TickMS))
	}
	if d := doc.Actions[2]; !sameMS(d.TickMS, ms(9000)) {
		t.Errorf("double timed %s, want 9000", show(d.TickMS))
	}

	// A die typed over a selected play writes nothing: it retypes the roll,
	// so no instant lands on any Action either.
	steps = append(timedTurn(6, 3, 1000, 4000),
		step{name: "first die", g: at(die(5), 6000)},
		step{name: "second die", g: at(die(2), 6400)},
		step{name: "play", g: candidate(0)},
		step{name: "next die", g: at(die(4), 9000)})
	doc = runSteps(t, New(7), steps)
	if len(doc.Actions) != 1 {
		t.Fatalf("a die recorded the selected play: %s", dumpActions(doc.Actions))
	}
	if e := doc.Entry; e == nil || !sameMS(e.RollTickMS, ms(6000)) || e.TickMS != nil {
		t.Fatalf("the entry should keep its first die's instant and no action instant: %+v", doc.Entry)
	}
}

// TestCorrectionToACubeActionDropsTheRollRepère: an Action corrected in place
// into a kind without a roll keeps its action Repère and loses the roll's,
// which no double, answer or resignation carries.
func TestCorrectionToACubeActionDropsTheRollRepère(t *testing.T) {
	doc := runSteps(t, seek(t, timedGame(t), 4), []step{
		{name: "clear the roll", g: Gesture{Kind: GestureClearDice}},
	})
	doc, err := Apply(doc, at(Gesture{Kind: GestureResign, Level: 1}, 99000))
	if err != nil {
		t.Fatal(err)
	}
	a := doc.Actions[4]
	if a.Kind != KindResign {
		t.Fatalf("Action 4 is a %s, want a resignation", a.Kind)
	}
	if a.RollTickMS != nil || !sameMS(a.TickMS, ms(20000)) {
		t.Errorf("corrected into a resignation: roll %s, action %s; want unknown, 20000", show(a.RollTickMS), show(a.TickMS))
	}
}

// checkEstimated compares each Action's play duration and whether it is estimated.
func checkEstimated(t *testing.T, ann Annotated, decision []*int64, estimated []bool) {
	t.Helper()
	for i := range decision {
		got := ann.Actions[i]
		if !sameMS(got.DecisionMS, decision[i]) || got.DecisionEstimated != estimated[i] {
			t.Errorf("Action %d (%s): decision %s estimated %v; want %s estimated %v", i, got.Kind,
				show(got.DecisionMS), got.DecisionEstimated, show(decision[i]), estimated[i])
		}
	}
}

// TestUntimedPlayIsEstimatedToTheNextInstant: a play with a roll and no action
// Repère is estimated, as an upper bound, to the next Action's roll or to a cube
// action's own instant; a measure stands, the last play has nothing to end it,
// a cube decision is never estimated, and a Repère that runs backwards serves
// no estimate.
func TestUntimedPlayIsEstimatedToTheNextInstant(t *testing.T) {
	untime := func(doc Document, idx ...int) Document {
		doc.Actions = append([]Action(nil), doc.Actions...)
		for _, i := range idx {
			doc.Actions[i].TickMS = nil
		}
		return doc
	}

	t.Run("to the next roll", func(t *testing.T) {
		ann := Replay(untime(timedGame(t), 0), 0)
		checkEstimated(t, ann, []*int64{ms(5000), ms(3000)}, []bool{true, false})
		// The cube decision after it would start on the missing instant: unknown.
		if ann.Actions[1].CubeDecisionMS != nil {
			t.Errorf("cube decision %s estimated", show(ann.Actions[1].CubeDecisionMS))
		}
	})

	t.Run("to a cube action", func(t *testing.T) {
		ann := Replay(untime(timedGame(t), 1), 0)
		checkEstimated(t, ann, []*int64{ms(3000), ms(6000)}, []bool{false, true})
		// The double ran from the missing end of the play: no estimate.
		if ann.Actions[2].DecisionMS != nil {
			t.Errorf("double %s estimated", show(ann.Actions[2].DecisionMS))
		}
	})

	t.Run("not the last play", func(t *testing.T) {
		ann := Replay(untime(timedGame(t), 4), 0)
		if d := ann.Actions[4].DecisionMS; d != nil || ann.Actions[4].DecisionEstimated {
			t.Errorf("last play %s", show(d))
		}
	})

	t.Run("a measure stands", func(t *testing.T) {
		doc := untime(timedGame(t), 0)
		doc.Actions[0].DecisionMS = ms(42)
		ann := Replay(doc, 0)
		checkEstimated(t, ann, []*int64{ms(42)}, []bool{false})
	})

	t.Run("not through a backwards Repère", func(t *testing.T) {
		doc := untime(timedGame(t), 0)
		doc.Actions[1].RollTickMS = ms(500)
		ann := Replay(doc, 0)
		checkEstimated(t, ann, []*int64{nil}, []bool{false})
	})
}

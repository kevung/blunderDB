package transcript

import (
	"math/rand"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// playLegalMatch types a whole match of legal Actions through the panel's gestures:
// opening rolls in player order, plays picked or entered by their steps, dances,
// doubles answered both ways in and out of the Crawford game. Nothing it records
// breaks a rule, so a Replay must find no Inconsistency in it.
func playLegalMatch(t *testing.T, seed int64, length int) *Editor {
	t.Helper()
	rng := rand.New(rand.NewSource(seed))
	ed := NewEditor(New(length))
	apply := func(g Gesture) {
		t.Helper()
		if err := ed.Apply(g); err != nil {
			t.Fatalf("seed %d, %s: %v", seed, g.Kind, err)
		}
	}
	roll := func() (int, int) { return rng.Intn(6) + 1, rng.Intn(6) + 1 }
	for turns := 0; turns < 4000; turns++ {
		ann := ed.Replay(len(ed.Doc.Actions))
		if ann.Finished {
			return ed
		}
		next := ann.Next
		if next.GameStart {
			d1, d2 := roll()
			for d1 == d2 {
				d1, d2 = roll()
			}
			apply(die(d1))
			apply(die(d2))
		} else {
			cube := next.Position.Cube
			mayDouble := !next.Crawford && (cube.Owner == domain.None || cube.Owner == next.Side) && cube.Value < 6
			if next.Expects == KindChecker && mayDouble && rng.Intn(8) == 0 {
				apply(Gesture{Kind: GestureDouble})
				if rng.Intn(3) == 0 {
					apply(Gesture{Kind: GesturePass})
					continue
				}
				apply(Gesture{Kind: GestureTake})
			}
			d1, d2 := roll()
			apply(die(d1))
			apply(die(d2))
		}
		cands := Candidates(ed.Doc)
		if len(cands) == 0 {
			apply(Gesture{Kind: GestureDance})
			continue
		}
		pick := rng.Intn(len(cands))
		if rng.Intn(2) == 0 {
			apply(candidate(pick))
		} else {
			steps := make([]domain.CheckerStep, len(cands[pick].Steps))
			for i, s := range cands[pick].Steps {
				steps[i] = domain.CheckerStep{From: s.From, To: s.To}
			}
			apply(Gesture{Kind: GestureEnterPlay, Steps: steps})
		}
		apply(confirm())
	}
	t.Fatalf("seed %d: the match did not end", seed)
	return ed
}

func requireConsistent(t *testing.T, seed int64, what string, ann Annotated) {
	t.Helper()
	for i, info := range ann.Actions {
		for _, f := range info.Inconsistencies {
			a := ann.Document.Actions[i]
			t.Fatalf("seed %d, %s: Action %d (%s side %d dice %v) marked %s: %s",
				seed, what, i, a.Kind, a.Side, a.Dice, f.Kind, f.Detail)
		}
	}
}

// TestLegalMatchHasNoInconsistency: a rule that marks a legal match is a false
// alarm. Each match replays the whole document at every die (quadratic), so a few
// seeds run here and FuzzTranscriptGestures explores the rest.
func TestLegalMatchHasNoInconsistency(t *testing.T) {
	if testing.Short() {
		t.Skip("plays whole matches through the gestures")
	}
	for seed := int64(1); seed <= 4; seed++ {
		length := []int{3, 5}[seed%2]
		ed := playLegalMatch(t, seed, length)
		requireConsistent(t, seed, "typed", ed.Replay(0))
		requireConsistent(t, seed, "replayed in full", Replay(ed.Doc, 0))

		swapped, err := Apply(ed.Doc, Gesture{Kind: GestureSwapPlayers})
		if err != nil {
			t.Fatal(err)
		}
		requireConsistent(t, seed, "players swapped", Replay(swapped, 0))
	}
}

package transcript

import (
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/kevung/gnubgparser"

	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

// matFixtures are the two real .mat files of the repository. One is a gnuBG export of
// a club match (illegal-looking cells included: a "???" play, a game ended by a
// resignation), the other a plain seven-point match — between them they exercise the
// cube, the Crawford game and both ways a game can end.
var matFixtures = []string{
	"../../../testdata/test.mat",
	"../../../testdata/charlot1-charlot2_7p_2025-11-08-2305.mat",
}

// TestFromMATReplayDerivesResults reads each fixture and checks that what the Replay
// DERIVES — the game boundaries, the initial scores, the winners and the points —
// matches what the file states. It is the strongest statement of the state machine:
// the file gives the plays, the package recomputes the outcome of every game from
// them alone.
//
// The oracle is the SCORE LINE of each game, never the parser's Winner/Points: a
// "Wins N points" line standing on its own is credited by gnubgparser to whoever acted
// last (wrong for the two games here that ended on a resignation) and a game ended by
// a drop leaves it no points at all. The score lines carry no such defect, and they
// are what a reader of the .mat sees.
func TestFromMATReplayDerivesResults(t *testing.T) {
	for _, path := range matFixtures {
		t.Run(path, func(t *testing.T) {
			doc, orig := loadFixture(t, path)
			ann := Replay(doc, 0)

			if len(ann.Games) != len(orig.Games) {
				t.Fatalf("games: replay %d, file %d", len(ann.Games), len(orig.Games))
			}
			// A real dance — the roll a closed board allowed nothing for — is
			// recorded as such and carries no Inconsistency. Both fixtures have some.
			dances, clean := 0, 0
			for _, info := range ann.Actions {
				if info.Kind != KindDance {
					continue
				}
				dances++
				if len(info.Inconsistencies) == 0 {
					clean++
				}
			}
			if dances == 0 || clean == 0 {
				t.Errorf("%d dances, %d of them legitimate: the fixture has both", dances, clean)
			}

			for i, og := range orig.Games {
				g := ann.Games[i]
				if g.InitialScore != og.Score {
					t.Errorf("game %d initial score: replay %v, file %v", i+1, g.InitialScore, og.Score)
				}
				if !g.Finished {
					t.Errorf("game %d: the replay leaves it unfinished", i+1)
				}
				if i+1 >= len(orig.Games) {
					continue
				}
				winner, points := scoredResult(orig.Games[i].Score, orig.Games[i+1].Score)
				if g.Winner != winner || g.PointsWon != points {
					t.Errorf("game %d: replay says player %d wins %d, the score lines say player %d wins %d",
						i+1, g.Winner+1, g.PointsWon, winner+1, points)
				}
			}
		})
	}
}

// scoredResult reads a game's outcome off the two score lines that surround it.
func scoredResult(before, after [2]int) (winner, points int) {
	if after[0] > before[0] {
		return 0, after[0] - before[0]
	}
	return 1, after[1] - before[1]
}

// TestFromMATRenderRoundTrip is the format gate: FromMAT → MatchParts →
// ingest.RenderMAT → ParseMAT gives back the graph the file carried. The comparison is
// on the fields of the graph, never on the raw text: the notation of one play has
// several spellings (token order, "bar/off" against "25/0") and the transcript is the
// play, not its spelling.
func TestFromMATRenderRoundTrip(t *testing.T) {
	for _, path := range matFixtures {
		t.Run(path, func(t *testing.T) {
			doc, orig := loadFixture(t, path)

			out := ingest.RenderMAT(MatchParts(doc))
			rt, err := gnubgparser.ParseMAT(strings.NewReader(out))
			if err != nil {
				t.Fatalf("re-parse rendered .mat: %v\n%s", err, out)
			}

			if rt.Metadata.MatchLength != orig.Metadata.MatchLength {
				t.Errorf("match length: %d vs %d", rt.Metadata.MatchLength, orig.Metadata.MatchLength)
			}
			if rt.Metadata.Player1 != orig.Metadata.Player1 || rt.Metadata.Player2 != orig.Metadata.Player2 {
				t.Errorf("players: %q/%q vs %q/%q", rt.Metadata.Player1, rt.Metadata.Player2,
					orig.Metadata.Player1, orig.Metadata.Player2)
			}
			if len(rt.Games) != len(orig.Games) {
				t.Fatalf("games: rendered %d, original %d", len(rt.Games), len(orig.Games))
			}
			for i := range orig.Games {
				og, rg := orig.Games[i], rt.Games[i]
				if rg.Score != og.Score {
					t.Errorf("game %d score: rendered %v, original %v", i+1, rg.Score, og.Score)
				}
				// og.Points is 0 for a game the file ended with a drop — the parser
				// stops before the "Wins" line there. Where it did read one, the
				// rendered file must announce the same.
				if og.Points > 0 && rg.Points != og.Points {
					t.Errorf("game %d points: rendered %d, original %d", i+1, rg.Points, og.Points)
				}
				compareRecords(t, i+1, og.Moves, rg.Moves)
			}
		})
	}
}

func loadFixture(t *testing.T, path string) (Document, *gnubgparser.Match) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	doc, err := FromMAT(string(data))
	if err != nil {
		t.Fatalf("FromMAT %s: %v", path, err)
	}
	orig, err := gnubgparser.ParseMAT(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("ParseMAT %s: %v", path, err)
	}
	return doc, orig
}

// compareRecords checks the two move streams say the same thing: the same kinds in the
// same order, the same dice, and the same checkers moved — the from/to pairs as a set,
// since their order in a cell is a matter of spelling.
func compareRecords(t *testing.T, game int, want, got []gnubgparser.MoveRecord) {
	t.Helper()
	if len(want) != len(got) {
		t.Errorf("game %d: %d records rendered, %d in the file\nwant %s\ngot  %s",
			game, len(got), len(want), summarise(want), summarise(got))
		return
	}
	for i := range want {
		w, g := want[i], got[i]
		if w.Type != g.Type || w.Player != g.Player {
			t.Errorf("game %d record %d: %s/p%d rendered as %s/p%d", game, i+1, w.Type, w.Player, g.Type, g.Player)
			continue
		}
		if w.Type != gnubgparser.MoveTypeNormal {
			continue
		}
		if w.Dice != g.Dice {
			t.Errorf("game %d record %d: dice %v rendered as %v", game, i+1, w.Dice, g.Dice)
		}
		if wp, gp := movePairs(w.Move), movePairs(g.Move); wp != gp {
			t.Errorf("game %d record %d: play %q (%s) rendered as %q (%s)",
				game, i+1, w.MoveString, wp, g.MoveString, gp)
		}
	}
}

func movePairs(move [8]int) string {
	var pairs []string
	for i := 0; i < len(move); i += 2 {
		if move[i] == -1 {
			break
		}
		pairs = append(pairs, string(rune('a'+move[i]))+string(rune('a'+move[i+1]+1)))
	}
	sort.Strings(pairs)
	return strings.Join(pairs, " ")
}

func summarise(records []gnubgparser.MoveRecord) string {
	var b strings.Builder
	for _, r := range records {
		b.WriteString(string(r.Type))
		b.WriteString(" ")
	}
	return b.String()
}

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
		// The decoded Move is all -1 for BOTH a dance and a play the file did not
		// record, so the pairs above say nothing about them: the mark is the only
		// difference, and the round trip has to keep it. A "???" that came back as
		// a dance would put a blank cell in the file where gnubg wrote a question.
		if isUnrecorded(w.MoveString) != isUnrecorded(g.MoveString) {
			t.Errorf("game %d record %d: %q rendered as %q — an unrecorded play and a dance are not the same cell",
				game, i+1, w.MoveString, g.MoveString)
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

// unrecordedFixtureCount is how many "???" cells testdata/test.mat carries — a gnuBG
// export of a club match where three plays were never written down.
const unrecordedFixtureCount = 3

// TestFromMATUnrecordedIsNotADance is the whole point of KindUnrecorded: a .mat cell
// with no play in it means one of two things, and reading both as a dance was a
// falsehood the round trip carried back into the file.
//
// A cell holding only its dice says the player COULD NOT play. A cell holding "???"
// says gnubg did not record what they played. The parser decodes both to an empty
// Move, so the two are told apart by the mark alone.
func TestFromMATUnrecordedIsNotADance(t *testing.T) {
	// A dance and an unrecorded play, side by side, on a board that allows plenty:
	// the opening roll leaves both players everything to play, so a dance here would
	// be flagged illegal and an unrecorded play must not be.
	const text = ` 1 point match

 Game 1
 A : 0                           B : 0
  1) 31: 8/5 6/5                 65:                         
  2) 42: ???                     
`
	doc, err := FromMAT(text)
	if err != nil {
		t.Fatalf("FromMAT: %v", err)
	}
	var kinds []Kind
	for _, a := range doc.Actions {
		if a.Kind != KindOpening {
			kinds = append(kinds, a.Kind)
		}
	}
	want := []Kind{KindChecker, KindDance, KindUnrecorded}
	if len(kinds) != len(want) {
		t.Fatalf("kinds %v, want %v", kinds, want)
	}
	for i := range want {
		if kinds[i] != want[i] {
			t.Fatalf("kinds %v, want %v", kinds, want)
		}
	}

	// The dance is the one the roll contradicts, and it says so; the unrecorded play
	// is reported for what it is and never as an impossible dance.
	ann := Replay(doc, 0)
	found := map[Kind][]InconsistencyKind{}
	for _, info := range ann.Actions {
		for _, bad := range info.Inconsistencies {
			found[info.Kind] = append(found[info.Kind], bad.Kind)
		}
	}
	if got := found[KindDance]; len(got) != 1 || got[0] != IllegalMove {
		t.Errorf("the dance carries %v, want one illegal_move — the roll allowed a play", got)
	}
	if got := found[KindUnrecorded]; len(got) != 1 || got[0] != UnrecordedMove {
		t.Errorf("the unrecorded play carries %v, want one unrecorded_move", got)
	}
	if got := found[KindChecker]; len(got) != 0 {
		t.Errorf("the legal play carries %v, want nothing", got)
	}
}

// TestFromMATUnrecordedFixture holds the real file: the three "???" of test.mat are
// read as unrecorded plays, each one reported, and no dance of the file is called
// impossible any more.
func TestFromMATUnrecordedFixture(t *testing.T) {
	doc, _ := loadFixture(t, "../../../testdata/test.mat")
	ann := Replay(doc, 0)

	unrecorded, reported := 0, 0
	for _, info := range ann.Actions {
		for _, bad := range info.Inconsistencies {
			if bad.Kind == IllegalMove && info.Kind == KindDance {
				t.Errorf("action %d: a dance of the file is called impossible — %s", info.Index, bad.Detail)
			}
		}
		if info.Kind != KindUnrecorded {
			continue
		}
		unrecorded++
		if info.Notation != UnrecordedNotation {
			t.Errorf("action %d: notation %q, want %q", info.Index, info.Notation, UnrecordedNotation)
		}
		for _, bad := range info.Inconsistencies {
			if bad.Kind == UnrecordedMove {
				reported++
			}
		}
	}
	if unrecorded != unrecordedFixtureCount || reported != unrecordedFixtureCount {
		t.Errorf("%d unrecorded plays, %d reported: the fixture has %d",
			unrecorded, reported, unrecordedFixtureCount)
	}
}

// TestRenderMATGivesBackTheQuestionMarks is the round trip on the text itself: what
// went in as "???" comes out as "???", in the same number. Rendering it as a dance
// would write a .mat that claims the player could not move.
func TestRenderMATGivesBackTheQuestionMarks(t *testing.T) {
	doc, _ := loadFixture(t, "../../../testdata/test.mat")
	out := ingest.RenderMAT(MatchParts(doc))
	if n := strings.Count(out, UnrecordedNotation); n != unrecordedFixtureCount {
		t.Errorf("%d %q cells rendered, want %d:\n%s", n, UnrecordedNotation, unrecordedFixtureCount, out)
	}
}

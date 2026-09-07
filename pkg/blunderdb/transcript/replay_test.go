package transcript

import (
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// bearingOffBoard returns a board where one side has borne off everything and the
// other has borne off nothing, its fifteen checkers standing wherever `points` says.
func bearingOffBoard(winner int, points map[int]int) domain.Board {
	b := domain.Board{}
	for i := range b.Points {
		b.Points[i] = domain.Point{Color: domain.None}
	}
	loser := opponent(winner)
	for idx, n := range points {
		b.Points[idx] = domain.Point{Checkers: n, Color: loser}
	}
	b.Bearoff[winner] = domain.CheckersPerPlayer
	return b
}

// TestGameEndByBearingOff covers the end-of-game arithmetic of fonctionnel.md §1.3: a
// single, a gammon, a backgammon, each times the value of the cube. The board is
// reached through an Action the Replay calls illegal, which is exactly the point — an
// illegal board is where the game goes on from, and the rules are applied to it.
func TestGameEndByBearingOff(t *testing.T) {
	tests := []struct {
		name   string
		board  domain.Board
		cube   bool // the cube was doubled and taken first
		points int
	}{
		{
			name: "single: the loser bore a checker off",
			board: func() domain.Board {
				b := bearingOffBoard(domain.Black, map[int]int{20: 14})
				b.Bearoff[domain.White] = 1
				return b
			}(),
			points: 1,
		},
		{
			name:   "gammon: nothing off, nothing trapped",
			board:  bearingOffBoard(domain.Black, map[int]int{20: 15}),
			points: 2,
		},
		{
			name:   "backgammon: a checker in the winner's home board",
			board:  bearingOffBoard(domain.Black, map[int]int{20: 14, 3: 1}),
			points: 3,
		},
		{
			name:   "backgammon: a checker on the bar",
			board:  bearingOffBoard(domain.Black, map[int]int{20: 14, domain.WhiteBar: 1}),
			points: 3,
		},
		{
			name:   "and the cube multiplies it",
			board:  bearingOffBoard(domain.Black, map[int]int{20: 15}),
			cube:   true,
			points: 4,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc := docOf(7, opening(domain.Black, 6, 3))
			if tc.cube {
				doc.Actions = append(doc.Actions,
					Action{Side: domain.Black, Kind: KindDouble},
					Action{Side: domain.White, Kind: KindTake},
				)
			}
			board := tc.board
			doc.Actions = append(doc.Actions, Action{
				Side: domain.Black, Kind: KindChecker, Dice: [2]int{6, 3},
				Steps:      []domain.CheckerStep{{From: 6, To: domain.Off}},
				BoardAfter: &board,
			})
			ann := Replay(doc, 0)
			g := ann.Games[0]
			if g.Winner != domain.Black || g.PointsWon != tc.points {
				t.Fatalf("game = player %d wins %d, want player 1 winning %d", g.Winner+1, g.PointsWon, tc.points)
			}
			if !g.Finished || ann.Next.Expects != KindOpening {
				t.Errorf("the game did not close: %+v / next %s", g, ann.Next.Expects)
			}
		})
	}
}

// TestPostCrawfordSentinel checks the away score written after the Crawford game:
// 0 rather than 1, which is the whole of the difference between "needs one point" and
// "needs one point, Crawford behind us" (CONTEXT.md, ADR-0045 rule 7).
func TestPostCrawfordSentinel(t *testing.T) {
	doc := docOf(2,
		opening(domain.Black, 6, 3),
		Action{Side: domain.White, Kind: KindResign, Level: 1}, // 1-0, game 2 is Crawford
		opening(domain.Black, 5, 2),
		Action{Side: domain.Black, Kind: KindResign, Level: 1}, // 1-1, game 3 is post-Crawford
		opening(domain.Black, 4, 1),
	)
	ann := Replay(doc, 0)
	if len(ann.Games) != 3 {
		t.Fatalf("games = %d", len(ann.Games))
	}
	if !ann.Games[1].Crawford || ann.Games[2].Crawford {
		t.Fatalf("Crawford is on the wrong game: %+v", ann.Games)
	}
	if got := ann.Actions[2].Before.Score; got != [2]int{domain.Crawford, 2} {
		t.Errorf("Crawford away score = %v, want [1 2]", got)
	}
	if got := ann.Actions[4].Before.Score; got != [2]int{domain.PostCrawford, domain.PostCrawford} {
		t.Errorf("post-Crawford away score = %v, want [0 0]", got)
	}
}

// TestReplayLatency measures a full Replay of a 300-Action document. The threshold is
// a REGRESSION guard on this package, measured and not estimated.
//
// The T0.2 sheet asked for 20 ms, which was never what a full Replay costs: it is
// domain.LegalMoves, called once per checker Action (175 µs for an ordinary roll from
// the opening position, 3.6 ms for a double), and nothing in this package closes that
// gap — it would take a faster legal-move generator, which is a change to domain with
// its own differential test to answer to. Measured: 27 ms here, 101 ms on the machine
// the sheet was written on, i.e. some 90 to 340 µs per Action. The ceiling below is
// five times the slower of the two, and the figure the test logs is the one to argue
// about.
//
// What the interactive budget rests on is NOT this number: typing replays one Action
// through a [Replayer], which TestReplayIncrementalCostsOneAction measures.
func TestReplayLatency(t *testing.T) {
	const actions = 300
	const ceiling = 500 * time.Millisecond

	data, err := os.ReadFile(matFixtures[0])
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	doc, err := FromMAT(string(data))
	if err != nil {
		t.Fatalf("FromMAT: %v", err)
	}
	if len(doc.Actions) < actions {
		t.Fatalf("the fixture has %d actions, %d wanted", len(doc.Actions), actions)
	}
	doc.Actions = doc.Actions[:actions]
	doc.Cursor = len(doc.Actions)

	best := time.Duration(1<<63 - 1)
	for i := 0; i < 3; i++ {
		start := time.Now()
		ann := Replay(doc, 0)
		if elapsed := time.Since(start); elapsed < best {
			best = elapsed
		}
		if len(ann.Actions) != actions {
			t.Fatalf("replayed %d actions", len(ann.Actions))
		}
	}
	t.Logf("Replay of %d Actions: %v (%v per Action)", actions, best, best/actions)
	if best > ceiling {
		t.Errorf("Replay of %d Actions took %v, over the %v guard", actions, best, ceiling)
	}
}

// TestPackageIsPure holds the one architectural property this package was given: its
// single internal dependency is domain. Importing ingest would drag storage in behind
// it (ingest/match.go), and with it the persistence this package exists to stay out of.
func TestPackageIsPure(t *testing.T) {
	forbidden := []string{
		"blunderdb/pkg/blunderdb/ingest",
		"blunderdb/pkg/blunderdb/storage",
		"blunderdb/pkg/blunderdb/database",
		"blunderdb/internal/",
	}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read the package directory: %v", err)
	}
	fset := token.NewFileSet()
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, imp := range file.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			for _, bad := range forbidden {
				if strings.Contains(path, bad) {
					t.Errorf("%s imports %s", name, path)
				}
			}
		}
	}
}

// TestReplayIncrementalCostsOneAction is the measurement the entry loop needs: adding
// an Action at the end of a long document must cost ONE Action, not the document.
//
// The ux.md §4.1 budget is 0.56 s for a whole turn, keystrokes included. A full Replay
// of a 300-Action match spends most of that on its own, at every keystroke; the
// ceiling below is what a single Action costs (some 340 µs at worst) with room for a
// loaded machine, and it is two orders of magnitude under a full Replay.
func TestReplayIncrementalCostsOneAction(t *testing.T) {
	const actions = 300
	const ceiling = 5 * time.Millisecond

	data, err := os.ReadFile(matFixtures[0])
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	doc, err := FromMAT(string(data))
	if err != nil {
		t.Fatalf("FromMAT: %v", err)
	}
	if len(doc.Actions) < actions {
		t.Fatalf("the fixture has %d actions, %d wanted", len(doc.Actions), actions)
	}
	full := doc
	full.Actions = doc.Actions[:actions]
	full.Cursor = actions
	short := doc
	short.Actions = doc.Actions[:actions-1]
	short.Cursor = actions - 1

	// The best of three: the measurement wanted is the work done, not the scheduler.
	best := time.Duration(1<<63 - 1)
	for i := 0; i < 3; i++ {
		var r Replayer
		r.Replay(short, 0) // the document as it stood before the last keystroke
		start := time.Now()
		ann := r.Replay(full, 0)
		if elapsed := time.Since(start); elapsed < best {
			best = elapsed
		}
		if len(ann.Actions) != actions {
			t.Fatalf("replayed %d actions", len(ann.Actions))
		}
	}
	t.Logf("one Action appended to a document of %d: %v", actions, best)
	if best > ceiling {
		t.Errorf("appending one Action to %d took %v, over the %v guard — the replay is not incremental",
			actions, best, ceiling)
	}

	// Correcting an Action replays it and everything after it, and nothing before.
	// Correcting the LAST one must therefore cost what appending one costs.
	var r Replayer
	r.Replay(full, 0)
	corrected := full.clone()
	corrected.Actions[actions-1].Side = opponent(corrected.Actions[actions-1].Side)
	start := time.Now()
	r.Replay(corrected, 0)
	if elapsed := time.Since(start); elapsed > ceiling {
		t.Errorf("correcting the last of %d Actions took %v, over the %v guard", actions, elapsed, ceiling)
	}
}

// kitchenSinkDoc is a document that goes through every Kind and raises every
// InconsistencyKind, unknown kinds included — the widest state machine a Replay can be
// asked to walk, and the one an incremental replay has to agree with.
func kitchenSinkDoc(t *testing.T) Document {
	t.Helper()
	// The board a play that moves one checker for a 63 really leaves.
	board := InitialBoard()
	board.Points[24] = domain.Point{Checkers: 1, Color: domain.Black}
	board.Points[18] = domain.Point{Checkers: 1, Color: domain.Black}

	// A one-point match: everything after the first game is played past the end.
	doc := docOf(1,
		opening(domain.Black, 3, 3), // a tie: another opening follows, same game
		opening(domain.Black, 6, 3),
		Action{Side: domain.Black, Kind: KindChecker, Dice: [2]int{6, 3},
			Steps: []domain.CheckerStep{{From: 24, To: 18}}, BoardAfter: &board}, // illegal
		Action{Side: domain.White, Kind: KindDance, Dice: [2]int{6, 5}}, // the roll allows a play
		Action{Side: domain.White, Kind: KindDouble},                    // and player 2 acts twice
		Action{Side: domain.Black, Kind: KindDouble},                    // an answer is already pending
		Action{Side: domain.White, Kind: KindTake},
		Action{Side: domain.Black, Kind: KindChecker, Dice: [2]int{2, 1},
			Steps: []domain.CheckerStep{{From: 13, To: 8}}}, // the play does not use the roll
		Action{Side: domain.White, Kind: KindPass}, // nothing to answer: the game ends here
		opening(domain.Black, 5, 2),                // past the end from now on
		Action{Side: domain.Black, Kind: KindResign, Level: 2},
		Action{Side: domain.White, Kind: "no such kind"},
	)

	ann := Replay(doc, 0)
	kinds := map[Kind]bool{}
	found := map[InconsistencyKind]bool{}
	for i, a := range doc.Actions {
		kinds[a.Kind] = true
		for _, inc := range ann.Actions[i].Inconsistencies {
			found[inc.Kind] = true
		}
	}
	for _, k := range []Kind{KindOpening, KindChecker, KindDance, KindDouble, KindTake, KindPass, KindResign} {
		if !kinds[k] {
			t.Fatalf("the fixture no longer covers kind %q", k)
		}
	}
	for _, k := range []InconsistencyKind{IllegalMove, DoubleTurn, ImpossibleCube, PastEnd, InconsistentDice} {
		if !found[k] {
			t.Fatalf("the fixture no longer raises %q", k)
		}
	}
	return doc
}

// TestReplayIncrementalMatchesFull is the soundness of the cache: a Replayer grown one
// Action at a time, then corrected in the middle, returns exactly what a Replay from
// scratch returns — on a document that covers every Kind and every Inconsistency.
//
// It is the property the whole optimisation rests on, and it is stated on the WHOLE
// Annotated, not on a chosen field: the derivation of an Action depends on the state
// left by the previous one and on nothing else, or this test goes red.
func TestReplayIncrementalMatchesFull(t *testing.T) {
	doc := kitchenSinkDoc(t)

	// Grown from nothing, one Action at a time — the entry loop.
	var r Replayer
	for n := 0; n <= len(doc.Actions); n++ {
		grown := doc
		grown.Actions = doc.Actions[:n]
		grown.Cursor = n
		if got, want := r.Replay(grown, 0), Replay(grown, 0); !reflect.DeepEqual(got, want) {
			t.Fatalf("incremental replay of the first %d Actions differs from a full one", n)
		}
	}

	// Corrected at every index in turn, on a Replayer that has just seen the whole
	// document — the correction loop.
	for n := range doc.Actions {
		corrected := doc.clone()
		corrected.Actions[n].Side = opponent(corrected.Actions[n].Side)
		if got, want := r.Replay(corrected, 0), Replay(corrected, 0); !reflect.DeepEqual(got, want) {
			t.Fatalf("incremental replay after correcting Action %d differs from a full one", n)
		}
		// And back, so the next round starts from the document the cache describes.
		if got, want := r.Replay(doc, 0), Replay(doc, 0); !reflect.DeepEqual(got, want) {
			t.Fatalf("incremental replay after undoing the correction at %d differs from a full one", n)
		}
	}

	// The header the Replay reads is part of the key: changing the match length, and
	// only it, must throw the cache away.
	longer := doc.clone()
	longer.Header.MatchLength = 7
	if got, want := r.Replay(longer, 0), Replay(longer, 0); !reflect.DeepEqual(got, want) {
		t.Fatal("incremental replay after a match-length change differs from a full one")
	}
	// A header field no Replay reads must NOT: it derives nothing.
	named := longer.clone()
	named.Header.Player1 = "Alice"
	if got, want := r.Replay(named, 0), Replay(named, 0); !reflect.DeepEqual(got, want) {
		t.Fatal("incremental replay after a player-name change differs from a full one")
	}
	if r.reusable(named) != len(named.Actions) {
		t.Errorf("naming a player threw the cache away: %d of %d Actions kept",
			r.reusable(named), len(named.Actions))
	}

	// `from` still chooses only where the Cursor lands, on either path.
	for _, from := range []int{-1, 0, 3, len(doc.Actions)} {
		if got, want := r.Replay(doc, from), Replay(doc, from); !reflect.DeepEqual(got, want) {
			t.Fatalf("incremental replay with from=%d differs from a full one", from)
		}
	}
}

// TestCubeFlowsThroughAMatch walks fonctionnel.md §6, flows 5 to 11, on one
// document: a double taken, a redouble passed, the score that follows, the game
// after it, and — on a match short enough to reach it — the Crawford game and the
// end of the match.
//
// The single-flow pieces are already held elsewhere (the gestures test for one
// double and one take, the inconsistency test for a double in the Crawford game).
// What is held HERE is their sequence, which is the thing a transcription
// actually is: each derivation feeds the next, and a score that advances by the
// wrong amount is invisible until the game after it.
func TestCubeFlowsThroughAMatch(t *testing.T) {
	// Flow 5 — double, take: the cube goes to the taker at the doubled value and
	// the DOUBLER rolls next.
	doc := docOf(7, opening(domain.Black, 6, 3))
	doc.Actions = append(doc.Actions, firstCandidate(t, doc, domain.Black, 6, 3))
	doc.Actions = append(doc.Actions,
		Action{Side: domain.White, Kind: KindDouble},
		Action{Side: domain.Black, Kind: KindTake},
	)
	doc.Cursor = len(doc.Actions)

	ann := Replay(doc, 0)
	if got := ann.Next.Position.Cube; got.Owner != domain.Black || got.Value != 1 {
		t.Fatalf("after the take the cube is %+v, want player 1 owning 2", got)
	}
	if ann.Next.Expects != KindChecker || ann.Next.Side != domain.White {
		t.Fatalf("after the take, next = %+v, want the doubler's roll", ann.Next)
	}
	if ann.Inconsistent() {
		t.Fatalf("an ordinary double and take are consistent: %+v", ann.Actions)
	}

	// Flow 7 — redouble: the cube is held by player 1, who may turn it again.
	// Flow 6 — pass: the game is won at the value the cube had BEFORE the offer.
	doc.Actions = append(doc.Actions,
		Action{Side: domain.Black, Kind: KindDouble},
		Action{Side: domain.White, Kind: KindPass},
	)
	doc.Cursor = len(doc.Actions)

	ann = Replay(doc, 0)
	if ann.Inconsistent() {
		t.Fatalf("the owner of the cube may redouble: %+v", ann.Actions)
	}
	g := ann.Games[0]
	if g.Winner != domain.Black || g.PointsWon != 2 || !g.Finished {
		t.Fatalf("game 1 = %+v, want player 1 winning 2 points", g)
	}
	if ann.Score != [2]int{2, 0} {
		t.Fatalf("score = %v, want [2 0]", ann.Score)
	}

	// Flow 10 — the next game: its number, its initial score and the away score
	// the next Position carries are all derived before a single Action of it.
	if ann.Next.Expects != KindOpening || ann.Next.GameNumber != 2 {
		t.Fatalf("next = %+v, want the opening of game 2", ann.Next)
	}
	if ann.Next.Crawford {
		t.Error("2-0 in a 7-point match is not the Crawford game")
	}
	if got := ann.Next.Position.Score; got != [2]int{5, 7} {
		t.Errorf("away score = %v, want [5 7]", got)
	}

	// Flow 11 — the end of the match, on a length short enough to reach by the
	// cube alone: a double passed in game 1 puts player 2 one point from a
	// two-point match, so game 2 is the Crawford game and a double in it is
	// impossible; a second pass ends the match.
	short := docOf(2,
		opening(domain.Black, 6, 3),
		Action{Side: domain.Black, Kind: KindDouble},
		Action{Side: domain.White, Kind: KindPass},
		opening(domain.Black, 5, 2),
		Action{Side: domain.Black, Kind: KindDouble},
		Action{Side: domain.White, Kind: KindPass},
	)
	ann = Replay(short, 0)
	if len(ann.Games) != 2 {
		t.Fatalf("games = %d, want 2", len(ann.Games))
	}
	if !ann.Games[1].Crawford {
		t.Errorf("game 2 is the Crawford game: %+v", ann.Games)
	}
	if !hasInconsistency(ann.Actions[4], ImpossibleCube) {
		t.Error("the cube is dead in the Crawford game; the double was not marked")
	}
	if !ann.Finished || ann.Winner != domain.Black || ann.Score != [2]int{2, 0} {
		t.Errorf("match = finished %v winner %d score %v, want player 1 at 2-0", ann.Finished, ann.Winner, ann.Score)
	}
	if !ann.Next.MatchOver {
		t.Error("the match is won and Next does not say so")
	}
}

// TestResignationIsAGameFactNotAMove holds the two properties of ADR-0045 §6 that
// nothing else states, and that are both invisible until a match is saved.
//
// A resignation adds no row to `move`: the Game carries the winner and the points,
// and the .mat format has no token for it anyway. And it is not a double turn —
// a player gives up at any moment, their own roll included, so counting it would
// mark every resignation that follows its author's last play.
//
// The abandoned match is the third: a draft left in the middle is an UNFINISHED
// Match, `winner = -1` on its last game, which is what the domain's Game already
// means by -1 and what a save must carry through untouched.
func TestResignationIsAGameFactNotAMove(t *testing.T) {
	doc := docOf(7, opening(domain.Black, 6, 3))
	doc.Actions = append(doc.Actions, firstCandidate(t, doc, domain.Black, 6, 3))
	// Player 1 has just played, and player 1 gives the game up: two Actions of the
	// same side in a row, and NOT a double turn.
	doc.Actions = append(doc.Actions, Action{Side: domain.Black, Kind: KindResign, Level: 2})
	doc.Cursor = len(doc.Actions)

	ann := Replay(doc, 0)
	if hasInconsistency(ann.Actions[2], DoubleTurn) {
		t.Error("a resignation after its author's own play was counted as a double turn")
	}
	if g := ann.Games[0]; g.Winner != domain.White || g.PointsWon != 2 {
		t.Fatalf("game = %+v, want player 2 winning a gammon", g)
	}
	if ann.Score != [2]int{0, 2} {
		t.Errorf("score = %v, want [0 2]", ann.Score)
	}
	// The resignation itself produces no Move, and no Position either.
	if info := ann.Actions[2]; info.MoveNumber != -1 || info.HasPosition {
		t.Errorf("the resignation produced a move slot: %+v", info)
	}
	_, games, moves := MatchParts(doc)
	if n := len(moves[games[0].ID]); n != 1 {
		t.Errorf("moves of game 1 = %d, want the single checker play", n)
	}

	// A match abandoned mid-game: the last Game is unfinished and says so.
	doc.Actions = append(doc.Actions, opening(domain.Black, 5, 2))
	doc.Actions = append(doc.Actions, firstCandidate(t, doc, domain.Black, 5, 2))
	doc.Cursor = len(doc.Actions)

	ann = Replay(doc, 0)
	if len(ann.Games) != 2 || ann.Games[1].Finished {
		t.Fatalf("games = %+v, want an unfinished second game", ann.Games)
	}
	_, games, _ = MatchParts(doc)
	if games[1].Winner != -1 || games[1].PointsWon != 0 {
		t.Errorf("abandoned game = winner %d, %d points; want -1 and 0", games[1].Winner, games[1].PointsWon)
	}
	if ann.Finished {
		t.Error("an abandoned match is not a finished one")
	}
}

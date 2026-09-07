// Contract cases for the "positions like this one" scan.
// The table that runs them lives in contract.go.
package storagetest

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testSimilarIsExactAndOrdered pins the two promises the contract makes about
// Similar, both of which an approximate index would break: the neighbours come
// back NEAREST FIRST, and the scan is exhaustive — so a position that is
// closer is never missed (issue #293).
//
// It also pins the one exclusion: a position is not its own neighbour. Asking
// "what is like this?" and being handed the thing itself is a non-answer, and
// it is the kind of thing a query written against the whole table gets wrong.
func testSimilarIsExactAndOrdered(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ps := s.Positions()

	// One reference and three variations, at growing distance from it: one
	// checker moved one pip, then three, then six.
	base := similarityBoard(map[int]int{13: 5, 8: 5, 6: 5})
	near := similarityBoard(map[int]int{13: 4, 12: 1, 8: 5, 6: 5})
	middle := similarityBoard(map[int]int{13: 4, 10: 1, 8: 5, 6: 5})
	far := similarityBoard(map[int]int{13: 4, 7: 1, 8: 5, 6: 5})

	baseID, err := ps.Save(ctx, "", &base)
	if err != nil {
		t.Fatalf("Save base: %v", err)
	}
	nearID, err := ps.Save(ctx, "", &near)
	if err != nil {
		t.Fatalf("Save near: %v", err)
	}
	middleID, err := ps.Save(ctx, "", &middle)
	if err != nil {
		t.Fatalf("Save middle: %v", err)
	}
	if _, err := ps.Save(ctx, "", &far); err != nil {
		t.Fatalf("Save far: %v", err)
	}

	base.ID = baseID
	got, err := rank(ctx, s, baseID, 2, 0, false)
	if err != nil {
		t.Fatalf("Similar: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("Similar returned %d neighbours, want 2", len(got))
	}
	if got[0].Position.ID != nearID || got[1].Position.ID != middleID {
		t.Errorf("neighbours must come back nearest first: got %d then %d, want %d then %d",
			got[0].Position.ID, got[1].Position.ID, nearID, middleID)
	}
	if got[0].Distance >= got[1].Distance {
		t.Errorf("distances must grow: got %d then %d", got[0].Distance, got[1].Distance)
	}
	for _, n := range got {
		if n.Position.ID == baseID {
			t.Error("a position is not its own neighbour")
		}
	}
}

// testSimilarRanksInsideTheClass pins what a neighbour IS (ADR-0043): the same
// PROBLEM nearby, not the nearest drawing.
//
// Ranking the whole library by distance alone answered a question nobody
// asked. Measured on the demo library, the nearest of any position was the
// checker play twinning its cube decision — the same board, distance zero, two
// rows — and the next ones were the plies before and after it in the same
// match, because two plies are one roll and no other game comes that close.
// Each case below is one of those three ways of being close and useless.
func testSimilarRanksInsideTheClass(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ps := s.Positions()

	board := map[int]int{13: 5, 8: 5, 6: 5}
	near := map[int]int{13: 4, 12: 1, 8: 5, 6: 5}

	// The target: a cube decision at a match score.
	target := similarityBoard(board)
	target.DecisionType = domain.CubeAction
	target.Dice = [2]int{0, 0}
	target.Score = [2]int{3, 5}
	targetID, err := ps.Save(ctx, "", &target)
	if err != nil {
		t.Fatalf("Save target: %v", err)
	}
	target.ID = targetID

	// The twin: the SAME board, one pip away, as a checker play. Distance ~1,
	// and a different problem entirely.
	twin := similarityBoard(near)
	twin.DecisionType = domain.CheckerAction
	twin.Dice = [2]int{3, 1}
	twin.Score = [2]int{3, 5}
	twinID, err := ps.Save(ctx, "", &twin)
	if err != nil {
		t.Fatalf("Save twin: %v", err)
	}

	// Same board, same kind of decision, but played for money: the regime
	// changes what a cube decision asks, so it is not the same problem.
	moneyTwin := similarityBoard(near)
	moneyTwin.DecisionType = domain.CubeAction
	moneyTwin.Dice = [2]int{0, 0}
	moneyTwin.Score = [2]int{-1, -1}
	moneyID, err := ps.Save(ctx, "", &moneyTwin)
	if err != nil {
		t.Fatalf("Save money twin: %v", err)
	}

	// A legitimate neighbour: same kind, same regime, further away.
	elsewhere := similarityBoard(map[int]int{13: 4, 10: 1, 8: 5, 6: 5})
	elsewhere.DecisionType = domain.CubeAction
	elsewhere.Dice = [2]int{0, 0}
	elsewhere.Score = [2]int{2, 4}
	elsewhereID, err := ps.Save(ctx, "", &elsewhere)
	if err != nil {
		t.Fatalf("Save elsewhere: %v", err)
	}

	got, err := rankWidened(ctx, s, targetID, 10)
	if err != nil {
		t.Fatalf("Similar: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("with no class asked for, every position is a candidate: got %d neighbours, want 3", len(got))
	}

	got, err = rank(ctx, s, targetID, 10, 0, false)
	if err != nil {
		t.Fatalf("Similar in class: %v", err)
	}
	ids := map[int64]bool{}
	for _, n := range got {
		ids[n.Position.ID] = true
	}
	if ids[twinID] {
		t.Error("the checker play on the same board is not a neighbour of a cube decision: it is the other question the board asks")
	}
	if ids[moneyID] {
		t.Error("a money cube decision is not a neighbour of one at a match score: the regime is what the position asks about")
	}
	if !ids[elsewhereID] {
		t.Error("a cube decision at another score, of the same regime, IS a neighbour and was dropped")
	}
}

// testSimilarExcludesTheTargetsMatches pins the third rule of the class: the
// plies around a position, in every match that played through it, are its
// closest structures and never its neighbours (ADR-0043).
//
// Two plies are one roll — eight to sixteen checker-pips — so on a library of
// imported matches they crowd out everything else, and "positions like this
// one" answers "here is the game you are looking at".
func testSimilarExcludesTheTargetsMatches(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ps, ms := s.Positions(), s.Matches()

	target := similarityBoard(map[int]int{13: 5, 8: 5, 6: 5})
	neighbourInMatch := similarityBoard(map[int]int{13: 4, 12: 1, 8: 5, 6: 5})
	neighbourElsewhere := similarityBoard(map[int]int{13: 4, 10: 1, 8: 5, 6: 5})

	targetID, err := ps.Save(ctx, "", &target)
	if err != nil {
		t.Fatalf("Save target: %v", err)
	}
	target.ID = targetID
	insideID, err := ps.Save(ctx, "", &neighbourInMatch)
	if err != nil {
		t.Fatalf("Save neighbour in match: %v", err)
	}
	outsideID, err := ps.Save(ctx, "", &neighbourElsewhere)
	if err != nil {
		t.Fatalf("Save neighbour elsewhere: %v", err)
	}

	// Both the target and its nearest neighbour are plies of the same match:
	// exactly the arrangement a real import produces.
	matchID, err := ms.Save(ctx, "", &domain.Match{Player1Name: "A", Player2Name: "B", MatchLength: 7})
	if err != nil {
		t.Fatalf("Save match: %v", err)
	}
	gameID, err := ms.CreateGame(ctx, "", &domain.Game{MatchID: matchID, GameNumber: 1})
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	for i, pid := range []int64{targetID, insideID} {
		mv := domain.Move{GameID: gameID, MoveNumber: int32(i + 1), PositionID: pid, MoveType: "checker"}
		if _, err := ms.CreateMove(ctx, "", &mv); err != nil {
			t.Fatalf("CreateMove: %v", err)
		}
	}

	got, err := rank(ctx, s, targetID, 10, 0, false)
	if err != nil {
		t.Fatalf("Similar: %v", err)
	}
	for _, n := range got {
		if n.Position.ID == insideID {
			t.Error("a ply of the target's own match is not a neighbour: it is the game being looked at")
		}
	}
	found := false
	for _, n := range got {
		if n.Position.ID == outsideID {
			found = true
		}
	}
	if !found {
		t.Error("a position outside the target's matches IS a neighbour and was dropped")
	}
}

// testSimilarCeilingLeavesAnEmptyRankingEmpty pins the honest answer: when
// nothing stands close enough, the ranking is EMPTY (ADR-0043 rule 4).
//
// A fixed count alone hands back the least distant of the unrelated, which on
// a small library is ten positions with nothing to do with the question and a
// figure in the status bar as the only warning.
func testSimilarCeilingLeavesAnEmptyRankingEmpty(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ps := s.Positions()

	target := similarityBoard(map[int]int{13: 5, 8: 5, 6: 5})
	far := similarityBoard(map[int]int{2: 5, 8: 5, 6: 5})

	targetID, err := ps.Save(ctx, "", &target)
	if err != nil {
		t.Fatalf("Save target: %v", err)
	}
	target.ID = targetID
	if _, err := ps.Save(ctx, "", &far); err != nil {
		t.Fatalf("Save far: %v", err)
	}

	loose, err := rank(ctx, s, targetID, 10, 0, false)
	if err != nil {
		t.Fatalf("Similar without a ceiling: %v", err)
	}
	if len(loose) == 0 {
		t.Fatal("without a ceiling the far position is still a neighbour")
	}

	tight, err := rank(ctx, s, targetID, 10, 1, false)
	if err != nil {
		t.Fatalf("Similar with a ceiling: %v", err)
	}
	if len(tight) != 0 {
		t.Errorf("a ceiling nothing passes returns nothing, never the least distant: got %d neighbours", len(tight))
	}

	exact, err := rank(ctx, s, targetID, 10, loose[0].Distance, false)
	if err != nil {
		t.Fatalf("Similar at the exact distance: %v", err)
	}
	if len(exact) != 1 {
		t.Errorf("the ceiling is inclusive: got %d neighbours at exactly %d checker-pips, want 1", len(exact), loose[0].Distance)
	}
}

// testSimilarWithoutAMatchExcludesNothing pins the case a drawn board and an
// individually imported position share: belonging to no match, they have no
// plies to exclude, and the rule costs them nothing.
func testSimilarWithoutAMatchExcludesNothing(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ps := s.Positions()

	target := similarityBoard(map[int]int{13: 5, 8: 5, 6: 5})
	near := similarityBoard(map[int]int{13: 4, 12: 1, 8: 5, 6: 5})
	targetID, err := ps.Save(ctx, "", &target)
	if err != nil {
		t.Fatalf("Save target: %v", err)
	}
	nearID, err := ps.Save(ctx, "", &near)
	if err != nil {
		t.Fatalf("Save near: %v", err)
	}

	got, err := rank(ctx, s, targetID, 10, 0, false)
	if err != nil {
		t.Fatalf("Rank: %v", err)
	}
	found := false
	for _, n := range got {
		if n.Position.ID == nearID {
			found = true
		}
	}
	if !found {
		t.Error("a position belonging to no match excludes nothing, so the near one is a neighbour")
	}
}

// testSimilarRanksAgainstADrawnBoard pins the target that was never stored:
// a board the user has merely DRAWN (ADR-0043 rule 3).
//
// It is the question the exact structure search believed it was asking — "I
// vaguely remember a position like this" — and the one the structure filter
// cannot answer, because it does not forgive an approximate drawing. The board
// is read as a POSITION and not as a pattern: a point left empty counts as
// checkers borne off, which is right for a real position.
func testSimilarRanksAgainstADrawnBoard(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ps := s.Positions()

	near := similarityBoard(map[int]int{13: 4, 12: 1, 8: 5, 6: 5})
	far := similarityBoard(map[int]int{2: 5, 8: 5, 6: 5})
	nearID, err := ps.Save(ctx, "", &near)
	if err != nil {
		t.Fatalf("Save near: %v", err)
	}
	if _, err := ps.Save(ctx, "", &far); err != nil {
		t.Fatalf("Save far: %v", err)
	}

	// Never saved: this board exists only on the screen.
	drawn := similarityBoard(map[int]int{13: 5, 8: 5, 6: 5})
	got, err := s.Search().Rank(ctx, "", domain.SearchFilters{
		LikeFilter:      true,
		LikeTargetBoard: drawn,
	}, storage.ListOpts{Limit: 10})
	if err != nil {
		t.Fatalf("Rank against a drawn board: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("a drawn board is a legitimate target: it ranked nothing")
	}
	if got[0].Position.ID != nearID {
		t.Errorf("nearest to the drawn board = %d, want %d", got[0].Position.ID, nearID)
	}
	// Nothing is excluded: a drawing belongs to no match, and is its own
	// neighbour of nobody.
	if len(got) != 2 {
		t.Errorf("a drawing excludes no match, so both stored positions are neighbours: got %d", len(got))
	}
}

// rank runs a ranked query the way every caller does: the `like` token, its
// target already resolved to an id, and the class the target imposes.
func rank(ctx context.Context, s storage.Storage, targetID int64, limit, maxDistance int, widened bool) ([]storage.SimilarPosition, error) {
	return s.Search().Rank(ctx, "", domain.SearchFilters{
		LikeFilter:      true,
		LikeTargetID:    targetID,
		LikeMaxDistance: maxDistance,
		LikeWidened:     widened,
	}, storage.ListOpts{Limit: limit})
}

// rankWidened is the `*` form: the match exclusion still applies, but nothing
// else does — every kind of decision, both regimes.
func rankWidened(ctx context.Context, s storage.Storage, targetID int64, limit int) ([]storage.SimilarPosition, error) {
	return rank(ctx, s, targetID, limit, 0, true)
}

// similarityBoard builds a money-game checker position from Black's points,
// with White standing clear of every point Black uses.
func similarityBoard(black map[int]int) domain.Position {
	var p domain.Position
	for i := range p.Board.Points {
		p.Board.Points[i] = domain.Point{Checkers: 0, Color: domain.None}
	}
	for pt, n := range black {
		p.Board.Points[pt] = domain.Point{Checkers: n, Color: domain.Black}
	}
	for _, pt := range []int{17, 19, 21} {
		p.Board.Points[pt] = domain.Point{Checkers: 5, Color: domain.White}
	}
	p.PlayerOnRoll = domain.Black
	p.Dice = [2]int{3, 1}
	p.Score = [2]int{-1, -1}
	return p
}

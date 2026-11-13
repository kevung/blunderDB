// Contract cases for "positions like this one": the distance in plays.
// The table that runs them lives in contract.go.
package storagetest

import (
	"context"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/reach"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// The opening, then Black's 3-1 (8/5 6/5), then White's 6-4 (8/2 6/2 in its
// own numbering), and a race no two plays lead to or from.
var (
	openingBlack = map[int]int{24: 2, 13: 5, 8: 3, 6: 5}
	openingWhite = map[int]int{1: 2, 12: 5, 17: 3, 19: 5}
	afterBlack   = map[int]int{24: 2, 13: 5, 8: 2, 6: 4, 5: 2}
	afterWhite   = map[int]int{1: 2, 12: 5, 17: 2, 19: 4, 23: 2}
	raceBlack    = map[int]int{6: 5, 5: 5, 4: 5}
	raceWhite    = map[int]int{19: 5, 20: 5, 21: 5}
)

// similarFixture saves the target and four others; it returns their ids.
type similarFixture struct{ target, same, one, two, far int64 }

func saveSimilarFixture(t *testing.T, s storage.Storage) similarFixture {
	t.Helper()
	ctx := context.Background()
	ps := s.Positions()
	save := func(p domain.Position) int64 {
		t.Helper()
		id, err := ps.Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("Save: %v", err)
		}
		return id
	}
	var f similarFixture
	f.target = save(board(openingBlack, openingWhite))
	// The same checkers as another problem: a cube decision at a match score,
	// the other side on roll. None of that is part of a structure.
	same := board(openingWhite2Black(), openingBlack2White())
	same.DecisionType = domain.CubeAction
	same.Score = [2]int{3, 5}
	f.same = save(same)
	f.one = save(board(afterBlack, openingWhite))
	f.two = save(board(afterBlack, afterWhite))
	f.far = save(board(raceBlack, raceWhite))
	return f
}

// openingWhite2Black and openingBlack2White draw the opening with the colours
// swapped: the same structure, read from the other side.
func openingWhite2Black() map[int]int { return flip(openingWhite) }
func openingBlack2White() map[int]int { return flip(openingBlack) }

func flip(m map[int]int) map[int]int {
	out := map[int]int{}
	for pt, n := range m {
		out[25-pt] = n
	}
	return out
}

// testSimilarCountsPlays pins what a distance is: the same structure at 0
// whatever the decision and score, one play at 1, two plays at 2, nearest
// first, and nothing beyond.
func testSimilarCountsPlays(t *testing.T, s storage.Storage) {
	f := saveSimilarFixture(t, s)
	got, err := rank(context.Background(), s, f.target, 0, 50)
	if err != nil {
		t.Fatalf("Rank: %v", err)
	}
	want := []struct {
		id   int64
		dist int
	}{{f.same, 0}, {f.one, 1}, {f.two, 2}}
	if len(got) != len(want) {
		t.Fatalf("Rank returned %d neighbours, want %d: %+v", len(got), len(want), distances(got))
	}
	for i, w := range want {
		if got[i].Position.ID != w.id || got[i].Distance != w.dist {
			t.Errorf("neighbour %d = id %d at %d, want id %d at %d", i, got[i].Position.ID, got[i].Distance, w.id, w.dist)
		}
	}
	// Symmetric: the target is one play from the position after it.
	back, err := rank(context.Background(), s, f.one, 1, 50)
	if err != nil {
		t.Fatalf("Rank: %v", err)
	}
	found := false
	for _, n := range back {
		if n.Position.ID == f.target && n.Distance == 1 {
			found = true
		}
	}
	if !found {
		t.Errorf("the position before a play is one play away too: %+v", distances(back))
	}
}

// testSimilarCeilingNeverWidens pins the honest answer: a ranking returns
// what stands within its order and never completes with farther positions,
// down to an empty list.
func testSimilarCeilingNeverWidens(t *testing.T, s storage.Storage) {
	f := saveSimilarFixture(t, s)
	got, err := rank(context.Background(), s, f.target, 1, 50)
	if err != nil {
		t.Fatalf("Rank: %v", err)
	}
	if len(got) != 2 || got[0].Position.ID != f.same || got[1].Position.ID != f.one {
		t.Errorf("like<1 = %+v, want the same structure then the one-play neighbour", distances(got))
	}
	alone, err := rank(context.Background(), s, f.far, 0, 50)
	if err != nil {
		t.Fatalf("Rank: %v", err)
	}
	if len(alone) != 0 {
		t.Errorf("nothing stands within two plays of the race: got %+v", distances(alone))
	}
}

// testSimilarSpansTheTargetsMatches pins that the relation holds across the
// whole library: the plies of the target's own match are neighbours too.
func testSimilarSpansTheTargetsMatches(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	f := saveSimilarFixture(t, s)
	ms := s.Matches()
	matchID, err := ms.Save(ctx, "", &domain.Match{Player1Name: "A", Player2Name: "B", MatchLength: 7})
	if err != nil {
		t.Fatalf("Save match: %v", err)
	}
	gameID, err := ms.CreateGame(ctx, "", &domain.Game{MatchID: matchID, GameNumber: 1})
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	for i, pid := range []int64{f.target, f.one} {
		mv := domain.Move{GameID: gameID, MoveNumber: int32(i + 1), PositionID: pid, MoveType: "checker"}
		if _, err := ms.CreateMove(ctx, "", &mv); err != nil {
			t.Fatalf("CreateMove: %v", err)
		}
	}
	got, err := rank(ctx, s, f.target, 1, 50)
	if err != nil {
		t.Fatalf("Rank: %v", err)
	}
	for _, n := range got {
		if n.Position.ID == f.one {
			return
		}
	}
	t.Errorf("the next ply of the target's own match is one play away: %+v", distances(got))
}

// testSimilarRefusesAnUnservedOrder pins that an order the search cannot
// answer exactly is refused, never approximated.
func testSimilarRefusesAnUnservedOrder(t *testing.T, s storage.Storage) {
	f := saveSimilarFixture(t, s)
	_, err := rank(context.Background(), s, f.target, reach.MaxOrder+1, 50)
	if !errors.Is(err, storage.ErrInvalid) {
		t.Fatalf("like<%d: err = %v, want ErrInvalid", reach.MaxOrder+1, err)
	}
}

// testSimilarRanksAgainstADrawnBoard pins the target of a bare `like` in edit
// mode: the board on the screen, never saved, under the same relation.
func testSimilarRanksAgainstADrawnBoard(t *testing.T, s storage.Storage) {
	f := saveSimilarFixture(t, s)
	got, err := s.Search().Rank(context.Background(), "", domain.SearchFilters{
		LikeFilter:      true,
		LikeTargetBoard: board(openingBlack, openingWhite),
	}, storage.ListOpts{Limit: 10})
	if err != nil {
		t.Fatalf("Rank against a drawn board: %v", err)
	}
	// A drawing is no stored position: the stored opening is at 0 too.
	if len(got) != 4 || got[0].Distance != 0 || got[1].Distance != 0 || got[3].Position.ID != f.two {
		t.Errorf("ranking against the drawn opening = %+v", distances(got))
	}
}

// testSimilarOnlyRanks pins that the relation is served by Rank alone. Every
// unordered reader refuses `like` rather than ignoring it, for an ignored
// `like` selects the whole library; a living collection is one such reader.
func testSimilarOnlyRanks(t *testing.T, s storage.Storage) {
	fx := saveSimilarFixture(t, s)
	ctx := context.Background()
	f := domain.SearchFilters{LikeFilter: true, LikeTargetID: fx.target}
	readers := []struct {
		name string
		run  func() error
	}{
		{"Find", func() error {
			for _, err := range s.Search().Find(ctx, "", f, storage.ListOpts{}) {
				if err != nil {
					return err
				}
			}
			return nil
		}},
		{"FindIDs", func() error { _, err := s.Search().FindIDs(ctx, "", f, storage.ListOpts{}); return err }},
		{"FindPage", func() error {
			_, _, err := s.Search().FindPage(ctx, "", f, storage.PageOpts{Limit: 3})
			return err
		}},
		{"FindIDsPage", func() error {
			_, _, err := s.Search().FindIDsPage(ctx, "", f, storage.PageOpts{Limit: 3})
			return err
		}},
		{"Count", func() error { _, err := s.Search().Count(ctx, "", f); return err }},
		{"CountUpTo", func() error { _, _, err := s.Search().CountUpTo(ctx, "", f, 10); return err }},
		{"IndexOf", func() error { _, _, err := s.Search().IndexOf(ctx, "", f, fx.one); return err }},
		{"CreateCollection", func() error {
			_, err := storage.CreateCollection(ctx, s, "", "like", "", "s like")
			return err
		}},
		{"EvaluateCollection", func() error {
			// A query saved before the token was refused there.
			id, err := storage.CreateCollection(ctx, s, "", "saved like", "", "")
			if err != nil {
				return err
			}
			if err := s.Collections().SetFilterQuery(ctx, "", id, "s like"); err != nil {
				return err
			}
			_, err = storage.EvaluateCollection(ctx, s, "", id, 0)
			return err
		}},
	}
	for _, r := range readers {
		if err := r.run(); !errors.Is(err, storage.ErrInvalid) {
			t.Errorf("%s with `like`: err = %v, want ErrInvalid", r.name, err)
		}
	}
}

// rank runs a ranked query the way every caller does: the `like` token, its
// target already resolved to an id.
func rank(ctx context.Context, s storage.Storage, targetID int64, order, limit int) ([]storage.SimilarPosition, error) {
	return s.Search().Rank(ctx, "", domain.SearchFilters{
		LikeFilter:      true,
		LikeTargetID:    targetID,
		LikeMaxDistance: order,
	}, storage.ListOpts{Limit: limit})
}

func distances(ns []storage.SimilarPosition) [][2]int64 {
	out := make([][2]int64, len(ns))
	for i, n := range ns {
		out[i] = [2]int64{n.Position.ID, int64(n.Distance)}
	}
	return out
}

// board builds a checker-play position, Black on roll, from both sides'
// points in board numbering.
func board(black, white map[int]int) domain.Position {
	var p domain.Position
	for i := range p.Board.Points {
		p.Board.Points[i] = domain.Point{Checkers: 0, Color: domain.None}
	}
	for pt, n := range black {
		p.Board.Points[pt] = domain.Point{Checkers: n, Color: domain.Black}
	}
	for pt, n := range white {
		p.Board.Points[pt] = domain.Point{Checkers: n, Color: domain.White}
	}
	p.PlayerOnRoll = domain.Black
	p.Dice = [2]int{3, 1}
	p.Score = [2]int{-1, -1}
	return p
}

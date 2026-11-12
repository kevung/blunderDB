package storagetest

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

var searchSorts = []string{"", "error", "winrate", "close", "pip_asc", "pip_desc", "date_asc", "date_desc"}

// cursorPos is a position whose pip count takes one of three values, so that every
// sort key both ties and differs across the fixture.
func cursorPos(n int) domain.Position {
	p := domain.InitializePosition()
	for i := range p.Board.Points {
		p.Board.Points[i] = domain.Point{Color: -1}
	}
	p.Board.Points[1+n%3] = domain.Point{Checkers: 2, Color: domain.Black}
	p.Board.Points[24] = domain.Point{Checkers: 1, Color: domain.White}
	p.Board.Bearoff = [2]int{13, 14}
	p.PlayerOnRoll = domain.Black
	p.DecisionType = domain.CheckerAction
	p.Dice = [2]int{3, 1}
	p.Score = [2]int{n % 50, n / 50} // the hash clamps a score at 63
	return p
}

// seedCursorFixture saves positions first..last: two thirds analysed (error and win rate
// tie in pairs), the odd ones commented "keep", one in two reached by a dated match
// (dates tie in pairs). It returns the ids, in creation order.
func seedCursorFixture(t *testing.T, s storage.Storage, scope string, first, last int) []int64 {
	t.Helper()
	ctx := context.Background()
	var ids []int64
	for n := first; n <= last; n++ {
		p := cursorPos(n)
		id, err := s.Positions().Save(ctx, scope, &p)
		if err != nil {
			t.Fatalf("Save %d: %v", n, err)
		}
		if slices.Contains(ids, id) {
			t.Fatalf("position %d saved over an earlier one (%d)", n, id)
		}
		ids = append(ids, id)
		if n%3 != 0 {
			played := "13/11 24/23"
			equityError := float64(n%4) * 0.02
			a := domain.PositionAnalysis{
				PlayedMoves: []string{played},
				CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
					{Move: "8/6 6/4", Equity: 0.5, PlayerWinChance: float64(n%2) * 10},
					{Move: played, Equity: 0.5 - equityError, EquityError: &equityError},
				}},
			}
			if err := s.Analyses().Save(ctx, scope, id, &a); err != nil {
				t.Fatalf("Save analysis %d: %v", n, err)
			}
		}
		if n%2 == 1 {
			if _, err := s.Comments().Add(ctx, scope, id, "keep this one"); err != nil {
				t.Fatalf("Add comment on %d: %v", id, err)
			}
		}
		if n%2 == 0 {
			m := domain.Match{Player1Name: "p", Player2Name: "q", MatchLength: 7,
				MatchDate: time.Date(2020+n%3, 1, 1, 0, 0, 0, 0, time.UTC)}
			matchID, err := s.Matches().Save(ctx, scope, &m)
			if err != nil {
				t.Fatalf("Save match: %v", err)
			}
			gameID, err := s.Matches().CreateGame(ctx, scope, &domain.Game{MatchID: matchID, GameNumber: 1})
			if err != nil {
				t.Fatalf("CreateGame: %v", err)
			}
			mv := domain.Move{GameID: gameID, MoveNumber: 1, MoveType: "checker",
				PositionID: id, Player: 1, Dice: [2]int32{3, 1}, CheckerMove: "8/5 6/5"}
			if _, err := s.Matches().CreateMove(ctx, scope, &mv); err != nil {
				t.Fatalf("CreateMove: %v", err)
			}
		}
	}
	return ids
}

// pageThrough reads the whole result of f by cursor, limit rows at a time; between
// two pages it calls between (nil for none) with the page count so far.
func pageThrough(t *testing.T, s storage.Storage, scope string, f domain.SearchFilters, limit int, between func(page int)) []int64 {
	t.Helper()
	ctx := context.Background()
	var got []int64
	cursor := ""
	for page := 0; ; page++ {
		if page > 100 {
			t.Fatalf("cursor never ends: %v", got)
		}
		ids, next, err := s.Search().FindIDsPage(ctx, scope, f, storage.PageOpts{Limit: limit, Cursor: cursor})
		if err != nil {
			t.Fatalf("FindIDsPage(page %d): %v", page, err)
		}
		positions, nextP, err := s.Search().FindPage(ctx, scope, f, storage.PageOpts{Limit: limit, Cursor: cursor})
		if err != nil {
			t.Fatalf("FindPage(page %d): %v", page, err)
		}
		var posIDs []int64
		for _, p := range positions {
			posIDs = append(posIDs, p.ID)
		}
		if !slices.Equal(posIDs, ids) || next != nextP {
			t.Fatalf("page %d: FindPage %v (next %q) and FindIDsPage %v (next %q) disagree", page, posIDs, nextP, ids, next)
		}
		if next != "" && len(ids) != limit {
			t.Fatalf("page %d: %v is short yet announces a next page", page, ids)
		}
		got = append(got, ids...)
		if next == "" {
			return got
		}
		cursor = next
		if between != nil {
			between(page)
		}
	}
}

// testSearchCursorWalksTheWholeResult: a result read by cursor, two rows at a time, is
// the result read at once, in every sort, with and without a Go phase, with the
// Go-filtered scan in chunks of two rows; the last page announces no next one.
func testSearchCursorWalksTheWholeResult(t *testing.T, s storage.Storage) {
	seedCursorFixture(t, s, "", 1, 12)
	for _, sort := range searchSorts {
		for name, f := range map[string]domain.SearchFilters{
			"sql only":   {Sort: sort},
			"go phase":   {SearchText: "keep", Sort: sort},
			"move error": {MoveErrorFilter: "E>10", Sort: sort},
		} {
			t.Run(name+"/"+sort, func(t *testing.T) {
				want := searchIDs(t, s, f)
				if len(want) < 3 {
					t.Fatalf("fixture too small: %v", want)
				}
				if got := pageThrough(t, s, "", f, 2, nil); !slices.Equal(got, want) {
					t.Errorf("by cursor: got %v, want %v", got, want)
				}
				defer sqlshared.SetSearchChunk(2)()
				if got := pageThrough(t, s, "", f, 3, nil); !slices.Equal(got, want) {
					t.Errorf("by cursor, scan in chunks: got %v, want %v", got, want)
				}
			})
		}
	}
}

// testSearchCursorSurvivesInserts: positions saved while a result is read by cursor
// neither repeat nor hide a row of it. Every row present before the first page is
// read exactly once, in its order; a new row is read at most once.
func testSearchCursorSurvivesInserts(t *testing.T, s storage.Storage) {
	seedCursorFixture(t, s, "", 1, 12)
	next := 13
	for _, sort := range searchSorts {
		t.Run(sort, func(t *testing.T) {
			f := domain.SearchFilters{Sort: sort}
			before := searchIDs(t, s, f)
			got := pageThrough(t, s, "", f, 3, func(page int) {
				if page >= 2 { // rows saved on every page could outrun the reader
					return
				}
				seedCursorFixture(t, s, "", next, next+1)
				next += 2
			})
			seen := map[int64]bool{}
			var old []int64
			was := map[int64]bool{}
			for _, id := range before {
				was[id] = true
			}
			for _, id := range got {
				if seen[id] {
					t.Fatalf("row %d read twice: %v", id, got)
				}
				seen[id] = true
				if was[id] {
					old = append(old, id)
				}
			}
			if !slices.Equal(old, before) {
				t.Errorf("rows present before the first page: read %v, want %v", old, before)
			}
		})
	}
}

// testSearchCursorRefusesWhatItDoesNotKnow: a cursor is only good for the search that
// issued it. Garbage, another sort, other filters, or a cursor joined to an offset are
// refused as invalid rather than read as some other page.
func testSearchCursorRefusesWhatItDoesNotKnow(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	seedCursorFixture(t, s, "", 1, 6)
	_, next, err := s.Search().FindIDsPage(ctx, "", domain.SearchFilters{Sort: "pip_asc"}, storage.PageOpts{Limit: 2})
	if err != nil || next == "" {
		t.Fatalf("first page: next %q, %v", next, err)
	}
	for name, tc := range map[string]struct {
		f    domain.SearchFilters
		opts storage.PageOpts
	}{
		"garbage":      {domain.SearchFilters{Sort: "pip_asc"}, storage.PageOpts{Limit: 2, Cursor: "not-a-cursor"}},
		"other sort":   {domain.SearchFilters{Sort: "pip_desc"}, storage.PageOpts{Limit: 2, Cursor: next}},
		"other filter": {domain.SearchFilters{Sort: "pip_asc", SearchText: "keep"}, storage.PageOpts{Limit: 2, Cursor: next}},
		"with offset":  {domain.SearchFilters{Sort: "pip_asc"}, storage.PageOpts{Limit: 2, Cursor: next, Offset: 2}},
		"no limit":     {domain.SearchFilters{Sort: "pip_asc"}, storage.PageOpts{Cursor: next}},
	} {
		if ids, _, err := s.Search().FindIDsPage(ctx, "", tc.f, tc.opts); !errors.Is(err, storage.ErrInvalid) {
			t.Errorf("%s: got %v, %v; want ErrInvalid", name, ids, err)
		}
	}
	// An offset alone still cuts the first page, for the clients that count rows.
	ids, _, err := s.Search().FindIDsPage(ctx, "", domain.SearchFilters{Sort: "pip_asc"}, storage.PageOpts{Limit: 2, Offset: 2})
	if want := searchIDs(t, s, domain.SearchFilters{Sort: "pip_asc"})[2:4]; err != nil || !slices.Equal(ids, want) {
		t.Errorf("offset 2: got %v, %v; want %v", ids, err, want)
	}
}

// checkSearchCursorIsolation: a cursor issued in a never reads b's rows: either b
// refuses it, or what b answers is b's own.
func checkSearchCursorIsolation(t *testing.T, ctx context.Context, s storage.Storage, a, b string) {
	seedCursorFixture(t, s, a, 1, 6)
	own := map[int64]bool{}
	for _, id := range seedCursorFixture(t, s, b, 7, 12) {
		own[id] = true
	}
	for _, sort := range searchSorts {
		f := domain.SearchFilters{Sort: sort}
		_, next, err := s.Search().FindIDsPage(ctx, a, f, storage.PageOpts{Limit: 2})
		if err != nil || next == "" {
			t.Fatalf("FindIDsPage(%s, %q): next %q, %v", a, sort, next, err)
		}
		ids, _, err := s.Search().FindIDsPage(ctx, b, f, storage.PageOpts{Limit: 50, Cursor: next})
		if err != nil && !errors.Is(err, storage.ErrInvalid) {
			t.Errorf("FindIDsPage(%s, cursor of %s, %q): %v", b, a, sort, err)
		}
		for _, id := range ids {
			if !own[id] {
				t.Errorf("FindIDsPage(%s, cursor of %s, %q): read %d, not %s's", b, a, sort, id, b)
			}
		}
	}
}

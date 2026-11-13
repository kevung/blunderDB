package storagetest

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
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
		checkGraftedAnchorStaysHome(t, ctx, s, a, b, f, next, own)
	}
}

// checkGraftedAnchorStaysHome: a's anchor grafted onto a cursor b was issued for the
// same search passes the fingerprint; b then reads past a's row only its own rows, and
// the cursor it hands back anchors on one of them — never on a's row or a's key.
func checkGraftedAnchorStaysHome(t *testing.T, ctx context.Context, s storage.Storage, a, b string, f domain.SearchFilters, cursorOfA string, own map[int64]bool) {
	t.Helper()
	anchors := map[string]bool{} // every anchor b issues, one row a page
	var first string
	for cursor, page := "", 0; page == 0 || cursor != ""; page++ {
		_, next, err := s.Search().FindIDsPage(ctx, b, f, storage.PageOpts{Limit: 1, Cursor: cursor})
		if err != nil {
			t.Fatalf("FindIDsPage(%s, %q, page %d): %v", b, f.Sort, page, err)
		}
		if page == 0 {
			first = next
		}
		if next != "" {
			anchors[anchorOf(t, next)] = true
		}
		cursor = next
	}
	stolen := map[string]any{}
	for k, v := range cursorFields(t, cursorOfA) {
		if slices.Contains(anchorFields, k) {
			stolen[k] = v
		}
	}
	grafted := withAnchor(t, first, stolen)
	ids, next, err := s.Search().FindIDsPage(ctx, b, f, storage.PageOpts{Limit: 2, Cursor: grafted})
	if err != nil && !errors.Is(err, storage.ErrInvalid) {
		t.Errorf("FindIDsPage(%s, anchor of %s, %q): %v", b, a, f.Sort, err)
	}
	for _, id := range ids {
		if !own[id] {
			t.Errorf("FindIDsPage(%s, anchor of %s, %q): read %d, not %s's", b, a, f.Sort, id, b)
		}
	}
	if next != "" && !anchors[anchorOf(t, next)] {
		t.Errorf("FindIDsPage(%s, anchor of %s, %q): next cursor anchors on %s, not on a row of %s", b, a, f.Sort, anchorOf(t, next), b)
	}
}

func anchorOf(t *testing.T, cursor string) string {
	m := cursorFields(t, cursor)
	anchor := map[string]any{}
	for _, k := range anchorFields {
		if v, ok := m[k]; ok {
			anchor[k] = v
		}
	}
	blob, _ := json.Marshal(anchor)
	return string(blob)
}

// anchorFields are the parts of a wire cursor that say where the page resumes; the
// rest ties it to its search.
var anchorFields = []string{"id", "n", "i", "t", "b"}

func cursorFields(t *testing.T, cursor string) map[string]any {
	t.Helper()
	blob, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		t.Fatalf("cursor %q: %v", cursor, err)
	}
	dec := json.NewDecoder(bytes.NewReader(blob))
	dec.UseNumber()
	m := map[string]any{}
	if err := dec.Decode(&m); err != nil {
		t.Fatalf("cursor %q: %v", cursor, err)
	}
	return m
}

func cursorOf(m map[string]any) string {
	blob, _ := json.Marshal(m)
	return base64.RawURLEncoding.EncodeToString(blob)
}

// withAnchor is cursor with its anchor replaced by the fields of anchor.
func withAnchor(t *testing.T, cursor string, anchor map[string]any) string {
	m := cursorFields(t, cursor)
	for _, k := range anchorFields {
		delete(m, k)
	}
	for k, v := range anchor {
		m[k] = v
	}
	return cursorOf(m)
}

// outOfColumnKeys are integer keys no row of the sort column can hold: pip_1 and
// player1_win_rate are smallint, best_move_equity_error integer in PostgreSQL. Bound
// as a parameter, such a key must not fail the statement.
var outOfColumnKeys = func() map[string][]string {
	int4 := []string{"2147483648", "-2147483649", "-9223372036854775808", "9223372036854775807"}
	int2 := append([]string{"32768", "70000", "-32769"}, int4...)
	return map[string][]string{"error": int4, "winrate": int2, "pip_asc": int2, "pip_desc": int2}
}()

// testSearchCursorRefusesAForgedAnchor: the anchor of a cursor is the client's to
// rewrite, the fingerprint of the search left as it was. Whatever it rewrites — a key
// of another type than the sort column's, an id out of range, a shape the server never
// issues — the page is refused as invalid, never failed as an internal error.
func testSearchCursorRefusesAForgedAnchor(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	seedCursorFixture(t, s, "", 1, 6)
	keys := map[string]any{"i": json.Number("3"), "t": json.Number("1600000000000000000"), "b": true}
	for _, sort := range searchSorts {
		f := domain.SearchFilters{Sort: sort}
		_, next, err := s.Search().FindIDsPage(ctx, "", f, storage.PageOpts{Limit: 2})
		if err != nil || next == "" {
			t.Fatalf("%q first page: next %q, %v", sort, next, err)
		}
		genuine := cursorFields(t, next)
		id := genuine["id"]
		forged := map[string]map[string]any{
			"id zero":       {"id": 0},
			"id negative":   {"id": -5},
			"null and key":  {"id": id, "n": true, "i": json.Number("3")},
			"two keys":      {"id": id, "i": json.Number("3"), "b": true},
			"key as string": {"id": id, "i": "3"},
		}
		if sort == "" {
			forged["null on the id order"] = map[string]any{"id": id, "n": true}
		}
		if _, issued := genuine["i"]; issued {
			for _, n := range outOfColumnKeys[sort] {
				cursor := withAnchor(t, next, map[string]any{"id": id, "i": json.Number(n)})
				if _, _, err := s.Search().FindIDsPage(ctx, "", f, storage.PageOpts{Limit: 2, Cursor: cursor}); err != nil && !errors.Is(err, storage.ErrInvalid) {
					t.Errorf("%q, key %s outside its column: %v; want a page or ErrInvalid", sort, n, err)
				}
			}
		}
		for kind, v := range keys {
			if _, issued := genuine[kind]; !issued {
				forged["key of kind "+kind] = map[string]any{"id": id, kind: v}
			}
		}
		for name, anchor := range forged {
			cursor := withAnchor(t, next, anchor)
			if ids, _, err := s.Search().FindIDsPage(ctx, "", f, storage.PageOpts{Limit: 2, Cursor: cursor}); !errors.Is(err, storage.ErrInvalid) {
				t.Errorf("%q, %s: got %v, %v; want ErrInvalid", sort, name, ids, err)
			}
			if _, _, err := s.Search().FindPage(ctx, "", f, storage.PageOpts{Limit: 2, Cursor: cursor}); !errors.Is(err, storage.ErrInvalid) {
				t.Errorf("%q, %s: FindPage got %v; want ErrInvalid", sort, name, err)
			}
		}
		extra := cursorFields(t, next)
		extra["x"] = 1
		if _, _, err := s.Search().FindIDsPage(ctx, "", f, storage.PageOpts{Limit: 2, Cursor: cursorOf(extra)}); !errors.Is(err, storage.ErrInvalid) {
			t.Errorf("%q, unknown field: got %v; want ErrInvalid", sort, err)
		}
		if _, _, err := s.Search().FindIDsPage(ctx, "", f, storage.PageOpts{Limit: 2, Cursor: cursorOf(genuine)}); err != nil {
			t.Errorf("%q, the genuine cursor rewritten as is: %v", sort, err)
		}
	}
}

// testSearchCursorKeyComesWithItsRow: the last row of a page deleted before the page
// is answered still anchors the next cursor at its own key — the key is read with the
// row, not looked up afterwards — so no row with a key is skipped.
func testSearchCursorKeyComesWithItsRow(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	seedCursorFixture(t, s, "", 1, 12)
	for _, sort := range searchSorts {
		t.Run(sort, func(t *testing.T) {
			f := domain.SearchFilters{Sort: sort}
			want := searchIDs(t, s, f)
			var deleted []int64
			restore := sqlshared.SetPageReadHook(func(lastID int64) {
				if len(deleted) > 0 {
					return
				}
				deleted = append(deleted, lastID)
				if err := s.Positions().Delete(ctx, "", lastID); err != nil {
					t.Fatalf("Delete %d: %v", lastID, err)
				}
			})
			ids, next, err := s.Search().FindIDsPage(ctx, "", f, storage.PageOpts{Limit: 3})
			restore()
			if err != nil || next == "" || len(deleted) != 1 {
				t.Fatalf("first page: %v, next %q, %v, deleted %v", ids, next, err, deleted)
			}
			var got []int64
			for cursor := next; cursor != ""; {
				page, n, err := s.Search().FindIDsPage(ctx, "", f, storage.PageOpts{Limit: 3, Cursor: cursor})
				if err != nil {
					t.Fatalf("FindIDsPage: %v", err)
				}
				got, cursor = append(got, page...), n
			}
			if wantRest := want[3:]; !slices.Equal(got, wantRest) {
				t.Errorf("after the deleted anchor %d: got %v, want %v", deleted[0], got, wantRest)
			}
		})
	}
}

// testMoveErrorProbeKeepsTheResult: the probe that reads a rare move-error condition
// first, cut to a few ids per branch of its union, answers the result the condition
// answers unread, wherever the cut falls in the checker-play branch (the only branch
// this fixture fills).
func testMoveErrorProbeKeepsTheResult(t *testing.T, s storage.Storage) {
	seedCursorFixture(t, s, "", 1, 12)
	for _, sort := range []string{"", "error", "pip_asc"} {
		f := domain.SearchFilters{MoveErrorFilter: "E>10", Sort: sort}
		restore := sqlshared.SetNarrowLimit(-1)
		want := searchIDs(t, s, f)
		restore()
		if len(want) < 3 {
			t.Fatalf("fixture too small: %v", want)
		}
		for n := 0; n <= len(want)+1; n++ {
			restore := sqlshared.SetNarrowLimit(n)
			got := searchIDs(t, s, f)
			paged := pageThrough(t, s, "", f, 2, nil)
			restore()
			if !slices.Equal(got, want) || !slices.Equal(paged, want) {
				t.Errorf("%q, probe cut at %d: got %v, by cursor %v; want %v", sort, n, got, paged, want)
			}
		}
	}
}

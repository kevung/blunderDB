// Contract cases for saved filters, command and search history, sessions,
// metadata, transactions and scope isolation.
// The table that runs them lives in contract.go.
package storagetest

import (
	"context"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func testFilterSaveAndList(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	fs := s.Filters()

	id1, err := fs.Save(ctx, "", "f1", "cmd1")
	if err != nil {
		t.Fatalf("Save f1: %v", err)
	}
	if _, err := fs.Save(ctx, "", "f1", "dup"); !errors.Is(err, storage.ErrConflict) {
		t.Fatalf("duplicate name: got %v, want ErrConflict", err)
	}
	id2, err := fs.Save(ctx, "", "f2", "cmd2")
	if err != nil {
		t.Fatalf("Save f2: %v", err)
	}

	var names []string
	for f, err := range fs.List(ctx, "") {
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		names = append(names, f.Name)
	}
	if len(names) != 2 || names[0] != "f1" || names[1] != "f2" {
		t.Fatalf("List order: %v, want [f1 f2]", names)
	}

	if err := fs.Update(ctx, "", id1, "f1b", "cmd1b"); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if err := fs.SaveEditPosition(ctx, "", "f1b", "editX"); err != nil {
		t.Fatalf("SaveEditPosition: %v", err)
	}
	if got, err := fs.LoadEditPosition(ctx, "", "f1b"); err != nil || got != "editX" {
		t.Fatalf("LoadEditPosition: got %q err %v, want editX", got, err)
	}
	if got, err := fs.LoadEditPosition(ctx, "", "unknown"); err != nil || got != "" {
		t.Fatalf("LoadEditPosition(unknown): got %q err %v, want empty", got, err)
	}
	// The "Sauf" structure lives beside the edit position and is independent
	// of it: a filter carries none until one is stored, and an unknown filter
	// reports ErrNotFound rather than silently creating a row.
	if got, err := fs.LoadExcludePosition(ctx, "", "f1b"); err != nil || got != "" {
		t.Fatalf("LoadExcludePosition(before save): got %q err %v, want empty", got, err)
	}
	if err := fs.SaveExcludePosition(ctx, "", "f1b", "exclX"); err != nil {
		t.Fatalf("SaveExcludePosition: %v", err)
	}
	if got, err := fs.LoadExcludePosition(ctx, "", "f1b"); err != nil || got != "exclX" {
		t.Fatalf("LoadExcludePosition: got %q err %v, want exclX", got, err)
	}
	if got, err := fs.LoadEditPosition(ctx, "", "f1b"); err != nil || got != "editX" {
		t.Fatalf("LoadEditPosition after SaveExcludePosition: got %q err %v, want editX", got, err)
	}
	if err := fs.SaveExcludePosition(ctx, "", "unknown", "x"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("SaveExcludePosition(unknown): got %v, want ErrNotFound", err)
	}
	if got, err := fs.LoadExcludePosition(ctx, "", "unknown"); err != nil || got != "" {
		t.Fatalf("LoadExcludePosition(unknown): got %q err %v, want empty", got, err)
	}

	if err := fs.Delete(ctx, "", id2); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err := fs.Delete(ctx, "", id2); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("Delete absent: got %v, want ErrNotFound", err)
	}
	n := 0
	for _, err := range fs.List(ctx, "") {
		if err != nil {
			t.Fatalf("List after delete: %v", err)
		}
		n++
	}
	if n != 1 {
		t.Fatalf("filters after delete: got %d, want 1", n)
	}
}

func testCommandHistory(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	h := s.History()

	if err := h.Save(ctx, "", "first"); err != nil {
		t.Fatalf("Save first: %v", err)
	}
	if err := h.Save(ctx, "", "second"); err != nil {
		t.Fatalf("Save second: %v", err)
	}
	got, err := h.Load(ctx, "")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got) != 2 || got[0] != "first" || got[1] != "second" {
		t.Fatalf("Load order: %v, want [first second]", got)
	}

	if err := h.Clear(ctx, ""); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	got, err = h.Load(ctx, "")
	if err != nil {
		t.Fatalf("Load after clear: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("after clear: %v, want empty", got)
	}
}

func testSearchHistory(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	sh := s.SearchHistory()

	if err := sh.Save(ctx, "", "cmd1", "pos1", ""); err != nil {
		t.Fatalf("Save 1: %v", err)
	}
	if err := sh.Save(ctx, "", "cmd2", "pos2", "excl2"); err != nil {
		t.Fatalf("Save 2: %v", err)
	}

	var entries []storage.SearchHistory
	for e, err := range sh.List(ctx, "") {
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		entries = append(entries, *e)
	}
	if len(entries) != 2 {
		t.Fatalf("List count: got %d, want 2", len(entries))
	}
	// Most recent first; the id-DESC tiebreak makes this deterministic even
	// when both rows share a millisecond timestamp.
	if entries[0].Command != "cmd2" {
		t.Fatalf("List order: first = %q, want cmd2", entries[0].Command)
	}
	if entries[0].ExcludePosition != "excl2" || entries[1].ExcludePosition != "" {
		t.Fatalf("List exclude positions: got %q / %q, want excl2 / empty",
			entries[0].ExcludePosition, entries[1].ExcludePosition)
	}

	// Delete by timestamp (covers the same-millisecond case: deleting a
	// timestamp removes every entry that carries it).
	for _, e := range entries {
		if err := sh.DeleteEntry(ctx, "", e.Timestamp); err != nil {
			t.Fatalf("DeleteEntry: %v", err)
		}
	}
	n := 0
	for _, err := range sh.List(ctx, "") {
		if err != nil {
			t.Fatalf("List after delete: %v", err)
		}
		n++
	}
	if n != 0 {
		t.Fatalf("search history after delete: got %d, want 0", n)
	}
}

func testSessionSaveLoad(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ss := s.Session()

	// A scope that never stored a session loads an empty (non-nil) state.
	empty, err := ss.Load(ctx, "")
	if err != nil {
		t.Fatalf("Load empty: %v", err)
	}
	if empty == nil || empty.LastSearchCommand != "" || empty.HasActiveSearch || len(empty.LastPositionIDs) != 0 {
		t.Fatalf("fresh session not empty: %+v", empty)
	}

	want := storage.SessionState{
		LastSearchCommand:  "decision_type checker",
		LastSearchPosition: "xgid",
		LastPositionIndex:  5,
		LastPositionIDs:    []int64{1, 2, 3},
		HasActiveSearch:    true,
		ViewsJSON:          `{"a":1}`,
	}
	if err := ss.Save(ctx, "", want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := ss.Load(ctx, "")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.LastSearchCommand != want.LastSearchCommand || got.LastSearchPosition != want.LastSearchPosition ||
		got.LastPositionIndex != want.LastPositionIndex || got.HasActiveSearch != want.HasActiveSearch ||
		got.ViewsJSON != want.ViewsJSON || len(got.LastPositionIDs) != 3 {
		t.Fatalf("Load round-trip:\n got %+v\nwant %+v", got, want)
	}
}

func testSessionMultiScope(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ss := s.Session()

	if err := ss.Save(ctx, "1", storage.SessionState{LastSearchCommand: "alpha", LastPositionIndex: 1}); err != nil {
		t.Fatalf("Save scope 1: %v", err)
	}
	if err := ss.Save(ctx, "2", storage.SessionState{LastSearchCommand: "beta", LastPositionIndex: 2}); err != nil {
		t.Fatalf("Save scope 2: %v", err)
	}

	s1, _ := ss.Load(ctx, "1")
	s2, _ := ss.Load(ctx, "2")
	if s1.LastSearchCommand != "alpha" || s1.LastPositionIndex != 1 {
		t.Fatalf("scope 1 leaked: %+v", s1)
	}
	if s2.LastSearchCommand != "beta" || s2.LastPositionIndex != 2 {
		t.Fatalf("scope 2 leaked: %+v", s2)
	}
	// The empty scope is independent of both.
	if e, _ := ss.Load(ctx, ""); e.LastSearchCommand != "" {
		t.Fatalf("empty scope sees other tenants: %+v", e)
	}

	// Clearing one scope leaves the other intact.
	if err := ss.Clear(ctx, "1"); err != nil {
		t.Fatalf("Clear scope 1: %v", err)
	}
	if e, _ := ss.Load(ctx, "1"); e.LastSearchCommand != "" {
		t.Fatalf("scope 1 not cleared: %+v", e)
	}
	if s2, _ := ss.Load(ctx, "2"); s2.LastSearchCommand != "beta" {
		t.Fatalf("scope 2 affected by clearing scope 1: %+v", s2)
	}
}

func testMetadataCounts(t *testing.T, s storage.Storage) {
	ctx := context.Background()

	p := checkerPos()
	posID, err := s.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}
	a := domain.PositionAnalysis{AnalysisType: "CheckerMove"}
	if err := s.Analyses().Save(ctx, "", posID, &a); err != nil {
		t.Fatalf("Save analysis: %v", err)
	}
	m := domain.Match{Player1Name: "A", Player2Name: "B"}
	matchID, err := s.Matches().Save(ctx, "", &m)
	if err != nil {
		t.Fatalf("Save match: %v", err)
	}
	g := domain.Game{MatchID: matchID, GameNumber: 1}
	gameID, err := s.Matches().CreateGame(ctx, "", &g)
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	mv := domain.Move{GameID: gameID, MoveNumber: 1, MoveType: "checker", PositionID: posID, Player: 1}
	if _, err := s.Matches().CreateMove(ctx, "", &mv); err != nil {
		t.Fatalf("CreateMove: %v", err)
	}

	// A second position carrying the individually_imported provenance flag,
	// enrolled in a study deck: exercises the IndividualPositions and
	// AnkiCards counters.
	ind := cubePos()
	ind.IndividuallyImported = true
	indID, err := s.Positions().Save(ctx, "", &ind)
	if err != nil {
		t.Fatalf("Save individual position: %v", err)
	}
	deckID, err := s.Anki().CreateDeck(ctx, "", "counts-deck", "", domain.AnkiSourceSearch, 0, "")
	if err != nil {
		t.Fatalf("CreateDeck: %v", err)
	}
	if err := s.Anki().SyncWithPositions(ctx, "", deckID, []int64{indID}); err != nil {
		t.Fatalf("SyncWithPositions: %v", err)
	}

	c, err := s.Metadata().Counts(ctx, "")
	if err != nil {
		t.Fatalf("Counts: %v", err)
	}
	want := storage.Counts{
		Positions: 2, Analyses: 1, Matches: 1, Games: 1, Moves: 1,
		IndividualPositions: 1, AnkiCards: 1,
	}
	if c != want {
		t.Fatalf("Counts = %+v, want %+v", c, want)
	}
}

func testTxRollbackUndoes(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	tx, err := s.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	p := checkerPos()
	id, err := tx.Positions().Save(ctx, "", &p)
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("tx Save: %v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatalf("Rollback: %v", err)
	}

	if _, err := s.Positions().Load(ctx, "", id); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("after rollback expected ErrNotFound, got %v", err)
	}
}

func testTxCommitPersists(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	tx, err := s.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	p := checkerPos()
	id, err := tx.Positions().Save(ctx, "", &p)
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("tx Save: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	got, err := s.Positions().Load(ctx, "", id)
	if err != nil {
		t.Fatalf("after commit Load: %v", err)
	}
	if got.ID != id {
		t.Errorf("loaded id: got %d, want %d", got.ID, id)
	}
}

// testScopeIsolation checks that command history, search history and saved
// filters are isolated per scope. PostgreSQL scopes them by tenant_id; SQLite
// scopes them by a `scope` column (added in schema 2.9.0). The same filter name
// may coexist in distinct scopes.
func testScopeIsolation(t *testing.T, s storage.Storage) {
	ctx := context.Background()

	// Command history.
	if err := s.History().Save(ctx, "1", "cmd-a"); err != nil {
		t.Fatalf("History.Save scope 1: %v", err)
	}
	if err := s.History().Save(ctx, "2", "cmd-b"); err != nil {
		t.Fatalf("History.Save scope 2: %v", err)
	}
	if got, _ := s.History().Load(ctx, "1"); len(got) != 1 || got[0] != "cmd-a" {
		t.Errorf("History scope 1: got %v, want [cmd-a]", got)
	}
	if got, _ := s.History().Load(ctx, "2"); len(got) != 1 || got[0] != "cmd-b" {
		t.Errorf("History scope 2: got %v, want [cmd-b]", got)
	}

	// Search history.
	if err := s.SearchHistory().Save(ctx, "1", "search-a", "pos-a", ""); err != nil {
		t.Fatalf("SearchHistory.Save scope 1: %v", err)
	}
	if err := s.SearchHistory().Save(ctx, "2", "search-b", "pos-b", ""); err != nil {
		t.Fatalf("SearchHistory.Save scope 2: %v", err)
	}
	if got := drainSearch(t, s, "1"); len(got) != 1 || got[0] != "search-a" {
		t.Errorf("SearchHistory scope 1: got %v, want [search-a]", got)
	}
	if got := drainSearch(t, s, "2"); len(got) != 1 || got[0] != "search-b" {
		t.Errorf("SearchHistory scope 2: got %v, want [search-b]", got)
	}

	// Filters: scope-isolated, and the same name may live in two scopes.
	if _, err := s.Filters().Save(ctx, "1", "f", "cmd1"); err != nil {
		t.Fatalf("Filters.Save scope 1: %v", err)
	}
	if _, err := s.Filters().Save(ctx, "2", "f", "cmd2"); err != nil {
		t.Errorf("same filter name in a different scope should be allowed: %v", err)
	}
	if got := drainFilters(t, s, "1"); len(got) != 1 || got[0] != "cmd1" {
		t.Errorf("Filters scope 1: got %v, want [cmd1]", got)
	}
	if got := drainFilters(t, s, "2"); len(got) != 1 || got[0] != "cmd2" {
		t.Errorf("Filters scope 2: got %v, want [cmd2]", got)
	}
}

func drainSearch(t *testing.T, s storage.Storage, scope string) []string {
	t.Helper()
	var out []string
	for e, err := range s.SearchHistory().List(context.Background(), scope) {
		if err != nil {
			t.Fatalf("SearchHistory.List: %v", err)
		}
		out = append(out, e.Command)
	}
	return out
}

func drainFilters(t *testing.T, s storage.Storage, scope string) []string {
	t.Helper()
	var out []string
	for f, err := range s.Filters().List(context.Background(), scope) {
		if err != nil {
			t.Fatalf("Filters.List: %v", err)
		}
		out = append(out, f.Command)
	}
	return out
}

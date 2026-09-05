package ingest

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// TestExportParity_GUIAndServerEntryPointsAgree covers the "same selection
// -> same tables/rows" invariant: ExportDatabase/ExportCollections/
// ExportTournaments (the GUI and CLI) call ExportSQLite directly, while the
// daemon's exports.sqlite calls it through SQLiteExporter.Export. Both must
// write the same content for the same source and the same Selection — a
// regression here would mean the two modes silently diverged after this
// package was unified into one exporter.
//
// This is deliberately not a byte-for-byte file comparison (SQLite page
// layout is not guaranteed stable across writers) but a comparison of the
// decoded content of every family an export carries.
func TestExportParity_GUIAndServerEntryPointsAgree(t *testing.T) {
	ctx := context.Background()
	src := buildParityFixture(t, ctx)

	opts := WholeTenant(FormatSQLite)

	// The GUI/CLI path: ExportDatabase and friends call ExportSQLite directly.
	guiPath := filepath.Join(t.TempDir(), "gui.sqlite")
	if _, err := ExportSQLite(ctx, src, "", guiPath, opts); err != nil {
		t.Fatalf("ExportSQLite (GUI/CLI path): %v", err)
	}

	// The server path: handleExportSQLite calls SQLiteExporter.Export.
	var buf bytes.Buffer
	if err := (SQLiteExporter{S: src}).Export(ctx, "", &buf, opts); err != nil {
		t.Fatalf("SQLiteExporter.Export (server path): %v", err)
	}
	serverPath := filepath.Join(t.TempDir(), "server.sqlite")
	if err := os.WriteFile(serverPath, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}

	guiDB, err := sqlite.Open(ctx, guiPath, nil)
	if err != nil {
		t.Fatalf("reopen GUI export: %v", err)
	}
	defer guiDB.Close()
	serverDB, err := sqlite.Open(ctx, serverPath, nil)
	if err != nil {
		t.Fatalf("reopen server export: %v", err)
	}
	defer serverDB.Close()

	assertSameExportedContent(t, ctx, guiDB, serverDB)
}

// buildParityFixture populates src with one of every family an export
// carries: a match (games, moves, move analyses, from a real GnuBG fixture),
// a collection, a tournament holding the match, and a saved filter.
func buildParityFixture(t *testing.T, ctx context.Context) *sqlite.Storage {
	t.Helper()
	st, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })

	sum, err := (GnuBGImporter{S: st}).Import(ctx, "", Source{Format: FormatGnuBG, Path: "../../../testdata/test.sgf"}, nil)
	if err != nil {
		t.Fatalf("import fixture: %v", err)
	}
	if sum.MatchID == 0 {
		t.Fatal("fixture import produced no match")
	}

	var firstPos int64
	for p, err := range st.Positions().List(ctx, "", storage.ListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		firstPos = p.ID
		break
	}
	if firstPos == 0 {
		t.Fatal("fixture import produced no positions")
	}

	cid, err := st.Collections().Create(ctx, "", "Openings", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Collections().AddPosition(ctx, "", cid, firstPos); err != nil {
		t.Fatal(err)
	}

	tid, err := st.Tournaments().Create(ctx, "", "Marseille", "2026-01-01", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Tournaments().AddMatch(ctx, "", tid, sum.MatchID); err != nil {
		t.Fatal(err)
	}

	if _, err := st.Filters().Save(ctx, "", "winners", "winrate>50"); err != nil {
		t.Fatal(err)
	}

	return st
}

// assertSameExportedContent compares every family the exporter writes
// between two reopened export files, by content rather than by id (ids are
// reassigned independently by each export run).
func assertSameExportedContent(t *testing.T, ctx context.Context, a, b *sqlite.Storage) {
	t.Helper()

	ca, err := a.Metadata().Counts(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	cb, err := b.Metadata().Counts(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if ca != cb {
		t.Fatalf("Metadata().Counts differ:\n GUI/CLI: %+v\n server : %+v", ca, cb)
	}

	var statesA, statesB []uint64
	for p, err := range a.Positions().List(ctx, "", storage.ListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		statesA = append(statesA, engine.ZobristHash(p))
	}
	for p, err := range b.Positions().List(ctx, "", storage.ListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		statesB = append(statesB, engine.ZobristHash(p))
	}
	slices.Sort(statesA)
	slices.Sort(statesB)
	if !slices.Equal(statesA, statesB) {
		t.Fatalf("exported position sets differ (by Zobrist hash):\n GUI/CLI: %v\n server : %v", statesA, statesB)
	}

	var namesA, namesB []string
	for c, err := range a.Collections().List(ctx, "") {
		if err != nil {
			t.Fatal(err)
		}
		namesA = append(namesA, c.Name)
	}
	for c, err := range b.Collections().List(ctx, "") {
		if err != nil {
			t.Fatal(err)
		}
		namesB = append(namesB, c.Name)
	}
	slices.Sort(namesA)
	slices.Sort(namesB)
	if !slices.Equal(namesA, namesB) {
		t.Fatalf("exported collections differ:\n GUI/CLI: %v\n server : %v", namesA, namesB)
	}

	var toursA, toursB []string
	for tt, err := range a.Tournaments().List(ctx, "", storage.ListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		toursA = append(toursA, tt.Name)
	}
	for tt, err := range b.Tournaments().List(ctx, "", storage.ListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		toursB = append(toursB, tt.Name)
	}
	slices.Sort(toursA)
	slices.Sort(toursB)
	if !slices.Equal(toursA, toursB) {
		t.Fatalf("exported tournaments differ:\n GUI/CLI: %v\n server : %v", toursA, toursB)
	}

	var matchesA, matchesB []string
	for m, err := range a.Matches().List(ctx, "", storage.MatchListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		matchesA = append(matchesA, m.Player1Name+" vs "+m.Player2Name)
	}
	for m, err := range b.Matches().List(ctx, "", storage.MatchListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		matchesB = append(matchesB, m.Player1Name+" vs "+m.Player2Name)
	}
	slices.Sort(matchesA)
	slices.Sort(matchesB)
	if !slices.Equal(matchesA, matchesB) {
		t.Fatalf("exported matches differ:\n GUI/CLI: %v\n server : %v", matchesA, matchesB)
	}

	var filtersA, filtersB []string
	for f, err := range a.Filters().List(ctx, "") {
		if err != nil {
			t.Fatal(err)
		}
		filtersA = append(filtersA, f.Name)
	}
	for f, err := range b.Filters().List(ctx, "") {
		if err != nil {
			t.Fatal(err)
		}
		filtersB = append(filtersB, f.Name)
	}
	slices.Sort(filtersA)
	slices.Sort(filtersB)
	if !slices.Equal(filtersA, filtersB) {
		t.Fatalf("exported filters differ:\n GUI/CLI: %v\n server : %v", filtersA, filtersB)
	}
}

package ingest

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// exportOneMatch stores m in a fresh library, exports it and returns the
// path of the exported file.
func exportOneMatch(t *testing.T, m domain.Match) string {
	t.Helper()
	ctx := context.Background()
	src, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	if _, err := src.Matches().Save(ctx, "", &m); err != nil {
		t.Fatal(err)
	}
	outPath := filepath.Join(t.TempDir(), "export.sqlite")
	out, err := os.Create(outPath)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	if err := (SQLiteExporter{S: src}).Export(ctx, "", out, WholeTenant(FormatSQLite)); err != nil {
		t.Fatalf("export: %v", err)
	}
	return outPath
}

// Every column of the match table is either carried by an export or left
// behind with a reason: a new column cannot slip into exports, nor out of
// them, without someone deciding (ADR-0007).
func TestExportMatchColumnsClassified(t *testing.T) {
	path := exportOneMatch(t, domain.Match{Player1Name: "A", Player2Name: "B"})
	db, err := sql.Open("sqlite", sqlite.DSN(path))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	rows, err := db.Query(`SELECT name FROM pragma_table_info('match')`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var col string
		if err := rows.Scan(&col); err != nil {
			t.Fatal(err)
		}
		_, left := notExportedMatchColumns[col]
		carried := slices.Contains(exportedMatchColumns, col)
		if carried == left {
			t.Errorf("match column %q: carried=%v, left behind=%v — name it in exactly one list", col, carried, left)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
}

// The source metadata travels with the match, and opening the export reads
// it back as stored — unknown stays unknown.
func TestExportCarriesMatchSourceMetadata(t *testing.T) {
	elo, exp, yes := 1854.5, 205, true
	m := domain.Match{Player1Name: "A", Player2Name: "B"}
	m.Player1Elo, m.Player1Experience = &elo, &exp
	m.Transcriber, m.HasJacoby, m.EngineVersion = "Carol", &yes, "eXtreme Gammon, file format 30"
	path := exportOneMatch(t, m)

	ctx := context.Background()
	dst, err := sqlite.Open(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer dst.Close()
	var got *domain.Match
	for mm, err := range dst.Matches().List(ctx, "", storage.MatchListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		got = mm
	}
	if got == nil {
		t.Fatal("export holds no match")
	}
	if got.Player1Elo == nil || *got.Player1Elo != elo || got.Player1Experience == nil || *got.Player1Experience != exp {
		t.Errorf("rating/experience not carried: %v %v", got.Player1Elo, got.Player1Experience)
	}
	if got.Player2Elo != nil || got.HasBeaver != nil {
		t.Errorf("unknown became known: elo2=%v beaver=%v", got.Player2Elo, got.HasBeaver)
	}
	if got.Transcriber != "Carol" || got.HasJacoby == nil || !*got.HasJacoby || got.EngineVersion != m.EngineVersion {
		t.Errorf("transcriber/rules/version not carried: %+v", got)
	}
}

// An XG file states its ratings, experience, transcriber, session rules and
// file format; MapXG keeps them all.
func TestMapXGSourceMetadata(t *testing.T) {
	g, err := MapXG("../../../testdata/test.xg")
	if err != nil {
		t.Fatal(err)
	}
	m := g.Match
	if m.Player1Elo == nil || m.Player2Elo == nil || *m.Player1Elo <= 0 {
		t.Errorf("ratings not read: %v %v", m.Player1Elo, m.Player2Elo)
	}
	if m.Player1Experience == nil || m.Player2Experience == nil {
		t.Error("experience not read")
	}
	if m.HasJacoby == nil || m.HasBeaver == nil {
		t.Error("session rules not read")
	}
	if m.Transcriber == "" {
		t.Error("transcriber not read")
	}
	if m.EngineVersion == "" {
		t.Error("file format version not read")
	}
}

// A money session's rules reach every position; at a match score they do
// not, whatever the file says.
func TestCopySessionRules(t *testing.T) {
	yes := true
	build := func(length int32) *MatchGraph {
		g := &MatchGraph{Match: domain.Match{MatchLength: length, HasJacoby: &yes, HasBeaver: &yes}}
		g.Games = []GameGraph{{Moves: []MoveGraph{{Position: &domain.Position{}}, {}}}}
		return g
	}
	money := build(0)
	copySessionRules(money)
	if p := money.Games[0].Moves[0].Position; p.HasJacoby != 1 || p.HasBeaver != 1 {
		t.Errorf("money position: jacoby=%d beaver=%d, want 1 1", p.HasJacoby, p.HasBeaver)
	}
	match := build(7)
	copySessionRules(match)
	if p := match.Games[0].Moves[0].Position; p.HasJacoby != 0 || p.HasBeaver != 0 {
		t.Errorf("match position: jacoby=%d beaver=%d, want 0 0", p.HasJacoby, p.HasBeaver)
	}
}

// Re-importing a file gives the stored match the metadata it lacks and never
// overwrites what it states: a corpus imported before the columns existed
// gains them from its files.
func TestReimportFillsMissingSourceMetadata(t *testing.T) {
	ctx := context.Background()
	s, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	write := func() WriteResult {
		g, err := MapXG("../../../testdata/test.xg")
		if err != nil {
			t.Fatal(err)
		}
		tx, err := s.BeginTx(ctx)
		if err != nil {
			t.Fatal(err)
		}
		res, err := WriteMatch(ctx, tx, "", g, nil)
		if err != nil {
			_ = tx.Rollback()
			t.Fatal(err)
		}
		if err := tx.Commit(); err != nil {
			t.Fatal(err)
		}
		return res
	}
	id := write().MatchID
	stored, err := s.Matches().Get(ctx, "", id)
	if err != nil {
		t.Fatal(err)
	}
	sentinel := 1234.5
	domain.CopySourceMetadata(stored, &domain.Match{})
	stored.Player2Elo = &sentinel
	if err := s.Matches().ReplaceHeader(ctx, "", id, stored); err != nil {
		t.Fatal(err)
	}

	if res := write(); !res.Skipped || res.MatchID != id {
		t.Fatalf("second import: %+v, want a skipped duplicate of %d", res, id)
	}
	got, err := s.Matches().Get(ctx, "", id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Player1Elo == nil || got.Transcriber == "" || got.HasJacoby == nil || got.EngineVersion == "" {
		t.Errorf("missing metadata not filled: %+v", got)
	}
	if got.Player2Elo == nil || *got.Player2Elo != sentinel {
		t.Errorf("stated rating overwritten: %v", got.Player2Elo)
	}
}

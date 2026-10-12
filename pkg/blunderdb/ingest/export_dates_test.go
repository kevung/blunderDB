package ingest

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// originDate is the date the fixture's rows bear at the source, as every
// backend spells a timestamp it reads back.
const originDate = "2001-02-03 04:05:06"

// datedTables lists, per table, the date columns the fixture backdates.
var datedTables = map[string][]string{
	"collection_position": {"added_at"},
	"lesson":              {"created_at", "updated_at"},
	"transcription":       {"created_at", "updated_at"},
	"anki_deck":           {"created_at", "updated_at"},
	"player_alias":        {"created_at"},
}

// An export keeps the dates its rows bear at the source: a membership, a
// lesson, a draft, a deck or an alias does not date from the day the file was
// handed over.
func TestExportKeepsOriginDates(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "src.db")
	src, err := sqlite.Open(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	checkExportKeepsOriginDates(t, src, "", func(table, column string) {
		db, err := sql.Open("sqlite", path)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		if _, err := db.ExecContext(ctx, `UPDATE `+table+` SET `+column+` = ?`, originDate); err != nil {
			t.Fatal(err)
		}
	})
}

// checkExportKeepsOriginDates builds one row of every dated family in src
// under scope, ages them through backdate, exports the scope and reads the
// dates back from the file.
func checkExportKeepsOriginDates(t *testing.T, src storage.Storage, scope string, backdate func(table, column string)) {
	t.Helper()
	ctx := context.Background()
	p := domain.InitializePosition()
	pid, err := src.Positions().Save(ctx, scope, &p)
	if err != nil {
		t.Fatal(err)
	}
	cid, err := src.Collections().Create(ctx, scope, "Ouvertures", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := src.Collections().AddPosition(ctx, scope, cid, pid); err != nil {
		t.Fatal(err)
	}
	lid, err := src.Lessons().Create(ctx, scope, "Leçon", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := src.Lessons().AddStep(ctx, scope, lid, domain.LessonStep{Title: "Un", PositionID: pid}); err != nil {
		t.Fatal(err)
	}
	mid, err := src.Matches().Save(ctx, scope, &domain.Match{Player1Name: "Alice", Player2Name: "Zoé", MatchLength: 5})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := src.Matches().CreateGame(ctx, scope, &domain.Game{MatchID: mid, GameNumber: 1}); err != nil {
		t.Fatal(err)
	}
	if err := src.Aliases().Set(ctx, scope, storage.AliasPlayer, "A.", "Alice"); err != nil {
		t.Fatal(err)
	}
	if _, err := src.Transcriptions().Save(ctx, scope, &storage.Transcription{
		MatchID: mid, Label: "Alice", FormatVersion: "3",
		Document: `{"format_version":3,"header":{"match_length":5},"actions":[],"cursor":0}`,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := src.Anki().CreateDeck(ctx, scope, "Paquet", "", domain.AnkiSourceCollection, cid, ""); err != nil {
		t.Fatal(err)
	}
	for table, columns := range datedTables {
		for _, column := range columns {
			backdate(table, column)
		}
	}

	out := filepath.Join(t.TempDir(), "export.db")
	if _, err := ExportSQLite(ctx, src, scope, out, WholeTenant(FormatSQLite)); err != nil {
		t.Fatal(err)
	}
	dst, err := sqlite.Open(ctx, out, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer dst.Close()

	got := map[string]string{}
	for c, err := range dst.Collections().List(ctx, "") {
		if err != nil {
			t.Fatal(err)
		}
		for m, err := range dst.Collections().Members(ctx, "", c.ID) {
			if err != nil {
				t.Fatal(err)
			}
			got["collection_position.added_at"] = m.AddedAt
		}
	}
	lessons, err := dst.Lessons().List(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range lessons {
		got["lesson.created_at"], got["lesson.updated_at"] = l.CreatedAt, l.UpdatedAt
	}
	for tr, err := range dst.Transcriptions().List(ctx, "") {
		if err != nil {
			t.Fatal(err)
		}
		got["transcription.created_at"], got["transcription.updated_at"] = tr.CreatedAt, tr.UpdatedAt
	}
	for d, err := range dst.Anki().ListDecks(ctx, "") {
		if err != nil {
			t.Fatal(err)
		}
		got["anki_deck.created_at"], got["anki_deck.updated_at"] = d.CreatedAt, d.UpdatedAt
	}
	aliases, err := dst.Aliases().List(ctx, "", storage.AliasPlayer)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range aliases {
		got["player_alias.created_at"] = a.CreatedAt
	}
	for table, columns := range datedTables {
		for _, column := range columns {
			key := table + "." + column
			if got[key] != originDate {
				t.Errorf("%s in the file = %q, want the source's %q", key, got[key], originDate)
			}
		}
	}
}

package database

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// A database shared by its producer keeps each comment's signature through the
// desktop import, on a position the recipient already holds (merged) as on a
// new one (copied whole), and the recipient's own name signs none of them.
func TestCommitImportDatabaseKeepsCommentAuthors(t *testing.T) {
	isolateIdentity(t)
	src := newTestDB(t)
	shared := initialPosition()
	sharedID, err := src.SavePosition(&shared)
	if err != nil {
		t.Fatal(err)
	}
	fresh := initialPosition()
	fresh.Board.Points[1].Checkers, fresh.Board.Points[2].Checkers = fresh.Board.Points[2].Checkers, fresh.Board.Points[1].Checkers
	fresh.Board.Points[1].Color, fresh.Board.Points[2].Color = fresh.Board.Points[2].Color, fresh.Board.Points[1].Color
	freshID, err := src.SavePosition(&fresh)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		author, text string
		pos          int64
	}{{"Élodie", "cube too early", sharedID}, {"", "unsigned note", sharedID}, {"Bob", "second row", freshID}, {"Carol", "third row", freshID}} {
		src.SetCommentAuthor(c.author)
		if err := src.AddComment(c.pos, c.text); err != nil {
			t.Fatal(err)
		}
	}
	path := exportTo(t, src, filepath.Join(t.TempDir(), "partage.db"), ExportOptions{
		AllPositions: true, IncludeAnalysis: true, IncludeComments: true,
	})

	dst := newTestDB(t)
	dst.SetCommentAuthor("Importer")
	mine := initialPosition()
	mineID, err := dst.SavePosition(&mine)
	if err != nil {
		t.Fatal(err)
	}
	if err := dst.AddComment(mineID, "my own note"); err != nil {
		t.Fatal(err)
	}
	if _, err := dst.CommitImportDatabase(path); err != nil {
		t.Fatal(err)
	}

	all, err := dst.GetAllComments()
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, c := range all {
		got[c.Text] = c.Author
	}
	want := map[string]string{"my own note": "Importer", "cube too early": "Élodie", "unsigned note": "", "second row": "Bob", "third row": "Carol"}
	for text, author := range want {
		if a, ok := got[text]; !ok || a != author {
			t.Errorf("comment %q signed %q (present %v), want %q", text, a, ok, author)
		}
	}
}

// An author the source file records untrimmed comes out of the desktop import
// as the daemon's import writes it: normalised by the comment contract.
func TestCommitImportDatabaseNormalisesCommentAuthorLikeDaemon(t *testing.T) {
	isolateIdentity(t)
	src := newTestDB(t)
	pos := initialPosition()
	id, err := src.SavePosition(&pos)
	if err != nil {
		t.Fatal(err)
	}
	if err := src.AddComment(id, "watch the gap"); err != nil {
		t.Fatal(err)
	}
	path := exportTo(t, src, filepath.Join(t.TempDir(), "partage.db"), ExportOptions{
		AllPositions: true, IncludeAnalysis: true, IncludeComments: true,
	})
	raw, err := openExistingSQLite(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(`UPDATE comment SET author = '  Coach  '`); err != nil {
		t.Fatal(err)
	}
	raw.Close()

	dst := newTestDB(t)
	if _, err := dst.CommitImportDatabase(path); err != nil {
		t.Fatal(err)
	}
	all, err := dst.GetAllComments()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || all[0].Author != "Coach" {
		t.Fatalf("desktop import: comments %+v, want one by %q", all, "Coach")
	}

	ctx := context.Background()
	daemon, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "daemon.db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer daemon.Close()
	if _, err := (ingest.DBImporter{S: daemon}).Import(ctx, "", ingest.Source{Format: ingest.FormatNativeDB, Path: path}, nil); err != nil {
		t.Fatal(err)
	}
	for c, err := range daemon.Comments().ListAll(ctx, "", storage.ListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		if c.Author != all[0].Author {
			t.Errorf("daemon import: author %q, desktop %q", c.Author, all[0].Author)
		}
	}
}

package database

import (
	"path/filepath"
	"testing"
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

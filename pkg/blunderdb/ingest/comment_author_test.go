package ingest

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// importer is the person running an import: the daemon's X-User-Name, the
// desktop's « Votre nom ». An imported note is never signed with it.
const importer = "Importer"

func memStore(t *testing.T) storage.Storage {
	t.Helper()
	s, err := sqlite.Open(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// commentAuthors returns every non-empty comment's author, keyed by its text.
func commentAuthors(t *testing.T, s storage.Storage) map[string]string {
	t.Helper()
	out := map[string]string{}
	for c, err := range s.Comments().ListAll(context.Background(), "", storage.ListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		out[c.Text] = c.Author
	}
	if len(out) == 0 {
		t.Fatal("no comment imported")
	}
	return out
}

func writeSigned(t *testing.T, s storage.Storage, g *MatchGraph) {
	t.Helper()
	ctx := storage.WithCommentAuthor(context.Background(), importer)
	tx, err := s.BeginTx(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := WriteMatch(ctx, tx, "", g, nil); err != nil {
		_ = tx.Rollback()
		t.Fatalf("WriteMatch: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
}

func commentedXG(t *testing.T) *MatchGraph {
	t.Helper()
	g, err := MapXG(filepath.Join("..", "..", "..", "testdata", "match_with_comment.xg"))
	if err != nil {
		t.Fatalf("MapXG: %v", err)
	}
	return g
}

// The notes of an eXtreme Gammon file are XG's own: signed "XG", whoever
// imports the file.
func TestImportSignsXGNotesAsXG(t *testing.T) {
	s := memStore(t)
	writeSigned(t, s, commentedXG(t))
	for text, author := range commentAuthors(t, s) {
		if author != "XG" {
			t.Errorf("comment %q signed %q, want XG", text, author)
		}
	}
}

// Any other source's notes are the transcriber's the file names, or nobody's:
// never the importer's.
func TestImportSignsTranscriberNeverImporter(t *testing.T) {
	for _, tc := range []struct{ transcriber, want string }{{"Carol", "Carol"}, {"", ""}} {
		s := memStore(t)
		g := commentedXG(t)
		g.CommentOrigin = domain.CommentOriginGnuBG
		g.Match.Transcriber = tc.transcriber
		writeSigned(t, s, g)
		for text, author := range commentAuthors(t, s) {
			if author != tc.want {
				t.Errorf("transcriber %q: comment %q signed %q, want %q", tc.transcriber, text, author, tc.want)
			}
		}
	}
}

// A .db travels with its signatures: export copies the author through the
// issuance allow-list, and the native import keeps each one rather than
// signing in the importer's name. The NDJSON interchange does the same.
func TestDBAndJSONRoundTripKeepAuthors(t *testing.T) {
	ctx := context.Background()
	src := memStore(t)
	p := domain.InitializePosition()
	pid, err := src.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range [][2]string{{"Élodie", "cube too early"}, {"", "unsigned note"}} {
		if _, err := src.Comments().Add(storage.WithCommentAuthor(ctx, c[0]), "", pid, c[1]); err != nil {
			t.Fatal(err)
		}
	}
	want := map[string]string{"cube too early": "Élodie", "unsigned note": ""}

	dbPath := filepath.Join(t.TempDir(), "shared.db")
	out, err := os.Create(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := (SQLiteExporter{S: src}).Export(ctx, "", out, WholeTenant(FormatSQLite)); err != nil {
		out.Close()
		t.Fatalf("export: %v", err)
	}
	out.Close()

	signed := storage.WithCommentAuthor(ctx, importer)
	viaDB := memStore(t)
	if _, err := (DBImporter{S: viaDB}).Import(signed, "", Source{Format: FormatNativeDB, Path: dbPath}, nil); err != nil {
		t.Fatalf("DBImporter: %v", err)
	}
	assertAuthors(t, ".db", commentAuthors(t, viaDB), want)

	var buf bytes.Buffer
	if err := (JSONExporter{S: src}).Export(ctx, "", &buf, WholeTenant(FormatJSON)); err != nil {
		t.Fatalf("JSON export: %v", err)
	}
	viaJSON := memStore(t)
	if _, err := (JSONImporter{S: viaJSON}).Import(signed, "", Source{Format: FormatJSON, Reader: &buf}, nil); err != nil {
		t.Fatalf("JSONImporter: %v", err)
	}
	assertAuthors(t, "NDJSON", commentAuthors(t, viaJSON), want)
}

func assertAuthors(t *testing.T, via string, got, want map[string]string) {
	t.Helper()
	for text, author := range want {
		if a, ok := got[text]; !ok || a != author {
			t.Errorf("%s: comment %q signed %q (present %v), want %q", via, text, a, ok, author)
		}
	}
}

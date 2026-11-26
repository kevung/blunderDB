package database

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/arkqr"
)

func archivePayloads(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	frames, err := arkqr.ReadPage(data)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, m := range arkqr.Assemble(frames).Matches {
		out = append(out, string(m))
	}
	sort.Strings(out)
	return out
}

// playedHashes are the Zobrist hashes of the checker positions played.
func playedHashes(t *testing.T, db *Database) []int64 {
	t.Helper()
	rows, err := db.db.Query(`SELECT DISTINCT p.zobrist_hash FROM move mv JOIN position p ON p.id = mv.position_id
		WHERE mv.dice_1 > 0 ORDER BY 1`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var z int64
		if err := rows.Scan(&z); err != nil {
			t.Fatal(err)
		}
		out = append(out, z)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestPaperArchiveRoundTrip(t *testing.T) {
	t.Parallel()
	src := newTestDB(t)
	for _, f := range []string{"test.mat", "match_with_comment.xg", "charlot1-charlot2_7p_2025-11-08-2305.xg"} {
		path := filepath.Join("testdata", f)
		var err error
		if strings.HasSuffix(f, ".xg") {
			_, err = src.ImportXGMatch(path)
		} else {
			_, err = src.ImportGnuBGMatch(path)
		}
		if err != nil {
			t.Fatalf("%s: %v", f, err)
		}
	}
	var pid int64
	if err := src.db.QueryRow(`SELECT position_id FROM move ORDER BY id LIMIT 1 OFFSET 12`).Scan(&pid); err != nil {
		t.Fatal(err)
	}
	if err := src.AddComment(pid, "à revoir sur papier"); err != nil {
		t.Fatal(err)
	}
	if _, err := src.db.Exec(`UPDATE position SET flagged = 1 WHERE id = ?`, pid); err != nil {
		t.Fatal(err)
	}
	var z int64
	if err := src.db.QueryRow(`SELECT zobrist_hash FROM position WHERE id = ?`, pid).Scan(&z); err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(t.TempDir(), "archive.html")
	exp, err := src.ExportPaperArchive(nil, out, "Club de test")
	if err != nil {
		t.Fatal(err)
	}
	if exp.Matches != 3 || exp.Codes != 4 || len(exp.Skipped) != 0 {
		t.Fatalf("export: %+v", exp)
	}
	page, err := os.ReadFile(out)
	if err != nil || !strings.Contains(string(page), "Club de test") || !IsPaperArchivePage(out) {
		t.Fatalf("page: %v", err)
	}

	dst := newTestDB(t)
	imp, err := dst.ImportPaperArchive([]string{out})
	if err != nil {
		t.Fatal(err)
	}
	if len(imp.Imported) != 3 || imp.Duplicates != 0 || len(imp.Missing) != 0 || len(imp.Failed) != 0 {
		t.Fatalf("import: %+v", imp)
	}
	// Archived again from the library it was imported into, every match
	// codes to the same bytes: same players, dice, plays, cube, results,
	// notes.
	out2 := filepath.Join(t.TempDir(), "again.html")
	if _, err := dst.ExportPaperArchive(nil, out2, ""); err != nil {
		t.Fatal(err)
	}
	if got, want := archivePayloads(t, out2), archivePayloads(t, out); !reflect.DeepEqual(got, want) {
		t.Fatalf("the archive changed the matches")
	}
	// Every position a match was played from in the source is in the copy.
	if got, want := playedHashes(t, dst), playedHashes(t, src); !reflect.DeepEqual(got, want) {
		t.Fatalf("positions differ: %d vs %d", len(got), len(want))
	}
	var text string
	var flagged bool
	if err := dst.db.QueryRow(`SELECT c.text, p.flagged FROM comment c JOIN position p ON p.id = c.position_id
		WHERE p.zobrist_hash = ?`, z).Scan(&text, &flagged); err != nil || text != "à revoir sur papier" || !flagged {
		t.Fatalf("note: %q flagged=%v (%v)", text, flagged, err)
	}

	// The same archive again, and into the library it came from: duplicates.
	for _, db := range []*Database{dst, src} {
		again, err := db.ImportPaperArchive([]string{out})
		if err != nil || len(again.Imported) != 0 || again.Duplicates != 3 {
			t.Fatalf("re-import: %+v (%v)", again, err)
		}
	}
}

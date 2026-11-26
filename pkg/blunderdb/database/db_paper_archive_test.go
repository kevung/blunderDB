package database

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/arkqr"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
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

// A money session keeps through the archive what a .mat cannot say: the
// session rules, the hour it was played at, the players' ratings.
func TestPaperArchiveRoundTrip_MoneySession(t *testing.T) {
	t.Parallel()
	src := newTestDB(t)
	state, err := src.CreateTranscription(transcript.Header{MatchLength: 0, Player1: "Kévin", Player2: "Alice", Jacoby: true, Beaver: false})
	if err != nil {
		t.Fatal(err)
	}
	typeChecker(t, src, state.ID, 6, 3)
	typeChecker(t, src, state.ID, 5, 2)
	fin, err := src.FinishTranscription(state.ID)
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 3, 14, 21, 37, 0, 0, time.UTC)
	if _, err := src.db.Exec(`UPDATE match SET match_date = ?, player1_elo = 1712.5, player2_elo = 1550,
		player1_experience = 400 WHERE id = ?`, at, fin.MatchID); err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(t.TempDir(), "money.html")
	if exp, err := src.ExportPaperArchive([]int64{fin.MatchID}, out, ""); err != nil || exp.Matches != 1 {
		t.Fatalf("export: %+v (%v)", exp, err)
	}
	dst := newTestDB(t)
	imp, err := dst.ImportPaperArchive([]string{out})
	if err != nil || len(imp.Imported) != 1 {
		t.Fatalf("import: %+v (%v)", imp, err)
	}
	m, err := dst.GetMatchByID(imp.Imported[0])
	if err != nil {
		t.Fatal(err)
	}
	if m.MatchLength != 0 || m.HasJacoby == nil || !*m.HasJacoby || m.HasBeaver == nil || *m.HasBeaver {
		t.Fatalf("session rules: length %d jacoby %v beaver %v", m.MatchLength, m.HasJacoby, m.HasBeaver)
	}
	if !m.MatchDate.Equal(at) {
		t.Fatalf("date %v, want %v", m.MatchDate, at)
	}
	if m.Player1Elo == nil || *m.Player1Elo != 1712.5 || m.Player2Elo == nil || *m.Player2Elo != 1550 ||
		m.Player1Experience == nil || *m.Player1Experience != 400 || m.Player2Experience != nil {
		t.Fatalf("ratings: %v %v %v %v", m.Player1Elo, m.Player2Elo, m.Player1Experience, m.Player2Experience)
	}
	// Every position keeps the session's rules, so the copy dedups on them.
	var played, without int
	if err := dst.db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(COALESCE(p.has_jacoby, 0) = 0), 0) FROM position p
		JOIN move mv ON mv.position_id = p.id JOIN game g ON g.id = mv.game_id WHERE g.match_id = ?`, m.ID).Scan(&played, &without); err != nil || played == 0 || without != 0 {
		t.Fatalf("%d of %d positions lost the Jacoby rule (%v)", without, played, err)
	}

	// The common pipeline — batch import, drag and drop — takes the page too.
	batch := newTestDB(t)
	outs, err := batch.ImportFiles([]string{out}, ImportFilesOptions{})
	if err != nil || len(outs) != 1 || outs[0].MatchID == 0 {
		t.Fatalf("pipeline: %+v (%v)", outs, err)
	}
	if got, err := batch.GetMatchByID(outs[0].MatchID); err != nil || !got.MatchDate.Equal(at) || got.HasJacoby == nil || !*got.HasJacoby {
		t.Fatalf("pipeline match: %+v (%v)", got, err)
	}
}

package ingest

import (
	"context"
	"database/sql"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// The importers read files a user downloads from anywhere: XG, gnubg (.sgf,
// .mat, .txt), BGBlitz (.bgf, text positions). The fuzz targets below hold
// every one of them to the same contract — no panic, no hang, an error rather
// than a crash — and, for what a mapper accepts, the store it is written into
// stays coherent: foreign keys hold, every stored position is held by
// something (positionIsHeldSQL), and SQLite's own integrity check passes.

// fuzzDeadline bounds one input. A mapper that loops on a crafted file is a
// denial of service on the import endpoint, so a hang is a failure, not a
// slow test.
const fuzzDeadline = 10 * time.Second

// withinDeadline runs fn and fails when it has not returned in time. The
// goroutine is abandoned on timeout; the failure ends the fuzz run anyway.
func withinDeadline[T any](t *testing.T, what string, fn func() (T, error)) (T, error) {
	t.Helper()
	type result struct {
		v   T
		err error
	}
	done := make(chan result, 1)
	go func() {
		v, err := fn()
		done <- result{v, err}
	}()
	select {
	case r := <-done:
		return r.v, r.err
	case <-time.After(fuzzDeadline):
		t.Fatalf("%s did not return within %s", what, fuzzDeadline)
		var zero T
		return zero, nil
	}
}

// fuzzStore opens a fresh in-memory store and keeps the raw handle, so the
// coherence checks can run SQL the storage contract does not expose.
func fuzzStore(t *testing.T) (*sqlite.Storage, *sql.DB) {
	t.Helper()
	ctx := context.Background()
	db, err := sql.Open("sqlite", sqlite.DSN(":memory:"))
	if err != nil {
		t.Fatal(err)
	}
	sqlite.ConfigurePool(db, ":memory:")
	if err := sqlite.Bootstrap(ctx, db); err != nil {
		db.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return sqlite.New(db), db
}

// assertCoherent checks the invariants any import must leave behind.
func assertCoherent(t *testing.T, db *sql.DB) {
	t.Helper()
	if violations := countRows(t, db, `PRAGMA foreign_key_check`); violations > 0 {
		t.Fatalf("%d foreign-key violations after import", violations)
	}
	var orphans int
	if err := db.QueryRow(`SELECT COUNT(*) FROM position WHERE NOT (` + positionIsHeldForTest + `)`).Scan(&orphans); err != nil {
		t.Fatal(err)
	}
	if orphans > 0 {
		t.Fatalf("%d positions held by nothing after import", orphans)
	}
	var check string
	if err := db.QueryRow(`PRAGMA integrity_check`).Scan(&check); err != nil {
		t.Fatal(err)
	}
	if check != "ok" {
		t.Fatalf("integrity_check: %s", check)
	}
}

// positionIsHeldForTest mirrors storage/sqlite's positionIsHeldSQL (which is
// unexported there); a drift shows up as orphans reported here.
const positionIsHeldForTest = `EXISTS (SELECT 1 FROM move WHERE position_id = position.id)
	OR EXISTS (SELECT 1 FROM collection_position WHERE position_id = position.id)
	OR EXISTS (SELECT 1 FROM anki_card WHERE position_id = position.id)
	OR EXISTS (SELECT 1 FROM lesson_step WHERE position_id = position.id)
	OR EXISTS (SELECT 1 FROM comment WHERE position_id = position.id AND origin = 'user')
	OR position.individually_imported = 1
	OR position.flagged = 1`

// writeAndCheck writes a mapped match into a fresh store and checks it. A
// write error is acceptable (the graph may be refused); a half-written
// transaction is not, which is what the coherence check catches.
func writeAndCheck(t *testing.T, g *MatchGraph) (*sqlite.Storage, *sql.DB, WriteResult, bool) {
	t.Helper()
	ctx := context.Background()
	s, db := fuzzStore(t)
	tx, err := s.BeginTx(ctx)
	if err != nil {
		t.Fatal(err)
	}
	res, err := withinDeadline(t, "WriteMatch", func() (WriteResult, error) {
		return WriteMatch(ctx, tx, "", g, nil)
	})
	if err != nil {
		_ = tx.Rollback()
		assertCoherent(t, db)
		return s, db, res, false
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
	assertCoherent(t, db)
	return s, db, res, true
}

// fuzzMoveCap bounds the part of an analysed match written per input. A whole
// XG match costs seconds to store (every analysis is compressed), which would
// leave the fuzzer a handful of executions a minute; the coherence of a write
// does not depend on how many moves it carries.
const fuzzMoveCap = 24

// capMoves keeps the first fuzzMoveCap moves of g, games in order.
func capMoves(g *MatchGraph) *MatchGraph {
	left := fuzzMoveCap
	for i := range g.Games {
		if left <= 0 {
			g.Games = g.Games[:i]
			break
		}
		if len(g.Games[i].Moves) > left {
			g.Games[i].Moves = g.Games[i].Moves[:left]
		}
		left -= len(g.Games[i].Moves)
	}
	return g
}

// fuzzFile writes data where a path-taking mapper can read it.
func fuzzFile(t *testing.T, name string, data []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// addSeedFile adds a repository fixture as a seed, cut to limit bytes: the
// mutator spends its time in proportion to the input's size, and the head of
// a file carries its header and first games.
func addSeedFile(f *testing.F, rel string, limit int) {
	f.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", rel))
	if err != nil {
		f.Fatalf("seed %s: %v", rel, err)
	}
	if limit > 0 && len(data) > limit {
		data = data[:limit]
	}
	f.Add(data)
}

// FuzzImportGnuBGMAT drives the gnubg/Jellyfish .mat reader, the format a
// clipboard paste also arrives in, through mapping and a store write.
func FuzzImportGnuBGMAT(f *testing.F) {
	addSeedFile(f, "test.mat", 0)
	addSeedFile(f, "gnubg_roll_then_resign_1p.mat", 0)
	addSeedFile(f, "gnubg_selfplay_drops_7p.mat", 0)
	addSeedFile(f, "test.txt", 4096)
	f.Add([]byte(" 1 point match\n\n Game 1\n A : 0                 B : 0\n  1) 31: 8/5 6/5\n"))

	f.Fuzz(func(t *testing.T, data []byte) {
		g, err := withinDeadline(t, "MapGnuBGText", func() (*MatchGraph, error) {
			return MapGnuBGText(string(data))
		})
		if err != nil || g == nil {
			return
		}
		writeAndCheck(t, g)
	})
}

// FuzzImportGnuBGSGF drives the gnubg .sgf reader, which carries analysis
// and move coordinates in the absolute frame.
func FuzzImportGnuBGSGF(f *testing.F) {
	addSeedFile(f, "test.sgf", 16384)
	addSeedFile(f, "charlot1-charlot2_7p_2025-11-08-2305.sgf", 16384)
	f.Add([]byte("(;FF[4]GM[6]CA[UTF-8]AP[GNU Backgammon:1.07]MI[length:1][game:0][ws:0][bs:0]RU[]PW[A]PB[B];B[31lh];W[64ab])"))

	f.Fuzz(func(t *testing.T, data []byte) {
		path := fuzzFile(t, "in.sgf", data)
		g, err := withinDeadline(t, "MapGnuBG(.sgf)", func() (*MatchGraph, error) {
			return MapGnuBG(path)
		})
		if err != nil || g == nil {
			return
		}
		writeAndCheck(t, capMoves(g))
	})
}

// FuzzImportXG drives the eXtreme Gammon .xg reader (a binary, compressed
// container).
func FuzzImportXG(f *testing.F) {
	addSeedFile(f, "HsbtMarseille_main_ronde4_LamourDeCaslouGildas_UngerKevin_7p.xg", 0)
	f.Add([]byte("RGMH"))

	f.Fuzz(func(t *testing.T, data []byte) {
		path := fuzzFile(t, "in.xg", data)
		g, err := withinDeadline(t, "MapXG", func() (*MatchGraph, error) {
			return MapXG(path)
		})
		if err != nil || g == nil {
			return
		}
		writeAndCheck(t, capMoves(g))
	})
}

// FuzzImportBGF drives the BGBlitz .bgf match reader.
func FuzzImportBGF(f *testing.F) {
	addSeedFile(f, "TachiAI_V_player_Nov_2__2025__16_55.bgf", 0)
	f.Add([]byte("{}"))

	f.Fuzz(func(t *testing.T, data []byte) {
		path := fuzzFile(t, "in.bgf", data)
		g, err := withinDeadline(t, "MapBGF", func() (*MatchGraph, error) {
			return MapBGF(path)
		})
		if err != nil || g == nil {
			return
		}
		writeAndCheck(t, capMoves(g))
	})
}

// FuzzImportPosition drives the two single-position readers — BGBlitz text
// and XG .xgp — through WritePosition. xgp selects which one reads data.
func FuzzImportPosition(f *testing.F) {
	for _, name := range []string{"01_checkerPosition_FR.txt", "02_NDT_FR.txt", "04_DP_EN.txt", "06_RT_FR.txt"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "bgf_positions", name))
		if err != nil {
			f.Fatal(err)
		}
		f.Add(data, false)
	}
	xgp, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "xgp", "Position 10.xgp"))
	if err != nil {
		f.Fatal(err)
	}
	f.Add(xgp, true)

	f.Fuzz(func(t *testing.T, data []byte, xgp bool) {
		var graphs []PositionGraph
		var err error
		if xgp {
			path := fuzzFile(t, "in.xgp", data)
			graphs, err = withinDeadline(t, "MapXGPPosition", func() ([]PositionGraph, error) {
				return MapXGPPosition(path)
			})
		} else {
			graphs, err = withinDeadline(t, "MapBGFTextPositionText", func() ([]PositionGraph, error) {
				return MapBGFTextPositionText(string(data))
			})
		}
		if err != nil || len(graphs) == 0 {
			return
		}
		ctx := context.Background()
		s, db := fuzzStore(t)
		tx, err := s.BeginTx(ctx)
		if err != nil {
			t.Fatal(err)
		}
		for i := range graphs {
			if _, err := WritePosition(ctx, tx, "", &graphs[i]); err != nil {
				_ = tx.Rollback()
				assertCoherent(t, db)
				return
			}
		}
		if err := tx.Commit(); err != nil {
			t.Fatalf("commit: %v", err)
		}
		assertCoherent(t, db)
	})
}

// FuzzMATExportRoundTrip: a match imported from .mat, exported back to .mat
// and imported again into a fresh store reaches the same positions — the
// Zobrist hashes of the positions its moves stand on are the same set.
func FuzzMATExportRoundTrip(f *testing.F) {
	addSeedFile(f, "test.mat", 0)
	addSeedFile(f, "gnubg_roll_then_resign_1p.mat", 0)
	addSeedFile(f, "gnubg_selfplay_drops_7p.mat", 0)
	addSeedFile(f, "charlot1-charlot2_7p_2025-11-08-2305.mat", 0)

	f.Fuzz(func(t *testing.T, data []byte) {
		ctx := context.Background()
		g, err := withinDeadline(t, "MapGnuBGText", func() (*MatchGraph, error) {
			return MapGnuBGText(string(data))
		})
		if err != nil || g == nil {
			return
		}
		s, db, res, ok := writeAndCheck(t, g)
		if !ok || res.MatchID == 0 {
			return
		}
		m, games, moves, err := ReadMatchForMAT(ctx, s, "", res.MatchID)
		if err != nil {
			t.Fatalf("ReadMatchForMAT on a match just written: %v", err)
		}
		out := RenderMAT(m, games, moves)
		g2, err := MapGnuBGText(out)
		if err != nil {
			t.Fatalf("the exported .mat does not read back: %v\n%s", err, out)
		}
		_, db2, _, ok := writeAndCheck(t, g2)
		if !ok {
			t.Fatalf("the exported .mat maps but does not write back\n%s", out)
		}
		before, after := moveHashes(t, db), moveHashes(t, db2)
		if !slices.Equal(before, after) {
			t.Fatalf("round trip changed the positions: %d hashes before, %d after\n%s", len(before), len(after), out)
		}
	})
}

// moveHashes lists, sorted and distinct, the Zobrist hashes of the positions
// a store's moves stand on.
func moveHashes(t *testing.T, db *sql.DB) []int64 {
	t.Helper()
	rows, err := db.Query(`SELECT DISTINCT p.zobrist_hash FROM move m JOIN position p ON p.id = m.position_id ORDER BY 1`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var h int64
		if err := rows.Scan(&h); err != nil {
			t.Fatal(err)
		}
		out = append(out, h)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// countRows counts the rows a statement returns.
func countRows(t *testing.T, db *sql.DB, query string) int {
	t.Helper()
	rows, err := db.Query(query)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		n++
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return n
}

// An XG header announcing a thumbnail or header larger than the file made
// xgparser allocate it outright (3.5 GB for a 69 KB .xgp): it is refused
// before xgparser reads it.
func TestCheckXGHeader_RefusesOversizedSizes(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "xgp", "Position 10.xgp"))
	if err != nil {
		t.Fatal(err)
	}
	if err := checkXGHeader(fuzzFile(t, "ok.xgp", data)); err != nil {
		t.Fatalf("a real .xgp is refused: %v", err)
	}
	for name, patch := range map[string]func(b []byte){
		"thumbnail": func(b []byte) { binary.LittleEndian.PutUint32(b[20:24], 0xF0000000) },
		"header":    func(b []byte) { binary.LittleEndian.PutUint32(b[8:12], 0x7FFFFFFF) },
		"negative":  func(b []byte) { binary.LittleEndian.PutUint32(b[8:12], 0xFFFFFFFF) },
	} {
		b := slices.Clone(data)
		patch(b)
		path := fuzzFile(t, name+".xgp", b)
		if _, err := MapXGPPosition(path); !errors.Is(err, ErrXGHeader) {
			t.Errorf("%s: MapXGPPosition err = %v, want ErrXGHeader", name, err)
		}
		if _, err := MapXG(path); !errors.Is(err, ErrXGHeader) {
			t.Errorf("%s: MapXG err = %v, want ErrXGHeader", name, err)
		}
	}
}

// bgfparser shifts by the cube exponent of an XGID line unchecked; a negative
// one panicked inside the text-position import instead of failing it.
func TestMapBGFTextPositionText_ParserPanicIsAnError(t *testing.T) {
	data, err := os.ReadFile("testdata/fuzz/FuzzImportPosition/753c203fcd260860")
	if err != nil {
		t.Fatal(err)
	}
	var content string
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "[]byte(") {
			content, err = strconv.Unquote(strings.TrimSuffix(strings.TrimPrefix(line, "[]byte("), ")"))
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if _, err := MapBGFTextPositionText(content); !errors.Is(err, ErrParserPanic) {
		t.Fatalf("err = %v, want ErrParserPanic", err)
	}
}

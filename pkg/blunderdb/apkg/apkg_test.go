package apkg_test

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/kevung/blunderdb/pkg/blunderdb/apkg"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

type fixture struct {
	st       storage.Storage
	deckID   int64
	handMade int64
	living   int64
	checker  domain.Position
	cube     domain.Position
}

func setup(t *testing.T) fixture {
	t.Helper()
	ctx := context.Background()
	st, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatalf("sqlite.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	f := fixture{st: st}

	f.checker = domain.InitializePosition()
	f.checker.Score = [2]int{3, 5}
	chkID, err := st.Positions().Save(ctx, "", &f.checker)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	f.checker.ID = chkID
	if err := st.Analyses().Save(ctx, "", chkID, &domain.PositionAnalysis{
		XGID: "XGID=-b----E-C---eE---c-e----B-:0:0:1:31:3:5:0:7:10",
		CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
			{Index: 1, Move: "8/5 6/5", Equity: 0.160},
			{Index: 2, Move: "24/23 13/10", Equity: 0.010},
		}},
		PlayedMoves: []string{"13/10 24/23"},
	}); err != nil {
		t.Fatalf("Save analysis: %v", err)
	}

	f.cube = domain.InitializePosition()
	f.cube.DecisionType = domain.CubeAction
	f.cube.Dice = [2]int{0, 0}
	f.cube.Score = [2]int{-1, -1}
	f.cube.Board.Points[3] = domain.Point{Checkers: 1, Color: domain.White}
	cubeID, err := st.Positions().Save(ctx, "", &f.cube)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	f.cube.ID = cubeID
	if err := st.Analyses().Save(ctx, "", cubeID, &domain.PositionAnalysis{
		DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{
			BestCubeAction: "Double, Take", CubefulNoDoubleEquity: 0.5, CubefulDoubleTakeEquity: 0.6, CubefulDoublePassEquity: 1,
			CubefulNoDoubleError: -0.1,
		},
		PlayedCubeActions: []string{"No Double"},
	}); err != nil {
		t.Fatalf("Save analysis: %v", err)
	}

	if f.deckID, err = st.Anki().CreateDeck(ctx, "", "Ouvertures", "", domain.AnkiSourceSearch, 0, ""); err != nil {
		t.Fatalf("CreateDeck: %v", err)
	}
	if err := st.Anki().SyncWithPositions(ctx, "", f.deckID, []int64{chkID, cubeID}); err != nil {
		t.Fatalf("SyncWithPositions: %v", err)
	}
	if f.handMade, err = st.Collections().Create(ctx, "", "Cubes", ""); err != nil {
		t.Fatalf("Create collection: %v", err)
	}
	if err := st.Collections().AddPositions(ctx, "", f.handMade, []int64{cubeID}); err != nil {
		t.Fatalf("AddPositions: %v", err)
	}
	// Nothing was ever played: a living collection of every position.
	if f.living, err = storage.CreateCollection(ctx, st, "", "Vivante", "", "s n<1"); err != nil {
		t.Fatalf("CreateCollection: %v", err)
	}
	if err := st.Metadata().Save(ctx, "", map[string]string{"user": "Alice", "description": "Ma base", "secret": "home only"}); err != nil {
		t.Fatalf("Metadata.Save: %v", err)
	}
	return f
}

var stamp = time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)

func export(t *testing.T, f fixture, src apkg.Source, now time.Time) ([]byte, *apkg.Result) {
	t.Helper()
	var buf bytes.Buffer
	res, err := apkg.Export(context.Background(), f.st, "", src, "fr", now, &buf)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	return buf.Bytes(), res
}

// unpacked is an .apkg read back: its collection opened, its media indexed.
type unpacked struct {
	db    *sql.DB
	media map[string][]byte // by file name
}

func unpack(t *testing.T, data []byte) unpacked {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("zip: %v", err)
	}
	files := map[string][]byte{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(rc)
		_ = rc.Close()
		files[f.Name] = b
	}
	col, ok := files["collection.anki2"]
	if !ok {
		t.Fatalf("no collection.anki2 in %v", keys(files))
	}
	var index map[string]string
	if err := json.Unmarshal(files["media"], &index); err != nil {
		t.Fatalf("media index: %v", err)
	}
	u := unpacked{media: map[string][]byte{}}
	for k, name := range index {
		b, ok := files[k]
		if !ok {
			t.Fatalf("media %q indexed as %s is missing", name, k)
		}
		u.media[name] = b
	}
	path := filepath.Join(t.TempDir(), "collection.anki2")
	if err := os.WriteFile(path, col, 0o600); err != nil {
		t.Fatal(err)
	}
	if u.db, err = sql.Open("sqlite", path); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = u.db.Close() })
	return u
}

// column reads a one-column query as strings.
func column(t *testing.T, db *sql.DB, query string, args ...any) []string {
	t.Helper()
	rows, err := db.Query(query, args...)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func keys(m map[string][]byte) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}

func TestPackageFollowsAnkiSchema11(t *testing.T) {
	f := setup(t)
	data, res := export(t, f, apkg.Source{DeckID: f.deckID}, stamp)
	if res.Notes != 2 || res.Name != "Ouvertures" {
		t.Fatalf("result = %+v", res)
	}
	u := unpack(t, data)

	want := map[string][]string{
		"col":    {"id", "crt", "mod", "scm", "ver", "dty", "usn", "ls", "conf", "models", "decks", "dconf", "tags"},
		"notes":  {"id", "guid", "mid", "mod", "usn", "tags", "flds", "sfld", "csum", "flags", "data"},
		"cards":  {"id", "nid", "did", "ord", "mod", "usn", "type", "queue", "due", "ivl", "factor", "reps", "lapses", "left", "odue", "odid", "flags", "data"},
		"revlog": {"id", "cid", "usn", "ease", "ivl", "lastIvl", "factor", "time", "type"},
		"graves": {"usn", "oid", "type"},
	}
	for table, cols := range want {
		got := column(t, u.db, `SELECT name FROM pragma_table_info(?)`, table)
		if strings.Join(got, ",") != strings.Join(cols, ",") {
			t.Errorf("%s columns = %v, want %v", table, got, cols)
		}
	}

	var ver int
	var models, decks string
	if err := u.db.QueryRow(`SELECT ver, models, decks FROM col`).Scan(&ver, &models, &decks); err != nil {
		t.Fatal(err)
	}
	if ver != apkg.SchemaVersion {
		t.Errorf("ver = %d", ver)
	}
	var m map[string]struct {
		Name string `json:"name"`
		Flds []struct {
			Name string `json:"name"`
		} `json:"flds"`
	}
	if err := json.Unmarshal([]byte(models), &m); err != nil {
		t.Fatalf("models: %v", err)
	}
	if len(m) != 1 || len(m["1760000000627"].Flds) != len(apkg.Fields) {
		t.Errorf("models = %s", models)
	}
	if !strings.Contains(decks, `"Ouvertures"`) || !strings.Contains(decks, "Alice") || strings.Contains(decks, "home only") {
		t.Errorf("decks = %s", decks)
	}

	var notes, cards, orphan int
	_ = u.db.QueryRow(`SELECT count(*) FROM notes`).Scan(&notes)
	_ = u.db.QueryRow(`SELECT count(*) FROM cards`).Scan(&cards)
	_ = u.db.QueryRow(`SELECT count(*) FROM cards WHERE nid NOT IN (SELECT id FROM notes) OR did != ?`, stamp.UnixMilli()).Scan(&orphan)
	if notes != 2 || cards != 2 || orphan != 0 {
		t.Errorf("notes %d, cards %d, orphan cards %d", notes, cards, orphan)
	}
}

func TestCardsCarryTheBoardAndTheAnswer(t *testing.T) {
	f := setup(t)
	data, _ := export(t, f, apkg.Source{DeckID: f.deckID}, stamp)
	u := unpack(t, data)

	fields := func(guid string) []string {
		var flds string
		if err := u.db.QueryRow(`SELECT flds FROM notes WHERE guid = ?`, guid).Scan(&flds); err != nil {
			t.Fatalf("note %s: %v", guid, err)
		}
		return strings.Split(flds, "\x1f")
	}
	chk := fields(apkg.NoteGUID(&f.checker))
	if !strings.HasPrefix(chk[0], "XGID=") {
		t.Errorf("position field = %q", chk[0])
	}
	img := strings.TrimSuffix(strings.TrimPrefix(chk[1], `<img src="`), `">`)
	if svg, ok := u.media[img]; !ok || !bytes.HasPrefix(svg, []byte("<svg")) {
		t.Errorf("board %q not in the media (%d bytes)", img, len(svg))
	}
	for _, s := range []string{"3 away – 5 away", "3-1 à jouer"} {
		if !strings.Contains(chk[2], s) {
			t.Errorf("situation %q lacks %q", chk[2], s)
		}
	}
	if chk[3] != "8/5 6/5" || !strings.Contains(chk[4], "+0.160") || !strings.Contains(chk[5], "erreur 0.150") {
		t.Errorf("back = %q", chk[3:])
	}

	cube := fields(apkg.NoteGUID(&f.cube))
	if !strings.Contains(cube[2], "Partie libre") || !strings.Contains(cube[2], "Décision de videau") {
		t.Errorf("cube situation = %q", cube[2])
	}
	if cube[3] != "Double, prise" || !strings.Contains(cube[4], "+0.600") || !strings.Contains(cube[5], "No Double (erreur -0.100)") {
		t.Errorf("cube back = %q", cube[3:])
	}
}

func TestReExportKeepsTheNoteIdentity(t *testing.T) {
	f := setup(t)
	first, _ := export(t, f, apkg.Source{DeckID: f.deckID}, stamp)
	second, _ := export(t, f, apkg.Source{DeckID: f.deckID}, stamp.Add(time.Hour))
	ids := func(data []byte) (string, int64) {
		u := unpack(t, data)
		out := column(t, u.db, `SELECT guid FROM notes ORDER BY guid`)
		var mod int64
		_ = u.db.QueryRow(`SELECT max(mod) FROM notes`).Scan(&mod)
		return strings.Join(out, ","), mod
	}
	a, modA := ids(first)
	b, modB := ids(second)
	if a != b {
		t.Errorf("note identity changed: %s vs %s", a, b)
	}
	if modB <= modA {
		t.Errorf("a later export must be newer, so Anki updates the notes: %d then %d", modA, modB)
	}
}

func TestCollectionsExportHandMadeAndLiving(t *testing.T) {
	f := setup(t)
	_, res := export(t, f, apkg.Source{CollectionID: f.handMade}, stamp)
	if res.Notes != 1 || res.Name != "Cubes" {
		t.Errorf("hand-made = %+v", res)
	}
	// The query is evaluated at export: a position saved after the
	// collection was made is in it.
	extra := domain.InitializePosition()
	extra.Score = [2]int{2, 2}
	if _, err := f.st.Positions().Save(context.Background(), "", &extra); err != nil {
		t.Fatal(err)
	}
	_, res = export(t, f, apkg.Source{CollectionID: f.living}, stamp)
	if res.Notes != 3 || res.Total != 3 || res.Truncated {
		t.Errorf("living = %+v", res)
	}
}

func TestExportWritesNothing(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	before, err := f.st.Anki().DeckStats(ctx, "", f.deckID)
	if err != nil {
		t.Fatal(err)
	}
	export(t, f, apkg.Source{DeckID: f.deckID}, stamp)
	export(t, f, apkg.Source{CollectionID: f.living}, stamp)
	after, err := f.st.Anki().DeckStats(ctx, "", f.deckID)
	if err != nil {
		t.Fatal(err)
	}
	if *before != *after {
		t.Errorf("deck stats moved: %+v then %+v", before, after)
	}
}

func TestSourceIsRequired(t *testing.T) {
	f := setup(t)
	for _, src := range []apkg.Source{{}, {DeckID: f.deckID, CollectionID: f.handMade}} {
		if _, err := apkg.Export(context.Background(), f.st, "", src, "fr", stamp, io.Discard); err == nil {
			t.Errorf("%+v: want an error", src)
		}
	}
	if _, err := apkg.Export(context.Background(), f.st, "", apkg.Source{DeckID: 999}, "fr", stamp, io.Discard); err == nil {
		t.Error("unknown deck: want an error")
	}
}

// TestAnkiImportsThePackage is the oracle: Anki's own importer. It runs only
// when BLUNDERDB_ANKI_PYTHON names a Python with the "anki" package
// (scripts/anki-apkg-check.py): the nightly anki-apkg job sets it.
func TestAnkiImportsThePackage(t *testing.T) {
	python := os.Getenv("BLUNDERDB_ANKI_PYTHON")
	if python == "" {
		t.Skip("BLUNDERDB_ANKI_PYTHON not set")
	}
	f := setup(t)
	crawford := oneAway(t, f, [2]int{domain.Crawford, 4})
	if err := f.st.Anki().SyncWithPositions(context.Background(), "", f.deckID, []int64{f.checker.ID, f.cube.ID, crawford}); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	first, _ := export(t, f, apkg.Source{DeckID: f.deckID}, stamp)
	second, _ := export(t, f, apkg.Source{DeckID: f.deckID}, stamp.Add(time.Hour))
	a, b := filepath.Join(dir, "a.apkg"), filepath.Join(dir, "b.apkg")
	_ = os.WriteFile(a, first, 0o600)
	_ = os.WriteFile(b, second, 0o600)
	out, err := exec.Command(python, "-I", "../../../scripts/anki-apkg-check.py", a, b).CombinedOutput()
	if err != nil {
		t.Fatalf("anki import: %v\n%s", err, out)
	}
	t.Logf("%s", out)
	if !strings.Contains(string(out), "b.apkg: new=0 updated=3") || !strings.Contains(string(out), "notes=3 cards=3 media=3") {
		t.Errorf("anki import:\n%s", out)
	}
}

// oneAway saves a checker position at score, the player on roll first.
func oneAway(t *testing.T, f fixture, score [2]int) int64 {
	t.Helper()
	p := domain.InitializePosition()
	p.Score = score
	p.Board.Points[6] = domain.Point{Checkers: 4, Color: p.PlayerOnRoll}
	id, err := f.st.Positions().Save(context.Background(), "", &p)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// A 1-away player is stored as Crawford (1) or post-Crawford (0): the card
// names which, never "0 away".
func TestOneAwayNamesCrawfordAndPostCrawford(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	ids := []int64{oneAway(t, f, [2]int{domain.Crawford, 4}), oneAway(t, f, [2]int{domain.PostCrawford, 3})}
	if err := f.st.Collections().AddPositions(ctx, "", f.handMade, ids); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ lang, crawford, post, score string }{
		{"fr", "Score\u00a0: 1 away (Crawford) – 4 away", "Score\u00a0: 1 away (post-Crawford) – 3 away", "Videau\u00a0: 1"},
		{"en", "Score: 1 away (Crawford) – 4 away", "Score: 1 away (post-Crawford) – 3 away", "Cube: 1"},
	} {
		var buf bytes.Buffer
		if _, err := apkg.Export(ctx, f.st, "", apkg.Source{CollectionID: f.handMade}, c.lang, stamp, &buf); err != nil {
			t.Fatal(err)
		}
		u := unpack(t, buf.Bytes())
		all := strings.Join(column(t, u.db, `SELECT flds FROM notes`), "\n")
		for _, want := range []string{c.crawford, c.post, c.score} {
			if !strings.Contains(all, want) {
				t.Errorf("%s: no %q in\n%s", c.lang, want, all)
			}
		}
		if strings.Contains(all, "0 away") {
			t.Errorf("%s: a post-Crawford score reads as 0 away", c.lang)
		}
	}
}

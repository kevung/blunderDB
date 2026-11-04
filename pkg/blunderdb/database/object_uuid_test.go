package database

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/changeset"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

var uuidV7 = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

// seedUUIDLibrary fills a library with one object of every family that has a uuid.
func seedUUIDLibrary(t *testing.T, d *Database) {
	t.Helper()
	var ids []int64
	for i := range 3 {
		p := InitializePosition()
		p.Board.Points[3+i].Checkers = 2
		id, err := d.SavePosition(&p)
		if err != nil {
			t.Fatalf("SavePosition: %v", err)
		}
		ids = append(ids, id)
	}
	for _, text := range []string{"first note", "second note"} {
		if err := d.AddComment(ids[0], text); err != nil {
			t.Fatalf("AddComment: %v", err)
		}
	}
	cid, err := d.CreateCollection("Openings", "")
	if err != nil {
		t.Fatalf("CreateCollection: %v", err)
	}
	if _, err := d.CreateLesson("Bearing off", ""); err != nil {
		t.Fatalf("CreateLesson: %v", err)
	}
	deck, err := d.CreateAnkiDeck("Deck", "", "collection", cid, "")
	if err != nil {
		t.Fatalf("CreateAnkiDeck: %v", err)
	}
	for _, stmt := range []string{
		fmt.Sprintf(`INSERT INTO anki_card (deck_id, kind, key, position_id) VALUES (%d, 'position', '', %d)`, deck, ids[1]),
		`INSERT INTO match (canonical_hash, match_hash, player1_name, player2_name) VALUES ('canon-1', 'raw-1', 'A', 'B')`,
		`INSERT INTO match (canonical_hash, match_hash, player1_name, player2_name) VALUES ('canon-2', 'raw-2', 'A', 'B')`,
		`INSERT INTO transcription (format_version, document, label) VALUES ('1', '{}', 'draft')`,
		`INSERT INTO tournament (name, date) VALUES ('Open', '2026-01-01')`,
	} {
		if _, err := d.db.Exec(stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
}

// rewindToBeforeUUIDs puts an open library back as a 2.50.0 one: no uuid
// column, trigger or index, no mark, no journal trigger.
func rewindToBeforeUUIDs(t *testing.T, d *Database) {
	t.Helper()
	for _, sfx := range []string{"_i", "_u"} {
		if _, err := d.db.Exec(`DROP TRIGGER IF EXISTS change_log_analysis` + sfx); err != nil {
			t.Fatal(err)
		}
	}
	// The triggers of the drawing and the derivation name position.uuid.
	for _, tbl := range []string{"position_drawing", "position_derivation"} {
		for _, sfx := range []string{"_i", "_u", "_d"} {
			if _, err := d.db.Exec(`DROP TRIGGER IF EXISTS change_log_` + tbl + sfx); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, f := range sqlshared.ObjectUUIDFamilies {
		for _, stmt := range []string{
			`DROP TRIGGER IF EXISTS object_uuid_` + f.Table,
			`DROP TRIGGER IF EXISTS change_log_` + f.Table + `_i`,
			`DROP TRIGGER IF EXISTS change_log_` + f.Table + `_u`,
			`DROP TRIGGER IF EXISTS change_log_` + f.Table + `_d`,
			`DROP INDEX IF EXISTS idx_` + f.Table + `_uuid`,
			`ALTER TABLE ` + f.Table + ` DROP COLUMN uuid`,
		} {
			if _, err := d.db.Exec(stmt); err != nil {
				t.Fatalf("%s: %v", stmt, err)
			}
		}
	}
	for _, f := range sqlshared.ObjectUUIDFamilies {
		if _, err := d.db.Exec(`DELETE FROM metadata WHERE key = ?`, f.MarkOf()); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := d.db.Exec(`UPDATE metadata SET value = '2.50.0' WHERE key = 'database_version'`); err != nil {
		t.Fatal(err)
	}
}

// uuidsByContent reads every uuid, keyed by the family and a content label
// that does not depend on row ids.
func uuidsByContent(t *testing.T, d *Database) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, c := range []struct{ family, query string }{
		{"position", `SELECT zobrist_hash, uuid FROM position`},
		{"comment", `SELECT p.zobrist_hash || '/' || c.text, c.uuid FROM comment c JOIN position p ON p.id = c.position_id`},
		{"collection", `SELECT name, uuid FROM collection`},
		{"lesson", `SELECT name, uuid FROM lesson`},
		{"match", `SELECT canonical_hash, uuid FROM match`},
		{"transcription", `SELECT label, uuid FROM transcription`},
		{"anki_card", `SELECT kind, uuid FROM anki_card`},
		{"anki_deck", `SELECT name, uuid FROM anki_deck`},
		{"tournament", `SELECT name, uuid FROM tournament`},
	} {
		func() {
			rows, err := d.db.Query(c.query)
			if err != nil {
				t.Fatalf("%s: %v", c.query, err)
			}
			defer rows.Close()
			for rows.Next() {
				var label, uuid any
				if err := rows.Scan(&label, &uuid); err != nil {
					t.Fatal(err)
				}
				u, _ := uuid.(string)
				if !uuidV7.MatchString(u) {
					t.Errorf("%s %v: uuid %q is not a version 7 UUID", c.family, label, u)
				}
				out[fmt.Sprintf("%s/%v", c.family, label)] = u
			}
			if err := rows.Err(); err != nil {
				t.Fatal(err)
			}
		}()
	}
	return out
}

func reopenMigrated(t *testing.T, path string) *Database {
	t.Helper()
	d := NewDatabase()
	if err := d.OpenDatabase(path); err != nil {
		t.Fatalf("open: %v", err)
	}
	closeOnCleanup(t, d)
	return d
}

func uuidLibrary(t *testing.T, name string) (string, *Database) {
	t.Helper()
	path := filepath.Join(tempDir(t), name)
	d := NewDatabase()
	if err := d.SetupDatabase(path); err != nil {
		t.Fatalf("setup: %v", err)
	}
	return path, d
}

func TestObjectUUID_BackfillIsTotalDeterministicAndIdempotent(t *testing.T) {
	t.Parallel()
	pathA, a := uuidLibrary(t, "a.db")
	seedUUIDLibrary(t, a)
	rewindToBeforeUUIDs(t, a)
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	a = reopenMigrated(t, pathA)
	first := uuidsByContent(t, a)
	if len(first) < 8 {
		t.Fatalf("expected one uuid per seeded object, got %d: %v", len(first), first)
	}
	seen := map[string]string{}
	for k, u := range first {
		if prev, dup := seen[u]; dup {
			t.Errorf("%s and %s share %s", k, prev, u)
		}
		seen[u] = k
	}

	var zobrist int64
	if err := a.db.QueryRow(`SELECT zobrist_hash FROM position ORDER BY id LIMIT 1`).Scan(&zobrist); err != nil {
		t.Fatal(err)
	}
	if want := sqlshared.DeriveObjectUUID("position", fmt.Sprint(zobrist), 1); first[fmt.Sprintf("position/%d", zobrist)] != want {
		t.Errorf("position uuid = %s, want the one derived from its Zobrist hash: %s", first[fmt.Sprintf("position/%d", zobrist)], want)
	}

	// Idempotent: another open changes nothing.
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	again := uuidsByContent(t, reopenMigrated(t, pathA))
	for k, u := range first {
		if again[k] != u {
			t.Errorf("%s changed across opens: %s then %s", k, u, again[k])
		}
	}

	// Deterministic: the same content in a library migrated on its own, with
	// other row ids and a note and a collection of its own, converges on the
	// same identifiers, whatever the family.
	pathB, b := uuidLibrary(t, "b.db")
	if _, err := b.db.Exec(`INSERT INTO collection (name) VALUES ('padding') `); err != nil {
		t.Fatal(err)
	}
	seedUUIDLibrary(t, b)
	if _, err := b.db.Exec(`INSERT INTO comment (position_id, text) SELECT id, 'foreign note' FROM position ORDER BY id LIMIT 1`); err != nil {
		t.Fatal(err)
	}
	rewindToBeforeUUIDs(t, b)
	if err := b.Close(); err != nil {
		t.Fatal(err)
	}
	second := uuidsByContent(t, reopenMigrated(t, pathB))
	for k, u := range first {
		if second[k] != u {
			t.Errorf("%s: %s in one library, %s in the other", k, u, second[k])
		}
	}
	if _, ok := second["comment/"+fmt.Sprint(zobrist)+"/foreign note"]; !ok {
		t.Errorf("the foreign note is not in the second library: %v", second)
	}
}

func TestObjectUUID_NewRowsGetAVersion7AndTheTriggerNamesNoFunction(t *testing.T) {
	t.Parallel()
	_, d := uuidLibrary(t, "new.db")
	closeOnCleanup(t, d)
	seedUUIDLibrary(t, d)
	var u string
	if err := d.db.QueryRow(`SELECT uuid FROM collection WHERE name = 'Openings'`).Scan(&u); err != nil {
		t.Fatal(err)
	}
	if !uuidV7.MatchString(u) || strings.HasPrefix(u, "00000000-0000") {
		t.Errorf("a new collection got %q, want a version 7 UUID with a clock timestamp", u)
	}
	rows, err := d.db.Query(`SELECT sql FROM sqlite_master WHERE type = 'trigger' AND name LIKE 'object_uuid_%'`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(s, sqlshared.ObjectUUIDDeriveFunc) {
			t.Errorf("a trigger calls a function a build may not register: %s", s)
		}
		n++
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if n != len(sqlshared.ObjectUUIDFamilies) {
		t.Errorf("%d uuid triggers, want %d", n, len(sqlshared.ObjectUUIDFamilies))
	}
}

// migratedLibrary is a library filled by fill, put back as a 2.50.0 one and
// reopened: the rows it holds are named by the backfill.
func migratedLibrary(t *testing.T, name string, fill func(d *Database)) *Database {
	t.Helper()
	path, d := uuidLibrary(t, name)
	fill(d)
	rewindToBeforeUUIDs(t, d)
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
	return reopenMigrated(t, path)
}

// notesOn fills a library with one position and these notes on it, in order.
func notesOn(t *testing.T, notes ...string) func(d *Database) {
	return func(d *Database) {
		t.Helper()
		p := InitializePosition()
		id, err := d.SavePosition(&p)
		if err != nil {
			t.Fatal(err)
		}
		for _, n := range notes {
			if err := d.AddComment(id, n); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func uuidsWhere(t *testing.T, d *Database, query string, args ...any) []string {
	t.Helper()
	rows, err := d.db.Query(query, args...)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var u string
		if err := rows.Scan(&u); err != nil {
			t.Fatal(err)
		}
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func noteUUIDs(t *testing.T, d *Database, text string) []string {
	return uuidsWhere(t, d, `SELECT uuid FROM comment WHERE text = ? ORDER BY id`, text)
}

// A note is named by what it says and where: another note on the same position
// is another object, the same note at another rank is the same one, and a
// note written twice is two objects.
func TestObjectUUID_ANoteIsNamedByItsContent(t *testing.T) {
	t.Parallel()
	a := migratedLibrary(t, "a.db", notesOn(t, "A"))
	b := migratedLibrary(t, "b.db", notesOn(t, "B"))
	c := migratedLibrary(t, "c.db", notesOn(t, "B", "A"))
	d := migratedLibrary(t, "d.db", notesOn(t, "A", "A"))

	if ua, ub := noteUUIDs(t, a, "A")[0], noteUUIDs(t, b, "B")[0]; ua == ub {
		t.Errorf("note A and note B on the same position share %s", ua)
	}
	if got, want := noteUUIDs(t, c, "A")[0], noteUUIDs(t, a, "A")[0]; got != want {
		t.Errorf("note A at rank 2 is %s, at rank 1 %s: the same note, two identities", got, want)
	}
	if got, want := noteUUIDs(t, c, "B")[0], noteUUIDs(t, b, "B")[0]; got != want {
		t.Errorf("note B at rank 1 is %s, at rank 2 %s", got, want)
	}
	twice := noteUUIDs(t, d, "A")
	if len(twice) != 2 || twice[0] == twice[1] {
		t.Fatalf("a note written twice must hold two identities, got %v", twice)
	}
	if twice[0] != noteUUIDs(t, a, "A")[0] {
		t.Errorf("the first of two identical notes is not the lone note's identity")
	}
}

func transcriptionsWith(rows ...[2]string) func(d *Database) {
	return func(d *Database) {
		for _, r := range rows {
			if _, err := d.db.Exec(`INSERT INTO transcription (format_version, document, label) VALUES ('1', ?, ?)`, r[1], r[0]); err != nil {
				panic(err)
			}
		}
	}
}

func TestObjectUUID_ATranscriptionIsNamedByItsContent(t *testing.T) {
	t.Parallel()
	a := migratedLibrary(t, "a.db", transcriptionsWith([2]string{"draft", `{"a":1}`}))
	b := migratedLibrary(t, "b.db", transcriptionsWith([2]string{"draft", `{"b":2}`}))
	c := migratedLibrary(t, "c.db", transcriptionsWith([2]string{"other", `{}`}, [2]string{"draft", `{"a":1}`}))
	q := `SELECT uuid FROM transcription WHERE document = ?`
	ua, ub := uuidsWhere(t, a, q, `{"a":1}`)[0], uuidsWhere(t, b, q, `{"b":2}`)[0]
	if ua == ub {
		t.Errorf("two transcriptions with one label and two documents share %s", ua)
	}
	if got := uuidsWhere(t, c, q, `{"a":1}`)[0]; got != ua {
		t.Errorf("the same transcription at another rank is %s, then %s", ua, got)
	}
}

// A row with no natural key is not named from its id: two libraries each
// holding "row 1" must not agree on an identity for different objects.
func TestObjectUUID_ARowWithoutNaturalKeyGetsARandomIdentity(t *testing.T) {
	t.Parallel()
	fill := func(d *Database) {
		for _, stmt := range []string{
			`INSERT INTO match (canonical_hash, match_hash, player1_name, player2_name) VALUES ('', '', 'A', 'B')`,
			`INSERT INTO comment (position_id, text) VALUES (NULL, 'orphan')`,
		} {
			if _, err := d.db.Exec(stmt); err != nil {
				panic(err)
			}
		}
	}
	a, b := migratedLibrary(t, "a.db", fill), migratedLibrary(t, "b.db", fill)
	for _, q := range []string{`SELECT uuid FROM match`, `SELECT uuid FROM comment`} {
		ua, ub := uuidsWhere(t, a, q)[0], uuidsWhere(t, b, q)[0]
		if !uuidV7.MatchString(ua) || ua == ub {
			t.Errorf("%s: %s and %s, want distinct version 7 identities", q, ua, ub)
		}
		if strings.HasPrefix(ua, "00000000-0000") {
			t.Errorf("%s: %s is derived, want a drawn identity", q, ua)
		}
	}
}

// A position stored without its hash is repaired before it is named, so its
// identity comes from the hash it ends up with, not from its row id.
func TestObjectUUID_APositionRepairedOnTheOpenIsNamedByItsHash(t *testing.T) {
	t.Parallel()
	d := migratedLibrary(t, "damaged.db", func(d *Database) {
		pos := InitializePosition()
		pos.Board.Points[5].Checkers = 3
		data, err := json.Marshal(pos.NormalizeForStorage())
		if err != nil {
			panic(err)
		}
		if _, err := d.db.Exec(`INSERT INTO position (state, individually_imported) VALUES (?, 1)`, string(data)); err != nil {
			panic(err)
		}
	})
	var zobrist int64
	var u string
	if err := d.db.QueryRow(`SELECT zobrist_hash, uuid FROM position`).Scan(&zobrist, &u); err != nil {
		t.Fatal(err)
	}
	if want := sqlshared.DeriveObjectUUID("position", fmt.Sprint(zobrist), 1); u != want {
		t.Errorf("repaired position uuid = %s, want the one derived from its hash: %s", u, want)
	}
}

// A library at 2.55.0 has its uuids and its journal for the families of 2.51.0;
// the deck and the tournament, added at 2.56.0, are named by their own pass
// when it opens, from the same natural key on every library, and an export
// from the start carries them.
func TestObjectUUID_DeckAndTournamentJoinALibraryAt2_55(t *testing.T) {
	t.Parallel()
	path, d := uuidLibrary(t, "at255.db")
	for _, stmt := range []string{
		`INSERT INTO anki_deck (name) VALUES ('Openings')`,
		`INSERT INTO anki_deck (name) VALUES ('Openings')`,
		`INSERT INTO tournament (name) VALUES ('Open')`,
	} {
		if _, err := d.db.Exec(stmt); err != nil {
			t.Fatal(err)
		}
	}
	for _, table := range []string{"anki_deck", "tournament"} {
		for _, stmt := range []string{
			`DROP TRIGGER IF EXISTS object_uuid_` + table,
			`DROP TRIGGER IF EXISTS change_log_` + table + `_i`,
			`DROP TRIGGER IF EXISTS change_log_` + table + `_u`,
			`DROP TRIGGER IF EXISTS change_log_` + table + `_d`,
			`DROP INDEX IF EXISTS idx_` + table + `_uuid`,
			`ALTER TABLE ` + table + ` DROP COLUMN uuid`,
			`DELETE FROM metadata WHERE key = '` + sqlshared.MarkFor(table) + `'`,
		} {
			if _, err := d.db.Exec(stmt); err != nil {
				t.Fatalf("%s: %v", stmt, err)
			}
		}
	}
	if _, err := d.db.Exec(`UPDATE metadata SET value = '2.55.0' WHERE key = 'database_version'`); err != nil {
		t.Fatal(err)
	}
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
	d = reopenMigrated(t, path)
	if v, err := d.CheckDatabaseVersion(); err != nil || v != DatabaseVersion {
		t.Fatalf("version after the open = %q, %v; want %q", v, err, DatabaseVersion)
	}
	got := uuidsWhere(t, d, `SELECT uuid FROM anki_deck ORDER BY id`)
	want := []string{sqlshared.DeriveObjectUUID("anki_deck", "Openings", 1), sqlshared.DeriveObjectUUID("anki_deck", "Openings", 2)}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("deck uuids = %v, want those derived from the name and the rank: %v", got, want)
	}
	if got := uuidsWhere(t, d, `SELECT uuid FROM tournament`); len(got) != 1 || got[0] != sqlshared.DeriveObjectUUID("tournament", "Open", 1) {
		t.Errorf("tournament uuid = %v", got)
	}
	for _, table := range []string{"anki_deck", "tournament"} {
		var n int
		if err := d.db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE (type = 'index' AND name = ?) OR (type = 'trigger' AND name IN (?, ?, ?, ?))`,
			"idx_"+table+"_uuid", "object_uuid_"+table, "change_log_"+table+"_i", "change_log_"+table+"_u", "change_log_"+table+"_d").Scan(&n); err != nil || n != 5 {
			t.Errorf("%s: %d of its index and 4 triggers exist (%v)", table, n, err)
		}
	}
	set, err := changeset.Export(context.Background(), d.store, "", storage.Cursor{}, 1000)
	if err != nil {
		t.Fatal(err)
	}
	carried := map[string]int{}
	for _, r := range set.Records {
		carried[r.Family]++
	}
	if carried["anki_deck"] != 2 || carried["tournament"] != 1 {
		t.Errorf("an export from the start carries %v, want the two decks and the tournament", carried)
	}
	// A new deck written now is named and journaled by the triggers.
	if _, err := d.db.Exec(`INSERT INTO anki_deck (name) VALUES ('New')`); err != nil {
		t.Fatal(err)
	}
	var u string
	if err := d.db.QueryRow(`SELECT uuid FROM anki_deck WHERE name = 'New'`).Scan(&u); err != nil || !uuidV7.MatchString(u) {
		t.Fatalf("a new deck got %q, %v", u, err)
	}
	var op string
	if err := d.db.QueryRow(`SELECT op FROM change_log WHERE uuid = ?`, u).Scan(&op); err != nil || op != "create" {
		t.Errorf("journal entry of the new deck: %q, %v", op, err)
	}
}

package database

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

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
	} {
		if _, err := d.db.Exec(stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
}

// rewindToBeforeUUIDs puts an open library back as a 2.50.0 one: no uuid
// column, trigger or index, no mark.
func rewindToBeforeUUIDs(t *testing.T, d *Database) {
	t.Helper()
	for _, f := range sqlshared.ObjectUUIDFamilies {
		for _, stmt := range []string{
			`DROP TRIGGER IF EXISTS object_uuid_` + f.Table,
			`DROP INDEX IF EXISTS idx_` + f.Table + `_uuid`,
			`ALTER TABLE ` + f.Table + ` DROP COLUMN uuid`,
		} {
			if _, err := d.db.Exec(stmt); err != nil {
				t.Fatalf("%s: %v", stmt, err)
			}
		}
	}
	for _, stmt := range []string{
		`DELETE FROM metadata WHERE key = '` + sqlshared.ObjectUUIDKey + `'`,
		`UPDATE metadata SET value = '2.50.0' WHERE key = 'database_version'`,
	} {
		if _, err := d.db.Exec(stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
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
	} {
		rows, err := d.db.Query(c.query)
		if err != nil {
			t.Fatalf("%s: %v", c.query, err)
		}
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
		rows.Close()
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
	// other row ids, converges on the same identifiers.
	pathB, b := uuidLibrary(t, "b.db")
	if _, err := b.db.Exec(`INSERT INTO collection (name) VALUES ('padding') `); err != nil {
		t.Fatal(err)
	}
	seedUUIDLibrary(t, b)
	if _, err := b.db.Exec(`DELETE FROM collection WHERE name = 'padding'`); err != nil {
		t.Fatal(err)
	}
	rewindToBeforeUUIDs(t, b)
	if err := b.Close(); err != nil {
		t.Fatal(err)
	}
	second := uuidsByContent(t, reopenMigrated(t, pathB))
	for k, u := range first {
		if strings.HasPrefix(k, "match/") || strings.HasPrefix(k, "collection/") || strings.HasPrefix(k, "position/") {
			if second[k] != u {
				t.Errorf("%s: %s in one library, %s in the other", k, u, second[k])
			}
		}
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
	if n != len(sqlshared.ObjectUUIDFamilies) {
		t.Errorf("%d uuid triggers, want %d", n, len(sqlshared.ObjectUUIDFamilies))
	}
}

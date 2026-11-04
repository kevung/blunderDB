//go:build postgres

package postgres_test

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/kevung/blunderdb/pkg/blunderdb/changeset"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	pg "github.com/kevung/blunderdb/pkg/blunderdb/storage/postgres"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

var pgUUIDv7 = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

// uuidSnapshot reads the uuid of every object of a tenant, keyed by family
// and a content label that does not depend on row ids.
func uuidSnapshot(t *testing.T, conn *pgx.Conn, tenant string) map[string]string {
	t.Helper()
	out := map[string]string{}
	asTenant(t, conn, tenant, func() {
		for _, c := range []struct{ family, query string }{
			{"position", `SELECT zobrist_hash::text, uuid::text FROM position`},
			{"comment", `SELECT p.zobrist_hash::text || '/' || c.text, c.uuid::text FROM comment c JOIN position p ON p.id = c.position_id`},
			{"collection", `SELECT name, uuid::text FROM collection`},
			{"lesson", `SELECT name, uuid::text FROM lesson`},
			{"match", `SELECT canonical_hash, uuid::text FROM match`},
			{"transcription", `SELECT label, uuid::text FROM transcription`},
			{"anki_card", `SELECT kind, uuid::text FROM anki_card`},
		} {
			rows, err := conn.Query(context.Background(), c.query)
			if err != nil {
				t.Fatalf("%s: %v", c.query, err)
			}
			for rows.Next() {
				var label string
				var u *string
				if err := rows.Scan(&label, &u); err != nil {
					t.Fatal(err)
				}
				got := ""
				if u != nil {
					got = *u
				}
				if !pgUUIDv7.MatchString(got) {
					t.Errorf("tenant %s, %s %s: uuid %q is not a version 7 UUID", tenant, c.family, label, got)
				}
				out[c.family+"/"+label] = got
			}
			if err := rows.Err(); err != nil {
				t.Fatal(err)
			}
			rows.Close()
		}
	})
	return out
}

// TestMigrate_054_ObjectUUIDsUnderRLS gives the rows of two tenants, held
// before the uuid columns, their identifiers while the library is FORCEd on a
// role that is neither superuser nor BYPASSRLS: every row is named, the
// identifiers are the ones Go derives, the two tenants converge on shared
// content, a second Migrate changes nothing, and FORCE is back afterwards.
func TestMigrate_054_ObjectUUIDsUnderRLS(t *testing.T) {
	ctx := context.Background()
	s, conn, _ := openAsRLSOwner(t)

	var zobrist []int64
	for _, scope := range []string{"1", "2"} {
		for i := range 3 {
			p := domain.InitializePosition()
			p.Board.Points[3+i].Checkers = 2
			pid, err := s.Positions().Save(ctx, scope, &p)
			if err != nil {
				t.Fatal(err)
			}
			if scope == "1" {
				var z int64
				if err := conn.QueryRow(ctx, `SELECT zobrist_hash FROM position WHERE id = $1`, pid).Scan(&z); err != nil {
					t.Fatal(err)
				}
				zobrist = append(zobrist, z)
			}
			if i == 0 {
				a := domain.PositionAnalysis{AnalysisType: "DoublingCube",
					DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{AnalysisDepth: "4-ply", BestCubeAction: "Double, Take"}}
				if err := s.Analyses().Save(ctx, scope, pid, &a); err != nil {
					t.Fatal(err)
				}
				var deck int64
				if err := conn.QueryRow(ctx, `INSERT INTO anki_deck (tenant_id, name) VALUES ($1::bigint, 'Deck') RETURNING id`, scope).Scan(&deck); err != nil {
					t.Fatal(err)
				}
				if _, err := conn.Exec(ctx, `INSERT INTO anki_card (tenant_id, deck_id, position_id) VALUES ($1::bigint, $2, $3)`, scope, deck, pid); err != nil {
					t.Fatal(err)
				}
				for _, text := range []string{"first note", "second note"} {
					if _, err := conn.Exec(ctx, `INSERT INTO comment (tenant_id, position_id, text) VALUES ($1::bigint, $2, $3)`, scope, pid, text); err != nil {
						t.Fatal(err)
					}
				}
			}
		}
		for _, stmt := range []string{
			`INSERT INTO collection (tenant_id, name) VALUES ($1::bigint, 'Openings')`,
			`INSERT INTO lesson (tenant_id, name) VALUES ($1::bigint, 'Bearing off')`,
			`INSERT INTO match (tenant_id, canonical_hash, match_hash, player1_name, player2_name) VALUES ($1::bigint, 'canon-1', 'raw-1', 'A', 'B')`,
			`INSERT INTO match (tenant_id, canonical_hash, match_hash, player1_name, player2_name) VALUES ($1::bigint, '', '', 'A', 'B')`,
			`INSERT INTO transcription (tenant_id, format_version, document, label) VALUES ($1::bigint, '1', '{}', 'draft')`,
		} {
			if _, err := conn.Exec(ctx, stmt, scope); err != nil {
				t.Fatalf("%s: %v", stmt, err)
			}
		}
	}
	if err := s.ApplyRLS(ctx); err != nil {
		t.Fatalf("ApplyRLS: %v", err)
	}

	// Back to a library from before 054: no identifiers, no mark.
	tables := []string{"position", "comment", "collection", "match", "transcription", "lesson", "anki_card"}
	var rewind []string
	for _, tbl := range tables {
		rewind = append(rewind, `UPDATE `+tbl+` SET uuid = NULL`)
	}
	rewind = append(rewind, `DELETE FROM metadata WHERE key = '`+sqlshared.ObjectUUIDKey+`'`)
	execUnforced(t, conn, tables, rewind)

	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("Migrate under RLS: %v", err)
	}
	assertForced(t, conn, tables...)

	one, two := uuidSnapshot(t, conn, "1"), uuidSnapshot(t, conn, "2")
	if len(one) < 10 {
		t.Fatalf("expected an identifier per seeded object, got %d: %v", len(one), one)
	}
	for k, u := range one {
		if k == "match/" {
			continue
		}
		if two[k] != u {
			t.Errorf("%s: %s for tenant 1, %s for tenant 2; shared content must converge", k, u, two[k])
		}
	}
	for _, z := range zobrist {
		if want, got := sqlshared.DeriveObjectUUID("position", fmt.Sprint(z), 1), one[fmt.Sprintf("position/%d", z)]; got != want {
			t.Errorf("position %d: SQL derived %s, Go derives %s", z, got, want)
		}
	}
	// No natural key: a drawn identifier, not one two tenants would share.
	if a, b := one["match/"], two["match/"]; a == "" || a == b || strings.HasPrefix(a, "00000000-0000") {
		t.Errorf("a match with no hash: %q and %q, want two drawn identifiers", a, b)
	}
	if want, got := sqlshared.DeriveObjectUUID("collection", "Openings", 1), one["collection/Openings"]; got != want {
		t.Errorf("collection: SQL derived %s, Go derives %s", got, want)
	}
	first := fmt.Sprintf("%d/first note", zobrist[0])
	second := fmt.Sprintf("%d/second note", zobrist[0])
	if want := sqlshared.DeriveObjectUUID("comment", fmt.Sprint(zobrist[0])+"|first note", 1); one["comment/"+first] != want {
		t.Errorf("first comment: %s, want the one derived from position and text: %s", one["comment/"+first], want)
	}
	if want := sqlshared.DeriveObjectUUID("comment", fmt.Sprint(zobrist[0])+"|second note", 1); one["comment/"+second] != want {
		t.Errorf("second comment: %s, want the one derived from position and text: %s", one["comment/"+second], want)
	}

	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
	again := uuidSnapshot(t, conn, "1")
	for k, u := range one {
		if again[k] != u {
			t.Errorf("%s changed on the second Migrate: %s then %s", k, u, again[k])
		}
	}

	// A row written afterwards draws a version 7 from the column default.
	var fresh string
	asTenant(t, conn, "1", func() {
		if err := conn.QueryRow(ctx, `INSERT INTO collection (tenant_id, name) VALUES (1, 'Later') RETURNING uuid::text`).Scan(&fresh); err != nil {
			t.Fatal(err)
		}
	})
	if !pgUUIDv7.MatchString(fresh) || fresh[:13] == "00000000-0000" {
		t.Errorf("a new collection got %q, want a version 7 UUID with a clock timestamp", fresh)
	}
}

// TestMigrate_059_DeckAndTournamentUUIDsUnderRLS: a library whose first uuid
// pass is long done still names its decks and tournaments, by a pass of their
// own, on a role that is neither superuser nor BYPASSRLS. The identifiers are
// the ones Go derives, two tenants holding the same names converge, FORCE is
// back, the naming is no change of the journal (an export from the start still
// carries the rows), and a deck written afterwards is journaled by its trigger.
func TestMigrate_059_DeckAndTournamentUUIDsUnderRLS(t *testing.T) {
	ctx := context.Background()
	s, conn, ownerDSN := openAsRLSOwner(t)
	for _, scope := range []string{"1", "2"} {
		for _, stmt := range []string{
			`INSERT INTO anki_deck (tenant_id, name) VALUES ($1::bigint, 'Openings')`,
			`INSERT INTO anki_deck (tenant_id, name) VALUES ($1::bigint, 'Openings')`,
			`INSERT INTO tournament (tenant_id, name) VALUES ($1::bigint, 'Open')`,
		} {
			if _, err := conn.Exec(ctx, stmt, scope); err != nil {
				t.Fatalf("%s: %v", stmt, err)
			}
		}
	}
	if err := s.ApplyRLS(ctx); err != nil {
		t.Fatalf("ApplyRLS: %v", err)
	}
	tables := []string{"anki_deck", "tournament"}
	var rewind []string
	for _, tbl := range tables {
		rewind = append(rewind, `UPDATE `+tbl+` SET uuid = NULL`, `DELETE FROM change_log WHERE family = '`+tbl+`'`,
			`DELETE FROM metadata WHERE key = '`+sqlshared.MarkFor(tbl)+`'`)
	}
	execUnforced(t, conn, tables, rewind)

	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("Migrate under RLS: %v", err)
	}
	assertForced(t, conn, tables...)

	for _, scope := range []string{"1", "2"} {
		var got []string
		asTenant(t, conn, scope, func() {
			rows, err := conn.Query(ctx, `SELECT CAST(uuid AS TEXT) FROM anki_deck ORDER BY id`)
			if err != nil {
				t.Fatal(err)
			}
			defer rows.Close()
			for rows.Next() {
				var u string
				if err := rows.Scan(&u); err != nil {
					t.Fatal(err)
				}
				got = append(got, u)
			}
		})
		want := []string{sqlshared.DeriveObjectUUID("anki_deck", "Openings", 1), sqlshared.DeriveObjectUUID("anki_deck", "Openings", 2)}
		if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
			t.Errorf("tenant %s: deck uuids %v, want %v", scope, got, want)
		}
		var tu string
		asTenant(t, conn, scope, func() {
			if err := conn.QueryRow(ctx, `SELECT CAST(uuid AS TEXT) FROM tournament`).Scan(&tu); err != nil {
				t.Fatal(err)
			}
		})
		if tu != sqlshared.DeriveObjectUUID("tournament", "Open", 1) {
			t.Errorf("tenant %s: tournament uuid %s", scope, tu)
		}
	}

	rls, err := pg.Open(ctx, ownerDSN, &storage.Options{EnableRLS: true})
	if err != nil {
		t.Fatal(err)
	}
	defer rls.Close()
	c := storage.WithTenant(ctx, 1)
	set, err := changeset.Export(c, rls, "1", storage.Cursor{}, 1000)
	if err != nil {
		t.Fatal(err)
	}
	carried := map[string]int{}
	for _, r := range set.Records {
		carried[r.Family]++
	}
	if carried["anki_deck"] != 2 || carried["tournament"] != 1 {
		t.Errorf("an export from the start carries %v", carried)
	}
	id, err := rls.Anki().CreateDeck(c, "1", "New", "", domain.AnkiSourceSearch, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	u, err := rls.ObjectUUIDs().ObjectUUID(c, "1", "anki_deck", id)
	if err != nil || u == "" {
		t.Fatalf("uuid of the new deck: %q, %v", u, err)
	}
	if e, err := rls.ChangeJournal().Entry(c, "1", u); err != nil || e.Op != storage.ChangeCreate || e.Family != "anki_deck" {
		t.Errorf("journal entry of the new deck: %+v, %v", e, err)
	}
}

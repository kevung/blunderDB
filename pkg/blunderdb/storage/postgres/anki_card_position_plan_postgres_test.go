//go:build postgres

package postgres

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

// The cards of a page of positions are read by position: a table of positions
// lists, per row, the decks that hold it, and a cascade from a deleted position
// reaches its cards through the (tenant_id, position_id) foreign key. Without
// an index leading with those two columns, either one reads every card of the
// tenant — and, with no tenant in any anki_card index but the uuid one, the
// cards of every tenant.
func TestAnkiCardsOfPositionsReadByIndex(t *testing.T) {
	ctx := context.Background()
	dsn := purgeTestDB(t)
	s, err := Open(ctx, dsn, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	if err := s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(ctx) })

	// Foreign keys and the journal's triggers are off for the seed: the plan
	// depends on how many cards each tenant holds, not on their positions.
	const perTenant = 10000
	if _, err := conn.Exec(ctx, `SET session_replication_role = replica;
		INSERT INTO anki_deck (tenant_id, name) SELECT t, 'deck ' || d FROM generate_series(1, 3) t, generate_series(1, 4) d;
		INSERT INTO anki_card (tenant_id, deck_id, position_id, kind, key)
		SELECT t, (SELECT min(id) FROM anki_deck WHERE tenant_id = t) + n % 4, n, 'position', n::text
		  FROM generate_series(1, 3) t, generate_series(1, `+strconv.Itoa(perTenant)+`) n;
		SET session_replication_role = DEFAULT;
		ANALYZE anki_deck; ANALYZE anki_card`); err != nil {
		t.Fatal(err)
	}

	const page = 50
	ids := make([]string, page)
	args := make([]any, 0, page+1)
	for i := range ids {
		ids[i] = "$" + strconv.Itoa(i+1)
		args = append(args, int64(i*97+1))
	}
	args = append(args, int64(1))
	stmt := `SELECT DISTINCT ac.position_id, d.id, COALESCE(d.name, '')
		  FROM anki_card ac JOIN anki_deck d ON d.id = ac.deck_id AND d.tenant_id = ac.tenant_id
		 WHERE ac.position_id IN (` + strings.Join(ids, ", ") + `) AND ac.tenant_id = $` + strconv.Itoa(page+1) + `
		 ORDER BY ac.position_id, d.id`
	var out []byte
	if err := conn.QueryRow(ctx, "EXPLAIN (FORMAT JSON) "+stmt, args...).Scan(&out); err != nil {
		t.Fatal(err)
	}
	var plans []struct {
		Plan planNode `json:"Plan"`
	}
	if err := json.Unmarshal(out, &plans); err != nil {
		t.Fatal(err)
	}
	var read []planNode
	var walk func(planNode)
	walk = func(n planNode) {
		if n.Relation == "anki_card" || (n.NodeType == "Bitmap Index Scan" && strings.HasPrefix(n.IndexName, "idx_anki_card")) {
			read = append(read, n)
		}
		for _, c := range n.Plans {
			walk(c)
		}
	}
	walk(plans[0].Plan)
	for _, n := range read {
		if n.IndexName == "idx_anki_card_position" && strings.Contains(n.IndexCond, "tenant_id") && strings.Contains(n.IndexCond, "position_id") {
			return
		}
	}
	t.Errorf("anki_card not read by idx_anki_card_position on (tenant_id, position_id): %+v\n  plan: %s", read, out)
}

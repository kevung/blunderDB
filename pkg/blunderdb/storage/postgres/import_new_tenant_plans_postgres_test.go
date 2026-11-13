//go:build postgres

package postgres

import (
	"encoding/json"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// The writes of a game's moves during the first import of a tenant, under
// row-level security: the read of what scores each move and the bulk insert
// that dates the positions. Estimated at one row per table, the new tenant may
// be reached through an index that names it and nothing else, every row of the
// tenant read per position or per chunk. Each statement runs here as
// createMovesChunk runs it — in a transaction, staged, under the same plan
// settings — and none may walk the tenant.
func TestImportOfANewTenantLookUpByKey(t *testing.T) {
	ctx, _, pool, _, _, importAll, _ := newTenantLibrary(t)
	importAll(1)
	tctx := storage.WithTenant(ctx, 1)

	tx, err := pool.Begin(tctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(tctx) }()

	var game int64
	if err := tx.QueryRow(tctx, `SELECT id FROM game WHERE tenant_id = 1 ORDER BY id DESC LIMIT 1`).Scan(&game); err != nil {
		t.Fatal(err)
	}
	var positionIDs []int64
	rows, err := tx.Query(tctx, `SELECT DISTINCT position_id FROM move WHERE tenant_id = 1 AND game_id = $1 AND position_id IS NOT NULL`, game)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		positionIDs = append(positionIDs, id)
	}
	if rows.Err() != nil || len(positionIDs) == 0 {
		t.Fatalf("positions of game %d: %v", game, rows.Err())
	}

	explain := func(what string, pinned bool, sql string, args ...any) {
		t.Helper()
		var out []byte
		run := func() error {
			return tx.QueryRow(tctx, "EXPLAIN (ANALYZE, FORMAT JSON) "+sql, args...).Scan(&out)
		}
		var err error
		if pinned {
			err = withPinnedPlan(tctx, tx, run)
		} else {
			err = run()
		}
		if err != nil {
			t.Fatalf("%s: %v", what, err)
		}
		var plans []struct {
			Plan executedNode `json:"Plan"`
		}
		if err := json.Unmarshal(out, &plans); err != nil || len(plans) == 0 {
			t.Fatalf("plan of %s: %v", what, err)
		}
		for _, bad := range tenantWalks(plans[0].Plan) {
			t.Errorf("%s: %s", what, bad)
		}
	}

	// Prepared, as the pool's statement cache sends it.
	if _, err := tx.Prepare(tctx, "played_bases", playedDecisionBasesSQL); err != nil {
		t.Fatal(err)
	}
	explain("playedDecisionBases", true, "EXECUTE played_bases('"+int64ArrayLiteral(positionIDs)+"', 1)")

	// The moves of the game written again, staged as the import stages them.
	mrows, err := tx.Query(tctx, `DELETE FROM move WHERE tenant_id = 1 AND game_id = $1 RETURNING
		tenant_id, game_id, move_number, move_type, position_id, player,
		dice_1, dice_2, checker_move, cube_action, luck_mp,
		decision_ms, cube_decision_ms, decision_error_mp, is_close_cube,
		roll_tick_ms, tick_ms`, game)
	if err != nil {
		t.Fatal(err)
	}
	moves, err := pgx.CollectRows(mrows, func(r pgx.CollectableRow) ([]any, error) { return r.Values() })
	if err != nil || len(moves) == 0 {
		t.Fatalf("moves of game %d: %d, %v", game, len(moves), err)
	}
	q, args, err := bulkStatement(tctx, copyTx{tx}, moveBulkInsertSQL, moveBulkTypes, moves)
	if err != nil {
		t.Fatal(err)
	}
	explain("moveBulkInsertSQL", false, q, args...)
}

func int64ArrayLiteral(ids []int64) string {
	b, _ := json.Marshal(ids)
	b[0], b[len(b)-1] = '{', '}'
	return string(b)
}

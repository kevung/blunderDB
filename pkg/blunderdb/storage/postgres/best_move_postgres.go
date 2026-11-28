package postgres

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// bestMoveBatch is how many analysis rows one backfill transaction reads and
// writes: on a cold HDD a batch takes 0.36 s at the median, 0.86 s at p99,
// the longest a writer of those rows may wait. A variable for the tests.
var bestMoveBatch = 2000

// bestMovePendingKey is the metadata row 077 writes: the position_id the
// backfill reached in the first tenant listed, then the tenants left. Each
// batch moves it in its own transaction, so an interrupted pass resumes at
// the batch it lost.
const bestMovePendingKey = "best_move_backfill_pending"

// bestMoveIndexSQL runs outside any transaction: CONCURRENTLY holds SHARE
// UPDATE EXCLUSIVE, which reads and writes pass, where CREATE INDEX would
// stop writes for the whole build.
const bestMoveIndexSQL = `CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_analysis_best_move ON analysis (tenant_id, best_move) INCLUDE (position_id)`

// ensureBestMoveIndex builds idx_analysis_best_move. An interrupted build
// leaves an invalid index that IF NOT EXISTS would keep: it is dropped and
// built again. A role that does not own analysis cannot build it and leaves
// it to the owner's start.
func ensureBestMoveIndex(ctx context.Context, db execer) error {
	var owner bool
	var valid *bool
	if err := db.QueryRow(ctx,
		`SELECT pg_has_role(c.relowner, 'USAGE'),
		        (SELECT i.indisvalid FROM pg_index i WHERE i.indexrelid = to_regclass('idx_analysis_best_move'))
		   FROM pg_class c WHERE c.oid = to_regclass('analysis')`).Scan(&owner, &valid); err != nil {
		return fmt.Errorf("postgres: probe idx_analysis_best_move: %w", err)
	}
	if !owner {
		slog.Warn("analysis best move: this role does not own analysis; the index waits for the owner's start")
		return nil
	}
	if valid != nil && *valid {
		return nil
	}
	if valid != nil {
		if _, err := db.Exec(ctx, `DROP INDEX CONCURRENTLY IF EXISTS idx_analysis_best_move`); err != nil {
			return fmt.Errorf("postgres: drop invalid idx_analysis_best_move: %w", err)
		}
	}
	if _, err := db.Exec(ctx, bestMoveIndexSQL); err != nil {
		return fmt.Errorf("postgres: build idx_analysis_best_move: %w", err)
	}
	return nil
}

// bestMoveCursor is the value of bestMovePendingKey.
type bestMoveCursor struct {
	after   int64
	tenants []int64
}

func parseBestMoveCursor(v string) (bestMoveCursor, error) {
	f := strings.Fields(v)
	if len(f) == 0 {
		return bestMoveCursor{}, errors.New("empty")
	}
	var c bestMoveCursor
	var err error
	if c.after, err = strconv.ParseInt(f[0], 10, 64); err != nil {
		return c, err
	}
	for _, s := range f[1:] {
		t, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return c, err
		}
		c.tenants = append(c.tenants, t)
	}
	return c, nil
}

func (c bestMoveCursor) String() string {
	var b strings.Builder
	b.WriteString(strconv.FormatInt(c.after, 10))
	for _, t := range c.tenants {
		b.WriteByte(' ')
		b.WriteString(strconv.FormatInt(t, 10))
	}
	return b.String()
}

// backfillAnalysisBestMove derives analysis.best_move (engine.BestMove) for
// the rows 077 left NULL, tenant by tenant, each transaction bound to its
// tenant: FORCEd RLS shows those rows without being lifted, so no batch takes
// a lock a reader waits on. In Go because the blob is compressed JSON SQL
// cannot read. A row rewritten meanwhile is left as its writer made it, an
// undecodable blob stays NULL; the search lets both through to Go.
// maxBatches bounds the pass for the tests (0: no bound); done reports that
// no tenant is left.
func backfillAnalysisBestMove(ctx context.Context, conn beginner, maxBatches int) (done bool, err error) {
	written := 0
	defer func() {
		if written > 0 {
			slog.Info("derived the best move of the stored analyses", "analyses", written)
		}
	}()
	for batches := 0; maxBatches == 0 || batches < maxBatches; batches++ {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		n, left, err := retryBestMoveBatch(ctx, func() (int, bool, error) { return bestMoveBatchPass(ctx, conn) })
		if err != nil {
			return false, err
		}
		written += n
		if !left {
			return true, nil
		}
	}
	return false, nil
}

// bestMoveRetries bounds how often one batch is replayed after a deadlock
// (40P01) or a serialization failure (40001): a writer of the same rows
// beside the open service can lose the race, and the batch rolled back whole,
// so replaying it is safe. Variables for the tests.
var (
	bestMoveRetries      = 5
	bestMoveRetryBackoff = 200 * time.Millisecond
)

func retryBestMoveBatch(ctx context.Context, pass func() (int, bool, error)) (int, bool, error) {
	for attempt := 0; ; attempt++ {
		n, left, err := pass()
		if err == nil || attempt >= bestMoveRetries || !isTransientConflict(err) {
			return n, left, err
		}
		slog.Warn("analysis best move: batch lost a conflict; replaying it", "attempt", attempt+1, "error", err)
		select {
		case <-ctx.Done():
			return 0, false, ctx.Err()
		case <-time.After(bestMoveRetryBackoff << attempt):
		}
	}
}

func isTransientConflict(err error) bool {
	var s interface{ SQLState() string }
	if !errors.As(err, &s) {
		return false
	}
	return s.SQLState() == "40P01" || s.SQLState() == "40001"
}

// bestMoveBatchPass derives one batch of the cursor's first tenant: the rows
// whose position_id lies in a range of the (tenant_id, position_id) index,
// read without ORDER BY … LIMIT on the NULL test, which the planner may
// answer by sorting the whole table. The cursor moves in the same
// transaction. It reports the rows written and whether a tenant is left.
func bestMoveBatchPass(ctx context.Context, conn beginner) (int, bool, error) {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return 0, false, fmt.Errorf("postgres: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	var raw string
	switch err := tx.QueryRow(ctx, `SELECT value FROM metadata WHERE key = $1 FOR UPDATE`, bestMovePendingKey).Scan(&raw); {
	case errors.Is(err, pgx.ErrNoRows):
		return 0, false, nil
	case err != nil:
		return 0, false, fmt.Errorf("postgres: read %s: %w", bestMovePendingKey, err)
	}
	cur, err := parseBestMoveCursor(raw)
	if err != nil || len(cur.tenants) == 0 {
		if err != nil {
			slog.Warn("analysis best move: unreadable backfill cursor; the search settles the rest on the blob", "value", raw, "error", err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM metadata WHERE key = $1`, bestMovePendingKey); err != nil {
			return 0, false, fmt.Errorf("postgres: close %s: %w", bestMovePendingKey, err)
		}
		return 0, false, commitBestMove(ctx, tx)
	}
	tenant := cur.tenants[0]
	// The column is derived from the blob, which travels: filling it is no
	// change of the position, and the journal trigger of analysis stays silent.
	if _, err := tx.Exec(ctx, `SELECT set_config('app.tenant_id', $1, true), set_config('blunderdb.applying_at', now()::text, true)`,
		strconv.FormatInt(tenant, 10)); err != nil {
		return 0, false, fmt.Errorf("postgres: bind the best move backfill: %w", err)
	}
	var bound *int64
	if err := tx.QueryRow(ctx,
		`SELECT position_id FROM analysis WHERE tenant_id = $1 AND position_id > $2 ORDER BY position_id OFFSET $3 LIMIT 1`,
		tenant, cur.after, bestMoveBatch-1).Scan(&bound); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return 0, false, fmt.Errorf("postgres: bound the best move batch: %w", err)
	}
	upTo := int64(math.MaxInt64)
	if bound != nil {
		upTo = *bound
	}
	n, err := deriveBestMoveRange(ctx, tx, tenant, cur.after, upTo)
	if err != nil {
		return 0, false, err
	}
	if bound != nil {
		cur.after = upTo
	} else {
		cur = bestMoveCursor{tenants: cur.tenants[1:]}
	}
	if _, err := tx.Exec(ctx, `UPDATE metadata SET value = $2 WHERE key = $1`, bestMovePendingKey, cur.String()); err != nil {
		return 0, false, fmt.Errorf("postgres: move %s: %w", bestMovePendingKey, err)
	}
	return n, true, commitBestMove(ctx, tx)
}

func commitBestMove(ctx context.Context, tx pgx.Tx) error {
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("postgres: commit the best move batch: %w", err)
	}
	return nil
}

// deriveBestMoveRange writes best_move for the NULL rows of tenant whose
// position_id is in (after, upTo]. The xmin read guards the write: a row
// another transaction rewrote since is left to that writer.
func deriveBestMoveRange(ctx context.Context, tx pgx.Tx, tenant, after, upTo int64) (int, error) {
	// The NULL test stays out of WHERE: idx_analysis_best_move answers it,
	// and the planner would read every NULL row of the tenant through it.
	rows, err := tx.Query(ctx,
		`SELECT id, xmin::text, CASE WHEN best_move IS NULL THEN data END FROM analysis
		  WHERE tenant_id = $1 AND position_id > $2 AND position_id <= $3`,
		tenant, after, upTo)
	if err != nil {
		return 0, fmt.Errorf("postgres: read analyses to derive the best move: %w", err)
	}
	raw := make(map[int64][]byte, bestMoveBatch)
	xmins := make(map[int64]string, bestMoveBatch)
	for rows.Next() {
		var id int64
		var x string
		var data []byte
		if err := rows.Scan(&id, &x, &data); err != nil {
			rows.Close()
			return 0, fmt.Errorf("postgres: scan analysis to derive the best move: %w", err)
		}
		if data != nil {
			raw[id], xmins[id] = data, x
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("postgres: read analyses to derive the best move: %w", err)
	}
	if len(raw) == 0 {
		return 0, nil
	}
	decoded, failed := engine.DecodeAnalysesConcurrently(raw)
	ids := make([]int64, 0, len(raw))
	xs := make([]string, 0, len(raw))
	moves := make([]string, 0, len(raw))
	// In id order, which the nested loop over unnest follows: two passes over
	// the same rows lock them in one order instead of deadlocking.
	for _, id := range slices.Sorted(maps.Keys(raw)) {
		if a := decoded[id]; a != nil {
			ids = append(ids, id)
			xs = append(xs, xmins[id])
			moves = append(moves, engine.BestMove(a))
		} else if err := failed[id]; err != nil {
			slog.Warn("analysis best move: undecodable blob", "analysis_id", id, "error", err)
		}
	}
	tag, err := tx.Exec(ctx,
		`UPDATE analysis a SET best_move = v.m
		   FROM unnest($1::bigint[], $2::text[], $3::text[]) AS v(id, x, m)
		  WHERE a.id = v.id AND a.xmin = v.x::xid AND a.best_move IS NULL`,
		ids, xs, moves)
	if err != nil {
		return 0, fmt.Errorf("postgres: write analysis best move: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

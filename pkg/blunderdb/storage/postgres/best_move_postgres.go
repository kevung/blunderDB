package postgres

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// bestMoveBatch is how many analysis rows one backfill round reads and writes.
const bestMoveBatch = 20000

// backfillAnalysisBestMove derives analysis.best_move (engine.BestMove) for
// every row 077 left NULL. In Go because the blob is compressed JSON SQL
// cannot read. Resumable by construction: a written row is no longer NULL; an
// undecodable blob stays NULL, which the search lets through to Go. complete
// reports whether the pass saw every tenant's rows (inUnforcedTx).
func backfillAnalysisBestMove(ctx context.Context, conn beginner) (complete bool, err error) {
	var last int64
	done := 0
	complete = true
	for {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		n := 0
		all, err := inUnforcedTx(ctx, conn, []string{"analysis"}, func(tx pgx.Tx) error {
			var err error
			n, last, err = bestMoveBatchPass(ctx, tx, last)
			return err
		})
		if err != nil {
			return false, err
		}
		complete = complete && all
		if n == 0 {
			break
		}
		done += n
	}
	if done > 0 {
		slog.Info("derived the best move of the stored analyses", "analyses", done)
	}
	return complete, nil
}

// bestMoveBatchPass derives the best move of the next batch of analyses past
// id after, and returns how many rows it read and the last id it read.
func bestMoveBatchPass(ctx context.Context, tx pgx.Tx, after int64) (int, int64, error) {
	// The column is derived from the blob, which travels: filling it is no
	// change of the position, and the journal trigger of analysis stays silent.
	if _, err := tx.Exec(ctx, `SELECT set_config('blunderdb.applying_at', now()::text, true)`); err != nil {
		return 0, after, fmt.Errorf("postgres: silence the journal for the best move: %w", err)
	}
	rows, err := tx.Query(ctx,
		`SELECT id, data FROM analysis WHERE best_move IS NULL AND id > $1 ORDER BY id LIMIT $2`,
		after, bestMoveBatch)
	if err != nil {
		return 0, after, fmt.Errorf("postgres: read analyses to derive the best move: %w", err)
	}
	raw := make(map[int64][]byte, bestMoveBatch)
	var ids []int64
	for rows.Next() {
		var id int64
		var data []byte
		if err := rows.Scan(&id, &data); err != nil {
			rows.Close()
			return 0, after, fmt.Errorf("postgres: scan analysis to derive the best move: %w", err)
		}
		raw[id] = data
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, after, fmt.Errorf("postgres: read analyses to derive the best move: %w", err)
	}
	if len(ids) == 0 {
		return 0, after, nil
	}
	decoded, failed := engine.DecodeAnalysesConcurrently(raw)
	written := make([]int64, 0, len(ids))
	moves := make([]string, 0, len(ids))
	for _, id := range ids {
		if a := decoded[id]; a != nil {
			written = append(written, id)
			moves = append(moves, engine.BestMove(a))
		} else if err := failed[id]; err != nil {
			slog.Warn("analysis best move: undecodable blob", "analysis_id", id, "error", err)
		}
	}
	if _, err := tx.Exec(ctx,
		`UPDATE analysis a SET best_move = v.m
		   FROM unnest($1::bigint[], $2::text[]) AS v(id, m)
		  WHERE a.id = v.id`,
		written, moves); err != nil {
		return 0, after, fmt.Errorf("postgres: write analysis best move: %w", err)
	}
	return len(ids), ids[len(ids)-1], nil
}

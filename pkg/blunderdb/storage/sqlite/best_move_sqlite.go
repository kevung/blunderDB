package sqlite

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

const bestMoveBatch = 5000

// BackfillBestMove derives analysis.best_move (engine.BestMove) for every row
// that holds NULL, the 2.61 step's one pass. Filling it is no change of the
// position: the journal is told it is applying, so the analysis trigger stays
// silent. An undecodable blob stays NULL, which the search settles on the blob.
func (s *analysisStore) BackfillBestMove(ctx context.Context) (int, error) {
	tables, err := s.tableNames(ctx)
	if err != nil {
		return 0, err
	}
	var last int64
	done := 0
	for {
		if err := ctx.Err(); err != nil {
			return done, err
		}
		raw := map[int64][]byte{}
		var ids []int64
		rows, err := s.db.QueryContext(ctx,
			`SELECT id, data FROM analysis WHERE best_move IS NULL AND id > ? ORDER BY id LIMIT ?`, last, bestMoveBatch)
		if err != nil {
			return done, fmt.Errorf("sqlite: read analyses to derive the best move: %w", err)
		}
		for rows.Next() {
			var id int64
			var data []byte
			if err := rows.Scan(&id, &data); err != nil {
				rows.Close()
				return done, fmt.Errorf("sqlite: scan analysis to derive the best move: %w", err)
			}
			raw[id] = data
			ids = append(ids, id)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return done, fmt.Errorf("sqlite: read analyses to derive the best move: %w", err)
		}
		if len(ids) == 0 {
			break
		}
		last = ids[len(ids)-1]
		decoded, failed := engine.DecodeAnalysesConcurrently(raw)
		err = withTx(ctx, s.db, func(tx execer) error {
			if tables["change_log_applying"] {
				at := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
				if _, err := tx.ExecContext(ctx, `INSERT INTO change_log_applying (at) VALUES (?)`, at); err != nil {
					return err
				}
			}
			for _, id := range ids {
				a := decoded[id]
				if a == nil {
					slog.Warn("analysis best move: undecodable blob", "analysis_id", id, "error", failed[id])
					continue
				}
				if _, err := tx.ExecContext(ctx, `UPDATE analysis SET best_move = ? WHERE id = ?`, engine.BestMove(a), id); err != nil {
					return err
				}
				done++
			}
			if tables["change_log_applying"] {
				if _, err := tx.ExecContext(ctx, `DELETE FROM change_log_applying`); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return done, fmt.Errorf("sqlite: write analysis best move: %w", err)
		}
	}
	if done > 0 {
		slog.Info("derived the best move of the stored analyses", "analyses", done)
	}
	return done, nil
}

func (s *analysisStore) tableNames(ctx context.Context) (map[string]bool, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT name FROM sqlite_master WHERE type = 'table'`)
	if err != nil {
		return nil, fmt.Errorf("sqlite: list tables: %w", err)
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		out[n] = true
	}
	return out, rows.Err()
}

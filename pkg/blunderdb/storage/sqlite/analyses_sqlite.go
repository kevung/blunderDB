package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"encoding/json"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"log/slog"
)

type analysisStore struct{ db execer }

var _ storage.AnalysisStore = (*analysisStore)(nil)

const analysisInsertSQL = `INSERT INTO analysis (
	position_id, data,
	best_cube_action, cube_error, best_move_equity_error,
	player1_win_rate, player1_gammon_rate, player1_backgammon_rate,
	player2_win_rate, player2_gammon_rate, player2_backgammon_rate,
	is_forced, is_close_cube
) VALUES (?,?, ?,?,?, ?,?,?, ?,?,?, ?,?)`

// analysisUpsertSQL is analysisInsertSQL with the conflict resolved in the
// same statement. It replaced a SELECT followed by an INSERT or an UPDATE:
// two concurrent saves both read "no row" and both inserted, and Load — a
// plain `WHERE position_id = ?` — then returned whichever row came first.
// The conflict target names the UNIQUE index idx_analysis_position, which the
// fresh schema declares and the 2.18.0 migration installs on existing files.
const analysisUpsertSQL = analysisInsertSQL + `
ON CONFLICT(position_id) DO UPDATE SET
	data=excluded.data,
	best_cube_action=excluded.best_cube_action,
	cube_error=excluded.cube_error,
	best_move_equity_error=excluded.best_move_equity_error,
	player1_win_rate=excluded.player1_win_rate,
	player1_gammon_rate=excluded.player1_gammon_rate,
	player1_backgammon_rate=excluded.player1_backgammon_rate,
	player2_win_rate=excluded.player2_win_rate,
	player2_gammon_rate=excluded.player2_gammon_rate,
	player2_backgammon_rate=excluded.player2_backgammon_rate,
	is_forced=excluded.is_forced,
	is_close_cube=excluded.is_close_cube`

// Save stores (or replaces) the analysis for positionID. The analysis JSON is
// zlib-compressed and the denormalised scalar columns are derived. Higher-level
// merge logic (combining XG and GnuBG analyses) stays in the Database wrapper,
// which loads, merges, then calls Save.
func (s *analysisStore) Save(ctx context.Context, scope string, positionID int64, a *domain.PositionAnalysis) error {
	a.PositionID = int(positionID)
	playedMove := firstOf(a.PlayedMoves)
	playedCubeAction := firstOf(a.PlayedCubeActions)

	engine.RoundAnalysisForStorage(a)
	data, err := engine.EncodeAnalysisForStorage(a)
	if err != nil {
		return fmt.Errorf("sqlite: encode analysis: %w", err)
	}
	c := engine.PopulateAnalysisColumns(a, playedMove, playedCubeAction)

	// The analysis row and the position flag it implies are one write: a
	// caller that already holds a transaction writes inside it, a caller that
	// does not gets one of its own (withTx).
	return withTx(ctx, s.db, func(tx execer) error {
		if _, err := tx.ExecContext(ctx, analysisUpsertSQL,
			positionID, data,
			c.BestCubeAction, c.CubeError, c.BestMoveEquityError,
			c.Player1WinRate, c.Player1GammonRate, c.Player1BackgammonRate,
			c.Player2WinRate, c.Player2GammonRate, c.Player2BackgammonRate,
			c.IsForced, c.IsCloseCube); err != nil {
			return fmt.Errorf("sqlite: save analysis: %w", err)
		}

		// Flag the position as a take/pass cube response if any played cube action is
		// a response (only ever set to 1; OR semantics for a deduped position).
		for _, action := range a.PlayedCubeActions {
			if engine.IsResponseCubeAction(action) {
				if _, err := tx.ExecContext(ctx,
					`UPDATE position SET is_cube_response = 1 WHERE id = ?`, positionID); err != nil {
					return fmt.Errorf("sqlite: flag cube response: %w", err)
				}
				break
			}
		}
		return nil
	})
}

// Load returns the decoded analysis for positionID, or ErrNotFound. The
// compressed payload is transparently decompressed.
func (s *analysisStore) Load(ctx context.Context, scope string, positionID int64) (*domain.PositionAnalysis, error) {
	var data []byte
	err := s.db.QueryRowContext(ctx,
		`SELECT data FROM analysis WHERE position_id = ?`, positionID).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("sqlite: load analysis for position %d: %w", positionID, storage.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("sqlite: load analysis for position %d: %w", positionID, err)
	}
	a, err := engine.DecodeAnalysisFromStorage(data)
	if err != nil {
		return nil, fmt.Errorf("sqlite: decode analysis for position %d: %w", positionID, err)
	}
	return &a, nil
}

// LoadMany — see storage.AnalysisStore. The payloads are read in one
// statement per batch and decoded in parallel (engine.DecodeAnalysesConcurrently).
func (s *analysisStore) LoadMany(ctx context.Context, scope string, ids []int64) (map[int64]*domain.PositionAnalysis, error) {
	raw := make(map[int64][]byte, len(ids))
	err := forEachIn(ctx, s.db, ids, `SELECT position_id, data FROM analysis WHERE position_id IN `, ``, func(rows *sql.Rows) error {
		var id int64
		var data []byte
		if err := rows.Scan(&id, &data); err != nil {
			return err
		}
		raw[id] = data
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("sqlite: load analyses: %w", err)
	}
	decoded, failed := engine.DecodeAnalysesConcurrently(raw)
	for id, err := range failed {
		slog.Warn("decoding stored analysis", "positionID", id, "err", err)
	}
	return decoded, nil
}

// SaveAnalysisUncompressed writes a's analysis row for positionID with its
// JSON payload left uncompressed, denormalised columns derived as Save would.
// It exists for one writer: an exported database is read by whatever version
// of blunderDB the recipient runs, and versions before 2.3.0 know only plain
// JSON — a live database compresses (Save), a file handed to someone else
// does not. The row must not exist yet (an export writes each position once).
func SaveAnalysisUncompressed(ctx context.Context, tx *sql.Tx, positionID int64, a *domain.PositionAnalysis) error {
	a.PositionID = int(positionID)
	engine.RoundAnalysisForStorage(a)
	data, err := json.Marshal(a)
	if err != nil {
		return fmt.Errorf("sqlite: encode analysis: %w", err)
	}
	c := engine.PopulateAnalysisColumns(a, firstOf(a.PlayedMoves), firstOf(a.PlayedCubeActions))
	if _, err := tx.ExecContext(ctx, analysisInsertSQL,
		positionID, data,
		c.BestCubeAction, c.CubeError, c.BestMoveEquityError,
		c.Player1WinRate, c.Player1GammonRate, c.Player1BackgammonRate,
		c.Player2WinRate, c.Player2GammonRate, c.Player2BackgammonRate,
		c.IsForced, c.IsCloseCube); err != nil {
		return fmt.Errorf("sqlite: save analysis: %w", err)
	}
	return nil
}

// Delete removes the analysis for positionID.
func (s *analysisStore) Delete(ctx context.Context, scope string, positionID int64) error {
	if _, err := s.db.ExecContext(ctx,
		`DELETE FROM analysis WHERE position_id = ?`, positionID); err != nil {
		return fmt.Errorf("sqlite: delete analysis for position %d: %w", positionID, err)
	}
	return nil
}

func firstOf(s []string) string {
	if len(s) > 0 {
		return s[0]
	}
	return ""
}

// RepairDenormalisedColumns — see storage.AnalysisStore.
//
// Rows are read, decoded and rewritten one at a time rather than loaded whole:
// a real database holds tens of thousands of analyses, and the point of a repair
// is to run on the biggest ones. Only rows whose columns actually change are
// written, so a second run reports 0 and touches nothing — the count is the
// answer to "was anything wrong?", not merely to "did it run?".
func (s *analysisStore) RepairDenormalisedColumns(ctx context.Context, _ string) (int, error) {
	type row struct {
		id                                     int64
		data                                   []byte
		bestCube                               sql.NullString
		cubeErr, bestMoveErr, forced, closeCub sql.NullInt64
	}
	var all []row
	if err := func() error {
		rows, err := s.db.QueryContext(ctx,
			`SELECT id, data, best_cube_action, cube_error, best_move_equity_error, is_forced, is_close_cube
			 FROM analysis ORDER BY id`)
		if err != nil {
			return fmt.Errorf("sqlite: repair: read analyses: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var r row
			if err := rows.Scan(&r.id, &r.data, &r.bestCube, &r.cubeErr, &r.bestMoveErr, &r.forced, &r.closeCub); err != nil {
				return fmt.Errorf("sqlite: repair: scan: %w", err)
			}
			all = append(all, r)
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("sqlite: repair: iterate: %w", err)
		}
		return nil
	}(); err != nil {
		return 0, err
	}

	repaired := 0
	for _, r := range all {
		a, err := engine.DecodeAnalysisFromStorage(r.data)
		if err != nil {
			// An unreadable blob is LEFT ALONE, never zeroed: the columns we have
			// may be wrong, but blanking them would lose the only information
			// left about that position.
			continue
		}
		c := engine.PopulateAnalysisColumns(&a, firstOf(a.PlayedMoves), firstOf(a.PlayedCubeActions))
		if c.BestCubeAction == r.bestCube.String &&
			c.CubeError == r.cubeErr.Int64 &&
			c.BestMoveEquityError == r.bestMoveErr.Int64 &&
			c.IsForced == r.forced.Int64 &&
			c.IsCloseCube == r.closeCub.Int64 {
			continue
		}
		if _, err := s.db.ExecContext(ctx,
			`UPDATE analysis SET best_cube_action=?, cube_error=?, best_move_equity_error=?,
			 is_forced=?, is_close_cube=? WHERE id=?`,
			c.BestCubeAction, c.CubeError, c.BestMoveEquityError, c.IsForced, c.IsCloseCube, r.id); err != nil {
			return repaired, fmt.Errorf("sqlite: repair: update %d: %w", r.id, err)
		}
		repaired++
	}
	return repaired, nil
}

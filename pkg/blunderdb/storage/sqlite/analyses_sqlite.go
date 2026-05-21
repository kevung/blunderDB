package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
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

const analysisUpdateSQL = `UPDATE analysis SET
	data=?, best_cube_action=?, cube_error=?, best_move_equity_error=?,
	player1_win_rate=?, player1_gammon_rate=?, player1_backgammon_rate=?,
	player2_win_rate=?, player2_gammon_rate=?, player2_backgammon_rate=?,
	is_forced=?, is_close_cube=?
	WHERE id=?`

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

	var existingID int64
	err = s.db.QueryRowContext(ctx,
		`SELECT id FROM analysis WHERE position_id = ?`, positionID).Scan(&existingID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		_, err = s.db.ExecContext(ctx, analysisInsertSQL,
			positionID, data,
			c.BestCubeAction, c.CubeError, c.BestMoveEquityError,
			c.Player1WinRate, c.Player1GammonRate, c.Player1BackgammonRate,
			c.Player2WinRate, c.Player2GammonRate, c.Player2BackgammonRate,
			c.IsForced, c.IsCloseCube)
	case err != nil:
		return fmt.Errorf("sqlite: save analysis lookup: %w", err)
	default:
		_, err = s.db.ExecContext(ctx, analysisUpdateSQL,
			data, c.BestCubeAction, c.CubeError, c.BestMoveEquityError,
			c.Player1WinRate, c.Player1GammonRate, c.Player1BackgammonRate,
			c.Player2WinRate, c.Player2GammonRate, c.Player2BackgammonRate,
			c.IsForced, c.IsCloseCube, existingID)
	}
	if err != nil {
		return fmt.Errorf("sqlite: save analysis: %w", err)
	}
	return nil
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

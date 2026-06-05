package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type analysisStore struct{ db execer }

var _ storage.AnalysisStore = (*analysisStore)(nil)

const analysisInsertSQL = `INSERT INTO analysis (
	tenant_id, position_id, data,
	best_cube_action, cube_error, best_move_equity_error,
	player1_win_rate, player1_gammon_rate, player1_backgammon_rate,
	player2_win_rate, player2_gammon_rate, player2_backgammon_rate,
	is_forced, is_close_cube
) VALUES ($1,$2,$3, $4,$5,$6, $7,$8,$9, $10,$11,$12, $13,$14)`

const analysisUpdateSQL = `UPDATE analysis SET
	data=$1, best_cube_action=$2, cube_error=$3, best_move_equity_error=$4,
	player1_win_rate=$5, player1_gammon_rate=$6, player1_backgammon_rate=$7,
	player2_win_rate=$8, player2_gammon_rate=$9, player2_backgammon_rate=$10,
	is_forced=$11, is_close_cube=$12
	WHERE id=$13`

// Save stores (or replaces) the analysis for positionID. The analysis JSON is
// zlib-compressed into the BYTEA data column and the denormalised scalar
// columns are derived. Higher-level merge logic (combining XG and GnuBG
// analyses) stays in the caller, which loads, merges, then calls Save.
func (s *analysisStore) Save(ctx context.Context, scope string, positionID int64, a *domain.PositionAnalysis) error {
	tenant := tenantID(scope)
	a.PositionID = int(positionID)
	playedMove := firstOf(a.PlayedMoves)
	playedCubeAction := firstOf(a.PlayedCubeActions)

	engine.RoundAnalysisForStorage(a)
	data, err := engine.EncodeAnalysisForStorage(a)
	if err != nil {
		return fmt.Errorf("postgres: encode analysis: %w", err)
	}
	c := engine.PopulateAnalysisColumns(a, playedMove, playedCubeAction)

	var existingID int64
	err = s.db.QueryRow(ctx,
		`SELECT id FROM analysis WHERE position_id = $1 AND tenant_id = $2`,
		positionID, tenant).Scan(&existingID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		_, err = s.db.Exec(ctx, analysisInsertSQL,
			tenant, positionID, data,
			c.BestCubeAction, c.CubeError, c.BestMoveEquityError,
			c.Player1WinRate, c.Player1GammonRate, c.Player1BackgammonRate,
			c.Player2WinRate, c.Player2GammonRate, c.Player2BackgammonRate,
			c.IsForced != 0, c.IsCloseCube != 0)
	case err != nil:
		return fmt.Errorf("postgres: save analysis lookup: %w", err)
	default:
		_, err = s.db.Exec(ctx, analysisUpdateSQL,
			data, c.BestCubeAction, c.CubeError, c.BestMoveEquityError,
			c.Player1WinRate, c.Player1GammonRate, c.Player1BackgammonRate,
			c.Player2WinRate, c.Player2GammonRate, c.Player2BackgammonRate,
			c.IsForced != 0, c.IsCloseCube != 0, existingID)
	}
	if err != nil {
		return fmt.Errorf("postgres: save analysis: %w", err)
	}

	// Flag the position as a take/pass cube response if any played cube action is
	// a response (only ever set to TRUE; OR semantics for a deduped position).
	for _, action := range a.PlayedCubeActions {
		if engine.IsResponseCubeAction(action) {
			if _, err := s.db.Exec(ctx,
				`UPDATE position SET is_cube_response = TRUE WHERE id = $1 AND tenant_id = $2`,
				positionID, tenant); err != nil {
				return fmt.Errorf("postgres: flag cube response: %w", err)
			}
			break
		}
	}
	return nil
}

// Load returns the decoded analysis for positionID, or ErrNotFound. The
// compressed payload is transparently decompressed.
func (s *analysisStore) Load(ctx context.Context, scope string, positionID int64) (*domain.PositionAnalysis, error) {
	var data []byte
	err := s.db.QueryRow(ctx,
		`SELECT data FROM analysis WHERE position_id = $1 AND tenant_id = $2`,
		positionID, tenantID(scope)).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("postgres: load analysis for position %d: %w", positionID, storage.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("postgres: load analysis for position %d: %w", positionID, err)
	}
	a, err := engine.DecodeAnalysisFromStorage(data)
	if err != nil {
		return nil, fmt.Errorf("postgres: decode analysis for position %d: %w", positionID, err)
	}
	return &a, nil
}

// Delete removes the analysis for positionID.
func (s *analysisStore) Delete(ctx context.Context, scope string, positionID int64) error {
	if _, err := s.db.Exec(ctx,
		`DELETE FROM analysis WHERE position_id = $1 AND tenant_id = $2`,
		positionID, tenantID(scope)); err != nil {
		return fmt.Errorf("postgres: delete analysis for position %d: %w", positionID, err)
	}
	return nil
}

func firstOf(s []string) string {
	if len(s) > 0 {
		return s[0]
	}
	return ""
}

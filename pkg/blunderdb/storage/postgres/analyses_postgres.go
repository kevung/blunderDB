package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"log/slog"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type analysisStore struct{ db execer }

var _ storage.AnalysisStore = (*analysisStore)(nil)

// repairPageSize bounds how many analysis rows RepairDenormalisedColumns
// holds in memory at once (id keyset pagination) — see the SQLite backend's
// identical constant, `sqlite.repairPageSize` (unexported in both packages,
// so restated rather than shared).
const repairPageSize = 500

const analysisInsertSQL = `INSERT INTO analysis (
	tenant_id, position_id, data,
	best_cube_action, cube_error, best_move_equity_error,
	player1_win_rate, player1_gammon_rate, player1_backgammon_rate,
	player2_win_rate, player2_gammon_rate, player2_backgammon_rate,
	is_forced, is_close_cube
) VALUES ($1,$2,$3, $4,$5,$6, $7,$8,$9, $10,$11,$12, $13,$14)`

// analysisUpsertSQL is analysisInsertSQL with the conflict resolved in the
// same statement. It replaced a SELECT followed by an INSERT or an UPDATE:
// two concurrent saves both read "no row" and both inserted, and Load — a
// plain `WHERE position_id = $1` — then returned whichever row came first.
// The conflict target names the UNIQUE index idx_analysis_position; position
// ids are unique across tenants (one BIGSERIAL sequence), so the index needs
// no tenant_id and the target is position_id alone.
const analysisUpsertSQL = analysisInsertSQL + `
ON CONFLICT (position_id) DO UPDATE SET
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
// compressed (zstd, see engine.CompressAnalysisData) into the BYTEA data column and the denormalised scalar
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

	// The analysis row and the position flag it implies are one write.
	return withTx(ctx, s.db, func(tx execer) error {
		if _, err := tx.Exec(ctx, analysisUpsertSQL,
			tenant, positionID, data,
			c.BestCubeAction, c.CubeError, c.BestMoveEquityError,
			c.Player1WinRate, c.Player1GammonRate, c.Player1BackgammonRate,
			c.Player2WinRate, c.Player2GammonRate, c.Player2BackgammonRate,
			c.IsForced != 0, c.IsCloseCube != 0); err != nil {
			return fmt.Errorf("postgres: save analysis: %w", err)
		}

		// Flag the position as a take/pass cube response if any played cube action is
		// a response (only ever set to TRUE; OR semantics for a deduped position).
		for _, action := range a.PlayedCubeActions {
			if engine.IsResponseCubeAction(action) {
				if _, err := tx.Exec(ctx,
					`UPDATE position SET is_cube_response = TRUE WHERE id = $1 AND tenant_id = $2`,
					positionID, tenant); err != nil {
					return fmt.Errorf("postgres: flag cube response: %w", err)
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

// LoadMany — see storage.AnalysisStore. One query, decoded in parallel
// (engine.DecodeAnalysesConcurrently).
func (s *analysisStore) LoadMany(ctx context.Context, scope string, ids []int64) (map[int64]*domain.PositionAnalysis, error) {
	raw := make(map[int64][]byte, len(ids))
	if len(ids) > 0 {
		rows, err := s.db.Query(ctx,
			`SELECT position_id, data FROM analysis WHERE position_id = ANY($1) AND tenant_id = $2`,
			ids, tenantID(scope))
		if err != nil {
			return nil, fmt.Errorf("postgres: load analyses: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var id int64
			var data []byte
			if err := rows.Scan(&id, &data); err != nil {
				return nil, fmt.Errorf("postgres: load analyses: %w", err)
			}
			raw[id] = data
		}
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("postgres: load analyses: %w", err)
		}
	}
	decoded, failed := engine.DecodeAnalysesConcurrently(raw)
	for id, err := range failed {
		slog.Warn("decoding stored analysis", "positionID", id, "err", err)
	}
	return decoded, nil
}

// RepairDenormalisedColumns — see storage.AnalysisStore. Scoped to the tenant,
// unlike the SQLite backend (one database, one library): a repair must never
// reach beyond the tenant it was asked for.
//
// Rows are read, decoded and rewritten a page at a time (id keyset
// pagination, repairPageSize — see the SQLite backend's counterpart), never
// loaded whole: a real database holds tens of thousands of analyses, and the
// point of a repair is to run on the biggest ones (B.11, #179).
func (s *analysisStore) RepairDenormalisedColumns(ctx context.Context, scope string) (int, error) {
	tid := tenantID(scope)
	type row struct {
		id                   int64
		data                 []byte
		bestCube             string
		cubeErr, bestMoveErr int64
		forced, closeCub     bool
	}
	repaired := 0
	var lastID int64
	for {
		var page []row
		if err := func() error {
			rows, err := s.db.Query(ctx,
				`SELECT id, data, COALESCE(best_cube_action,''), COALESCE(cube_error,0),
				        COALESCE(best_move_equity_error,0), is_forced, is_close_cube
				 FROM analysis WHERE tenant_id = $1 AND id > $2 ORDER BY id LIMIT $3`,
				tid, lastID, repairPageSize)
			if err != nil {
				return fmt.Errorf("postgres: repair: read analyses: %w", err)
			}
			defer rows.Close()
			for rows.Next() {
				var r row
				if err := rows.Scan(&r.id, &r.data, &r.bestCube, &r.cubeErr, &r.bestMoveErr, &r.forced, &r.closeCub); err != nil {
					return fmt.Errorf("postgres: repair: scan: %w", err)
				}
				page = append(page, r)
			}
			return rows.Err()
		}(); err != nil {
			return repaired, err
		}
		if len(page) == 0 {
			return repaired, nil
		}
		lastID = page[len(page)-1].id

		for _, r := range page {
			a, err := engine.DecodeAnalysisFromStorage(r.data)
			if err != nil {
				continue // blob illisible : on laisse en place plutôt que de l'écraser
			}
			c := engine.PopulateAnalysisColumns(&a, firstOf(a.PlayedMoves), firstOf(a.PlayedCubeActions))
			if c.BestCubeAction == r.bestCube && c.CubeError == r.cubeErr &&
				c.BestMoveEquityError == r.bestMoveErr &&
				(c.IsForced == 1) == r.forced && (c.IsCloseCube == 1) == r.closeCub {
				continue
			}
			if _, err := s.db.Exec(ctx,
				`UPDATE analysis SET best_cube_action=$1, cube_error=$2, best_move_equity_error=$3,
				 is_forced=$4, is_close_cube=$5 WHERE id=$6 AND tenant_id=$7`,
				c.BestCubeAction, c.CubeError, c.BestMoveEquityError,
				c.IsForced == 1, c.IsCloseCube == 1, r.id, tid); err != nil {
				return repaired, fmt.Errorf("postgres: repair: update %d: %w", r.id, err)
			}
			repaired++
		}
	}
}

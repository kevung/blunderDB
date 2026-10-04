package sqlite

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"iter"

	"encoding/json"
	"log/slog"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

type analysisStore struct{ db execer }

var _ storage.AnalysisStore = (*analysisStore)(nil)

const analysisInsertSQL = `INSERT INTO analysis (
	position_id, data,
	best_cube_action, cube_error, best_move_equity_error,
	player1_win_rate, player1_gammon_rate, player1_backgammon_rate,
	player2_win_rate, player2_gammon_rate, player2_backgammon_rate,
	is_forced, is_close_cube,
	analysis_engine, analysis_depth, creation_date
) VALUES (?,?, ?,?,?, ?,?,?, ?,?,?, ?,?, ?,?,?)`

// analysisUpsertSQL is analysisInsertSQL with the conflict resolved in the
// same statement, so concurrent saves cannot insert two rows. The target is
// the UNIQUE index idx_analysis_position.
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
	is_close_cube=excluded.is_close_cube,
	analysis_engine=excluded.analysis_engine,
	analysis_depth=excluded.analysis_depth,
	creation_date=excluded.creation_date`

// Save stores (or replaces) the analysis for positionID. The analysis JSON is
// compressed (zstd, see engine.CompressAnalysisData) and the denormalised scalar columns are derived. Higher-level
// merge logic (combining XG and GnuBG analyses) stays in the Database wrapper,
// which loads, merges, then calls Save.
func (s *analysisStore) Save(ctx context.Context, scope string, positionID int64, a *domain.PositionAnalysis) error {
	c, err := s.prepare(ctx, positionID, a, nil)
	if err != nil {
		return err
	}
	return s.write(ctx, positionID, a, c, nil)
}

const analysisMergeSelectSQL = `SELECT data, best_cube_action, cube_error, best_move_equity_error, is_forced, is_close_cube,
	analysis_engine, analysis_depth
	FROM analysis WHERE position_id = ?`

const cubeResponseSQL = `UPDATE position SET is_cube_response = 1 WHERE id = ?`

// Merge — see storage.AnalysisStore.
func (s *analysisStore) Merge(ctx context.Context, scope string, positionID int64, played *storage.PlayedActions, merge func(*domain.PositionAnalysis) *domain.PositionAnalysis) (bool, error) {
	var (
		data      []byte
		stored    storedPlayedColumns
		storedEng sql.NullString
		storedDep sql.NullInt64
	)
	err := s.db.QueryRowContext(ctx, analysisMergeSelectSQL, positionID).
		Scan(&data, &stored.bestCube, &stored.cubeErr, &stored.bestMoveErr, &stored.forced, &stored.closeCube, &storedEng, &storedDep)
	found := err == nil
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, fmt.Errorf("sqlite: load analysis for position %d: %w", positionID, err)
	}
	var existing *domain.PositionAnalysis
	var oldKey []byte
	if found {
		a, err := engine.DecodeAnalysisFromStorage(data)
		if err != nil {
			return false, fmt.Errorf("sqlite: decode analysis for position %d: %w", positionID, err)
		}
		// The key is taken before merge runs: merge may share and mutate
		// existing's slices.
		if oldKey, err = engine.AnalysisContentKey(&a); err != nil {
			return false, fmt.Errorf("sqlite: key analysis for position %d: %w", positionID, err)
		}
		existing = &a
	}
	merged := merge(existing)
	if merged == nil {
		return false, nil
	}
	c, err := s.prepare(ctx, positionID, merged, played)
	if err != nil {
		return false, err
	}
	if found && stored.equal(c) {
		newKey, err := engine.AnalysisContentKey(merged)
		if err != nil {
			return false, fmt.Errorf("sqlite: key analysis for position %d: %w", positionID, err)
		}
		if bytes.Equal(oldKey, newKey) {
			return false, nil
		}
	}
	// The columns just read decide the match_stats invalidation: the import
	// path pays no second read for it.
	statsChanged := !found || !stored.cubeErr.Valid || !stored.bestMoveErr.Valid ||
		stored.cubeErr.Int64 != c.CubeError || stored.bestMoveErr.Int64 != c.BestMoveEquityError ||
		stored.forced.Int64 != c.IsForced || stored.closeCube.Int64 != c.IsCloseCube ||
		storedEng.String != c.AnalysisEngine || storedDep.Int64 != c.AnalysisDepth
	return true, s.write(ctx, positionID, merged, c, &statsChanged)
}

// storedPlayedColumns are the stored columns that depend on the played
// actions as well as on the blob — the ones RepairDenormalisedColumns checks.
type storedPlayedColumns struct {
	bestCube                                sql.NullString
	cubeErr, bestMoveErr, forced, closeCube sql.NullInt64
}

func (p storedPlayedColumns) equal(c engine.AnalysisColumns) bool {
	return c.BestCubeAction == p.bestCube.String &&
		c.CubeError == p.cubeErr.Int64 &&
		c.BestMoveEquityError == p.bestMoveErr.Int64 &&
		c.IsForced == p.forced.Int64 &&
		c.IsCloseCube == p.closeCube.Int64
}

// prepare stamps a with its position, rounds it for storage and derives its
// scalar columns. The played actions come from the analysis when it states
// them, and from the match when it does not — see engine.PlayedActionsFor:
// from played when the caller knows the decision, else from the move table.
// The lookup is skipped when the blob or played already answers, so an import
// pays nothing for it.
func (s *analysisStore) prepare(ctx context.Context, positionID int64, a *domain.PositionAnalysis, played *storage.PlayedActions) (engine.AnalysisColumns, error) {
	a.PositionID = int(positionID)
	playedMove, playedCubeAction := engine.PlayedActionsFor(a.PlayedMoves, a.PlayedCubeActions, nil, nil)
	switch {
	case playedMove != "" && playedCubeAction != "":
	case played != nil:
		playedMove, playedCubeAction = engine.PlayedActionsFor(
			[]string{playedMove}, []string{playedCubeAction}, []string{played.CheckerMove}, []string{played.CubeAction})
	default:
		mvMove, mvCube, err := s.playedActionsFromMatch(ctx, positionID)
		if err != nil {
			return engine.AnalysisColumns{}, err
		}
		playedMove, playedCubeAction = engine.PlayedActionsFor(
			[]string{playedMove}, []string{playedCubeAction}, []string{mvMove}, []string{mvCube})
	}
	engine.RoundAnalysisForStorage(a)
	return engine.PopulateAnalysisColumns(a, playedMove, playedCubeAction), nil
}

// write encodes a prepared analysis and upserts it with its columns.
//
// statsChanged says whether the write changes a column match_stats
// summarises; nil when the caller has not read the stored row, and write
// reads it.
func (s *analysisStore) write(ctx context.Context, positionID int64, a *domain.PositionAnalysis, c engine.AnalysisColumns, statsChanged *bool) error {
	data, err := engine.EncodeAnalysisForStorage(a)
	if err != nil {
		return fmt.Errorf("sqlite: encode analysis: %w", err)
	}

	// The analysis row and the position flag it implies are one write: a
	// caller that already holds a transaction writes inside it, a caller that
	// does not gets one of its own (withTx).
	return withTx(ctx, s.db, func(tx execer) error {
		if statsChanged == nil {
			changed, err := analysisStatsColumnsChange(ctx, tx, positionID, c)
			if err != nil {
				return err
			}
			statsChanged = &changed
		}
		if *statsChanged {
			if _, err := tx.ExecContext(ctx, invalidateMatchStatsOfPositionSQL, positionID); err != nil {
				return fmt.Errorf("sqlite: invalidate match stats: %w", err)
			}
		}
		if _, err := tx.ExecContext(ctx, analysisUpsertSQL,
			positionID, data,
			c.BestCubeAction, c.CubeError, c.BestMoveEquityError,
			c.Player1WinRate, c.Player1GammonRate, c.Player1BackgammonRate,
			c.Player2WinRate, c.Player2GammonRate, c.Player2BackgammonRate,
			c.IsForced, c.IsCloseCube,
			c.AnalysisEngine, c.AnalysisDepth, nullableString(c.CreationDate)); err != nil {
			return fmt.Errorf("sqlite: save analysis: %w", referenced(err))
		}

		// Flag the position as a take/pass cube response if any played cube action is
		// a response (only ever set to 1; OR semantics for a deduped position).
		for _, action := range a.PlayedCubeActions {
			if engine.IsResponseCubeAction(action) {
				if _, err := tx.ExecContext(ctx, cubeResponseSQL, positionID); err != nil {
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
		c.IsForced, c.IsCloseCube,
		c.AnalysisEngine, c.AnalysisDepth, nullableString(c.CreationDate)); err != nil {
		return fmt.Errorf("sqlite: save analysis: %w", err)
	}
	return nil
}

// Delete removes the analysis for positionID.
// The matches reaching the position lose their match_stats rows with it:
// their counted decisions just changed.
func (s *analysisStore) Delete(ctx context.Context, scope string, positionID int64) error {
	return withTx(ctx, s.db, func(tx execer) error {
		if _, err := tx.ExecContext(ctx, invalidateMatchStatsOfPositionSQL, positionID); err != nil {
			return fmt.Errorf("sqlite: invalidate match stats of position %d: %w", positionID, err)
		}
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM analysis WHERE position_id = ?`, positionID); err != nil {
			return fmt.Errorf("sqlite: delete analysis for position %d: %w", positionID, err)
		}
		return nil
	})
}

func firstOf(s []string) string {
	if len(s) > 0 {
		return s[0]
	}
	return ""
}

// playedActionsFromMatch reads the earliest recorded checker move and cube
// action for a position from the `move` table — the actions an analysis
// computed here cannot know. Earliest by move id, so the answer is
// stable across runs when a deduplicated position was played more than once.
func (s *analysisStore) playedActionsFromMatch(ctx context.Context, positionID int64) (checkerMove, cubeAction string, err error) {
	var mv, ca sql.NullString
	err = s.db.QueryRowContext(ctx, playedActionsSQL, positionID, positionID).Scan(&mv, &ca)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", nil
	}
	if err != nil {
		return "", "", fmt.Errorf("sqlite: read played actions for position %d: %w", positionID, err)
	}
	return mv.String, ca.String, nil
}

// playedActionsSQL is one row of two scalar subqueries rather than a join: a
// position may have several move rows, and each column wants the earliest
// NON-EMPTY one of its own — a cube action and a checker move are recorded on
// different rows.
const playedActionsSQL = `SELECT
	(SELECT mv.checker_move FROM move mv WHERE mv.position_id = ? AND COALESCE(mv.checker_move, '') <> '' ORDER BY mv.id LIMIT 1),
	(SELECT mv.cube_action  FROM move mv WHERE mv.position_id = ? AND COALESCE(mv.cube_action, '')  <> '' ORDER BY mv.id LIMIT 1)`

// repairPageSize bounds how many analysis rows RepairDenormalisedColumns
// holds in memory at once (id keyset pagination, both backends): a real
// database holds tens of thousands of rows at ~600 bytes of compressed blob
// each, and the point of a repair is to run on the biggest ones.
const repairPageSize = 500

// RepairDenormalisedColumns — see storage.AnalysisStore.
//
// Rows are read, decoded and rewritten a page at a time (repairPageSize),
// never loaded whole. Only rows whose columns actually change are written, so
// a second run reports 0: the count answers "was anything wrong?".
func (s *analysisStore) RepairDenormalisedColumns(ctx context.Context, _ string) (int, error) {
	type row struct {
		id                                     int64
		data                                   []byte
		bestCube                               sql.NullString
		cubeErr, bestMoveErr, forced, closeCub sql.NullInt64
		// The match's own record of what was played, joined in here rather
		// than looked up row by row: a repair walks every analysis.
		mvMove, mvCube sql.NullString
	}
	repaired := 0
	var lastID int64
	for {
		var page []row
		if err := func() error {
			rows, err := s.db.QueryContext(ctx,
				`SELECT a.id, a.data, a.best_cube_action, a.cube_error, a.best_move_equity_error, a.is_forced, a.is_close_cube,
				        (SELECT mv.checker_move FROM move mv WHERE mv.position_id = a.position_id AND COALESCE(mv.checker_move, '') <> '' ORDER BY mv.id LIMIT 1),
				        (SELECT mv.cube_action  FROM move mv WHERE mv.position_id = a.position_id AND COALESCE(mv.cube_action, '')  <> '' ORDER BY mv.id LIMIT 1)
				 FROM analysis a WHERE a.id > ? ORDER BY a.id LIMIT ?`, lastID, repairPageSize)
			if err != nil {
				return fmt.Errorf("sqlite: repair: read analyses: %w", err)
			}
			defer rows.Close()
			for rows.Next() {
				var r row
				if err := rows.Scan(&r.id, &r.data, &r.bestCube, &r.cubeErr, &r.bestMoveErr, &r.forced, &r.closeCub, &r.mvMove, &r.mvCube); err != nil {
					return fmt.Errorf("sqlite: repair: scan: %w", err)
				}
				page = append(page, r)
			}
			return rows.Err()
		}(); err != nil {
			return repaired, err
		}
		if len(page) == 0 {
			// A repaired column is one match_stats summarises.
			if repaired > 0 {
				if _, err := s.db.ExecContext(ctx, `DELETE FROM match_stats`); err != nil {
					return repaired, fmt.Errorf("sqlite: repair: invalidate match stats: %w", err)
				}
			}
			return repaired, nil
		}
		lastID = page[len(page)-1].id

		for _, r := range page {
			a, err := engine.DecodeAnalysisFromStorage(r.data)
			if err != nil {
				// An unreadable blob is LEFT ALONE, never zeroed: the columns we have
				// may be wrong, but blanking them would lose the only information
				// left about that position.
				continue
			}
			playedMove, playedCubeAction := engine.PlayedActionsFor(
				a.PlayedMoves, a.PlayedCubeActions,
				[]string{r.mvMove.String}, []string{r.mvCube.String})
			c := engine.PopulateAnalysisColumns(&a, playedMove, playedCubeAction)
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
	}
}

// WithoutAnalysis streams the positions carrying no analysis at all, by
// ascending id. See the contract's doc comment for why this exists rather than
// a Load per position.
//
// LEFT JOIN … IS NULL rather than NOT EXISTS: the two are equivalent here and
// SQLite plans them the same way, but the join says in one line what the sweep
// is looking for.
func (s *analysisStore) WithoutAnalysis(ctx context.Context, _ string, opts storage.ListOpts) iter.Seq2[*domain.Position, error] {
	return func(yield func(*domain.Position, error) bool) {
		query := `SELECT ` + qualify(positionCols, "p") + `
			 FROM position p LEFT JOIN analysis a ON a.position_id = p.id
			 WHERE a.position_id IS NULL
			 ORDER BY p.id` + opts.SQL("LIMIT -1")
		rows, err := s.db.QueryContext(ctx, query)
		if err != nil {
			yield(nil, fmt.Errorf("sqlite: list positions without analysis: %w", err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			p, err := scanPosition(rows)
			if err != nil {
				yield(nil, fmt.Errorf("sqlite: list positions without analysis: %w", err))
				return
			}
			if !yield(&p, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, fmt.Errorf("sqlite: list positions without analysis: %w", err))
		}
	}
}

// invalidateMatchStatsOfPositionSQL drops the match_stats rows of every
// match reaching one position: an analysis write that changes a column
// those rows summarise (or gives the position its first analysis) makes
// them stale.
const invalidateMatchStatsOfPositionSQL = sqlshared.InvalidateMatchStatsOfPositionsSQL + "(?)" + sqlshared.InvalidateMatchStatsOfPositionsSuffix

// analysisStatsColumnsChange reports whether writing c over positionID's
// stored analysis changes a column match_stats summarises.
func analysisStatsColumnsChange(ctx context.Context, tx execer, positionID int64, c engine.AnalysisColumns) (bool, error) {
	var cubeErr, moveErr, forced, closeCube, depth sql.NullInt64
	var eng sql.NullString
	err := tx.QueryRowContext(ctx, `SELECT cube_error, best_move_equity_error, is_forced, is_close_cube,
		analysis_engine, analysis_depth FROM analysis WHERE position_id = ?`, positionID).
		Scan(&cubeErr, &moveErr, &forced, &closeCube, &eng, &depth)
	if errors.Is(err, sql.ErrNoRows) {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("sqlite: read analysis columns: %w", err)
	}
	return !cubeErr.Valid || !moveErr.Valid || cubeErr.Int64 != c.CubeError || moveErr.Int64 != c.BestMoveEquityError ||
		forced.Int64 != c.IsForced || closeCube.Int64 != c.IsCloseCube ||
		eng.String != c.AnalysisEngine || depth.Int64 != c.AnalysisDepth, nil
}

// withEngineBatch is how many candidate rows WithEngine reads and decodes per
// round trip.
const withEngineBatch = 1000

// WithEngine — see storage.AnalysisStore.
func (s *analysisStore) WithEngine(ctx context.Context, _ string, enginePrefix string) iter.Seq2[storage.AnalysisRecord, error] {
	return func(yield func(storage.AnalysisRecord, error) bool) {
		var last int64
		for {
			raw, ids, err := s.engineBatch(ctx, last, enginePrefix)
			if err != nil {
				yield(storage.AnalysisRecord{}, fmt.Errorf("sqlite: analyses by engine: %w", err))
				return
			}
			if len(ids) == 0 {
				return
			}
			last = ids[len(ids)-1]
			decoded, failed := engine.DecodeAnalysesConcurrently(raw)
			for id, err := range failed {
				slog.Warn("decoding stored analysis", "positionID", id, "err", err)
			}
			for _, id := range ids {
				if a, ok := decoded[id]; ok && !yield(storage.AnalysisRecord{PositionID: id, Analysis: a}, nil) {
					return
				}
			}
		}
	}
}

// engineBatch reads the next withEngineBatch candidates past last, closing its
// cursor before returning.
func (s *analysisStore) engineBatch(ctx context.Context, last int64, enginePrefix string) (map[int64][]byte, []int64, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT position_id, data FROM analysis
		 WHERE position_id > ? AND (analysis_engine IS NULL OR substr(analysis_engine, 1, ?) = ?)
		 ORDER BY position_id LIMIT ?`,
		last, len(enginePrefix), enginePrefix, withEngineBatch)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	raw := make(map[int64][]byte, withEngineBatch)
	var ids []int64
	for rows.Next() {
		var id int64
		var data []byte
		if err := rows.Scan(&id, &data); err != nil {
			return nil, nil, err
		}
		raw[id] = data
		ids = append(ids, id)
	}
	return raw, ids, rows.Err()
}

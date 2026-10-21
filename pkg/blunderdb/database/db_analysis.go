package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// The analysis-encoding helpers (zstd compression, scalar-column derivation,
// float rounding) live in package engine so the SQLite Storage backend can
// share them. These aliases keep this package compiling against the
// unqualified names. See pkg/blunderdb/engine/analysiscodec.go.
var (
	compressAnalysisData      = engine.CompressAnalysisData
	recompressAnalysisData    = engine.RecompressAnalysisData
	encodeAnalysisForStorage  = engine.EncodeAnalysisForStorage
	decodeAnalysisFromStorage = engine.DecodeAnalysisFromStorage
	computeIsCloseCube        = engine.ComputeIsCloseCube
	populateAnalysisColumns   = engine.PopulateAnalysisColumns
	roundToMillipoint         = engine.RoundToMillipoint
	roundToHundredthPercent   = engine.RoundToHundredthPercent
	roundAnalysisForStorage   = engine.RoundAnalysisForStorage
)

// positionUnknown is what the historical migrations pass for the position:
// they ran before is_forced needed it, and migrate_2_36_0_to_2_37_0 recomputes
// the columns with it.
var positionUnknown *domain.Position

func (d *Database) SaveAnalysis(positionID int64, analysis PositionAnalysis) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.saveAnalysisLocked(positionID, analysis, nil)
}

// saveAnalysisAt is SaveAnalysis for a job that started on generation gen: it
// refuses to write once another database has been opened in the meantime.
// The analysis is a gammonNet verdict valued with the table metID.
func (d *Database) saveAnalysisAt(gen uint64, positionID int64, analysis PositionAnalysis, metID int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.generation != gen {
		return ErrDatabaseChanged
	}
	return d.saveAnalysisLocked(positionID, analysis, &metID)
}

// saveAnalysisLocked is SaveAnalysis's body; the caller holds d.mu. A
// non-nil metID is the match equity table a gammonNet verdict was valued
// with (0: the built-in one), written in the analysis's own transaction: a
// verdict stored without its table would be read as Kazaross-XG2 (ADR-0068).
func (d *Database) saveAnalysisLocked(positionID int64, analysis PositionAnalysis, metID *int64) error {
	ctx := context.Background()
	analysis.PositionID = int(positionID)
	analysis.LastModifiedDate = time.Now()

	var existing *PositionAnalysis
	stored, err := d.store.Analyses().Load(ctx, "", positionID)
	switch {
	case err == nil:
		existing = stored
	case !errors.Is(err, storage.ErrNotFound):
		return err
	}

	// A gammonNet verdict is written by the rule the serve daemon writes by
	// too, with or without a stored row, so both leave the same row
	// (CLI/GUI/server parity); any other caller's analysis is merged.
	if metID != nil {
		analysis = gammonnet.SupersedeEntries(existing, analysis)
		if analysis.CreationDate.IsZero() {
			analysis.CreationDate = time.Now()
		}
		return d.saveAnalysisTx(positionID, &analysis, *metID)
	}
	analysis = mergeSavedAnalysis(existing, analysis)

	// The store rounds, encodes and derives the scalar columns, then
	// inserts-or-updates the row keyed by position_id.
	return d.store.Analyses().Save(ctx, "", positionID, &analysis)
}

// mergeSavedAnalysis folds analysis into existing, the analysis the position
// already carries (nil when none): the candidate moves and cube analyses of
// both, keyed by notation and by engine, the union of what was played, the
// rollouts of both, the creation date of the stored one. The moves are ranked
// by equity (engine.RankCheckerMoves).
func mergeSavedAnalysis(existing *PositionAnalysis, analysis PositionAnalysis) PositionAnalysis {
	if existing == nil {
		if analysis.CreationDate.IsZero() {
			analysis.CreationDate = time.Now()
		}
		if analysis.PlayedMove != "" && len(analysis.PlayedMoves) == 0 {
			analysis.PlayedMoves = []string{analysis.PlayedMove}
			analysis.PlayedMove = ""
		}
		if analysis.PlayedCubeAction != "" && len(analysis.PlayedCubeActions) == 0 {
			analysis.PlayedCubeActions = []string{analysis.PlayedCubeAction}
			analysis.PlayedCubeAction = ""
		}
		if analysis.CheckerAnalysis != nil {
			engine.RankCheckerMoves(analysis.CheckerAnalysis.Moves)
		}
		return analysis
	}

	analysis.CreationDate = existing.CreationDate

	if existing.CheckerAnalysis != nil && analysis.CheckerAnalysis != nil {
		analysis.CheckerAnalysis.Moves = engine.MergeCheckerMoves(
			existing.CheckerAnalysis.Moves, analysis.CheckerAnalysis.Moves)
	} else if existing.CheckerAnalysis != nil {
		analysis.CheckerAnalysis = existing.CheckerAnalysis
	}

	// Every engine's cube analysis is kept (ADR-0013). DoublingCubeAnalysis
	// stays the PRIMARY one the scalar columns read; AllCubeAnalyses carries
	// the whole set.
	analysis.DoublingCubeAnalysis, analysis.AllCubeAnalyses = engine.MergeCubeAnalyses(
		existing.DoublingCubeAnalysis, existing.AllCubeAnalyses,
		analysis.DoublingCubeAnalysis, analysis.AllCubeAnalyses)

	// The old single fields still count as a play when the array is empty.
	analysis.PlayedMoves = engine.UnionPlayed(
		playedOrSingle(existing.PlayedMoves, existing.PlayedMove),
		playedOrSingle(analysis.PlayedMoves, analysis.PlayedMove))
	analysis.PlayedCubeActions = engine.UnionPlayed(
		playedOrSingle(existing.PlayedCubeActions, existing.PlayedCubeAction),
		playedOrSingle(analysis.PlayedCubeActions, analysis.PlayedCubeAction))
	analysis.PlayedMove = ""
	analysis.PlayedCubeAction = ""

	// Rollouts are their own analyses: a caller that does not know them
	// (an import, the frontend saving what it edited) must not drop them.
	analysis.Rollouts = domain.MergeRollouts(existing.Rollouts, analysis.Rollouts)

	if analysis.CheckerAnalysis != nil {
		engine.RankCheckerMoves(analysis.CheckerAnalysis.Moves)
	}
	return analysis
}

// playedOrSingle is played, or the legacy single field when played is empty.
func playedOrSingle(played []string, single string) []string {
	if single != "" && len(played) == 0 {
		return []string{single}
	}
	return played
}

// saveAnalysisTx writes analysis and tags it with the table metID in one
// transaction: a verdict stored without its table would be read as
// Kazaross-XG2 (ADR-0068).
func (d *Database) saveAnalysisTx(positionID int64, analysis *PositionAnalysis, metID int64) error {
	ctx := context.Background()
	tx, err := d.store.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := tx.Analyses().Save(ctx, "", positionID, analysis); err != nil {
		return err
	}
	if err := tx.MatchEquityTables().TagAnalyses(ctx, "", metID, []int64{positionID}); err != nil {
		return err
	}
	return tx.Commit()
}

func (d *Database) LoadAnalysis(positionID int64) (*PositionAnalysis, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	loaded, err := d.store.Analyses().Load(context.Background(), "", positionID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			// Preserve the pre-delegation contract: callers expect sql.ErrNoRows.
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	analysis := *loaded

	// What the matches played at this position supplements the played
	// actions the analysis stored.
	byPosition, err := d.store.Matches().MovesByPositions(context.Background(), "", []int64{positionID})
	if err != nil {
		return nil, err
	}
	var checkerMoves, cubeActions []string
	for _, mv := range byPosition[positionID] {
		checkerMoves = append(checkerMoves, mv.CheckerMove)
		cubeActions = append(cubeActions, mv.CubeAction)
	}
	mergePlayedActions(&analysis, checkerMoves, cubeActions)

	// Sort AllCubeAnalyses so XG comes first (for existing data in DB)
	if len(analysis.AllCubeAnalyses) > 0 {
		engine.SortCubeAnalysesByEngine(analysis.AllCubeAnalyses)
	}

	return &analysis, nil
}

// mergePlayedActions folds the moves recorded against a position into its
// analysis's played actions: the union of what the analysis stored and what
// the matches played there, normalised, sorted, the legacy single fields set
// to the first. An empty entry is not a play.
func mergePlayedActions(analysis *PositionAnalysis, checkerMoves, cubeActions []string) {
	existingMoves := make(map[string]bool)
	for _, m := range analysis.PlayedMoves {
		existingMoves[engine.NormalizeMove(m)] = true
	}
	// Backward compatibility: include old single PlayedMove field
	if analysis.PlayedMove != "" {
		existingMoves[engine.NormalizeMove(analysis.PlayedMove)] = true
	}
	for _, m := range checkerMoves {
		if m != "" {
			existingMoves[engine.NormalizeMove(m)] = true
		}
	}
	analysis.PlayedMoves = make([]string, 0, len(existingMoves))
	for m := range existingMoves {
		analysis.PlayedMoves = append(analysis.PlayedMoves, m)
	}
	sort.Strings(analysis.PlayedMoves)

	existingCubeActions := make(map[string]bool)
	for _, a := range analysis.PlayedCubeActions {
		existingCubeActions[a] = true
	}
	// Backward compatibility: include old single PlayedCubeAction field
	if analysis.PlayedCubeAction != "" {
		existingCubeActions[analysis.PlayedCubeAction] = true
	}
	for _, a := range cubeActions {
		if a != "" {
			existingCubeActions[a] = true
		}
	}
	analysis.PlayedCubeActions = make([]string, 0, len(existingCubeActions))
	for a := range existingCubeActions {
		analysis.PlayedCubeActions = append(analysis.PlayedCubeActions, a)
	}
	sort.Strings(analysis.PlayedCubeActions)

	// The old single fields, read by older frontends, hold the first.
	if len(analysis.PlayedMoves) > 0 {
		analysis.PlayedMove = analysis.PlayedMoves[0]
	}
	if len(analysis.PlayedCubeActions) > 0 {
		analysis.PlayedCubeAction = analysis.PlayedCubeActions[0]
	}
}

// DeleteAnalysis removes the analysis of positionID through the contract,
// which also unscores the moves played there and drops the match_stats rows
// of the matches reaching it.
func (d *Database) DeleteAnalysis(positionID int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.store.Analyses().Delete(context.Background(), "", positionID)
}

// RepairAnalyses recomputes the scalar columns of every analysis from its
// stored JSON and returns how many rows actually changed.
//
// The columns are a projection of `data`, so a projection bug is repairable
// without re-importing (see storage/analyses.go). Explicit, never automatic
// on open.
func (d *Database) RepairAnalyses() (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return 0, fmt.Errorf("no database is currently open")
	}
	return d.store.Analyses().RepairDenormalisedColumns(context.Background(), "")
}

// RebuildMatchStats discards the per-match statistics (match_stats) and
// computes them all again from the moves; it returns the number of matches.
// Every write that changes what a row summarises already drops it, so this
// is the explicit way back after a bug in that bookkeeping, never a routine.
func (d *Database) RebuildMatchStats() (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return 0, fmt.Errorf("no database is currently open")
	}
	return d.store.Stats().RebuildMatchStats(context.Background(), "", nil)
}

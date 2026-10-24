package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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
	if metID == nil {
		// The store rounds, encodes and derives the scalar columns; the fold
		// is the daemon's too (CLI/GUI/server parity).
		return storage.SaveMergedAnalysis(ctx, d.store.Analyses(), "", positionID, analysis)
	}
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
	// (CLI/GUI/server parity).
	analysis = gammonnet.SupersedeEntries(existing, analysis)
	if analysis.CreationDate.IsZero() {
		analysis.CreationDate = time.Now()
	}
	return d.saveAnalysisTx(positionID, &analysis, *metID)
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

	analysis, err := storage.LoadAnalysisWithPlayed(context.Background(), d.store, "", positionID)
	if errors.Is(err, storage.ErrNotFound) {
		// Preserve the pre-delegation contract: callers expect sql.ErrNoRows.
		return nil, sql.ErrNoRows
	}
	return analysis, err
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

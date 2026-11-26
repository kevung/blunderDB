package database

import (
	"fmt"
	"log/slog"

	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

// ============================================================================
// BGBlitz BGF import functions
// ============================================================================

// ImportBGFMatch imports a match from a BGBlitz BGF file, delegating to the
// shared ingest pipeline (ingest.MapBGF -> ingest.WriteMatch) — the same path
// the headless server uses.
// The desktop's task tracker lists it (TaskTracker).
func (d *Database) ImportBGFMatch(filePath string) (int64, error) {
	return tracked(d, TaskImport, filePath, d.CancelImport, func() (int64, error) { return d.importBGFMatch(filePath) })
}

// importBGFMatch is ImportBGFMatch without the tracker.
func (d *Database) importBGFMatch(filePath string) (int64, error) {
	ctx, done := d.beginCancellableImport()
	defer done()

	d.mu.Lock()
	defer d.mu.Unlock()

	graph, err := ingest.MapBGF(filePath)
	if err != nil {
		return 0, err
	}
	matchID, err := d.writeImportedMatch(ctx, graph)
	if err != nil {
		return 0, err
	}
	slog.Info("imported BGF match", "matchID", matchID, "file", filePath)
	return matchID, nil
}

// ImportBGFPosition imports a single BGBlitz position from a TXT file
// The desktop's task tracker lists it (TaskTracker).
func (d *Database) ImportBGFPosition(filePath string) (int64, error) {
	return tracked(d, TaskImport, filePath, d.CancelImport, func() (int64, error) { return d.importBGFPosition(filePath) })
}

// importBGFPosition is ImportBGFPosition without the tracker.
func (d *Database) importBGFPosition(filePath string) (int64, error) {
	ctx, done := d.beginCancellableImport()
	defer done()

	d.mu.Lock()
	defer d.mu.Unlock()

	graphs, err := ingest.MapBGFTextPosition(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to parse BGBlitz position file: %w", err)
	}
	return d.writeImportedPosition(ctx, graphs)
}

// ImportBGFPositionFromText imports a BGBlitz position from text content (clipboard/string)
// The desktop's task tracker lists it (TaskTracker).
func (d *Database) ImportBGFPositionFromText(content string) (int64, error) {
	return tracked(d, TaskImport, "", d.CancelImport, func() (int64, error) { return d.importBGFPositionFromText(content) })
}

// importBGFPositionFromText is ImportBGFPositionFromText without the tracker.
func (d *Database) importBGFPositionFromText(content string) (int64, error) {
	ctx, done := d.beginCancellableImport()
	defer done()

	d.mu.Lock()
	defer d.mu.Unlock()

	graphs, err := ingest.MapBGFTextPositionText(content)
	if err != nil {
		return 0, fmt.Errorf("failed to parse BGBlitz position text: %w", err)
	}
	return d.writeImportedPosition(ctx, graphs)
}

// ImportXGPPosition imports an XG position file (.xgp) as a standalone position with analysis.
// XGP files use the same binary format as .xg match files but contain a single position.
func (d *Database) ImportXGPPosition(filePath string) (int64, error) {
	ctx, done := d.beginCancellableImport()
	defer done()

	d.mu.Lock()
	defer d.mu.Unlock()

	graphs, err := ingest.MapXGPPosition(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to parse XGP file: %w", err)
	}
	return d.writeImportedPosition(ctx, graphs)
}

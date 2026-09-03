package database

import (
	"context"
	"fmt"
	"os"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

// ExportDatabase writes the selection described by opts into a new database
// file. It is the GUI's export dialog and the CLI's `export`; the work is
// ingest.ExportSQLite, the one exporter every mode runs — this method only
// seals the watermark with this machine's identity and translates the
// dialog's options into a Selection. The public signature is bound to Wails
// and stays.
//
// This is the context.Background() convenience for callers with no context of
// their own (the GUI); ExportDatabaseCtx is the one to call when the caller
// can offer a real deadline/cancellation (B.13, #181: the CLI's `export`
// cancels a large export on Ctrl-C through it).
func (d *Database) ExportDatabase(opts ExportOptions) error {
	return d.ExportDatabaseCtx(context.Background(), opts)
}

// ExportDatabaseCtx is ExportDatabase with a caller-supplied context: ctx is
// threaded into ingest.ExportSQLite, so cancelling it aborts an in-flight
// export — a large selection can run long enough to matter — instead of
// writing a file nobody will read to completion.
func (d *Database) ExportDatabaseCtx(ctx context.Context, opts ExportOptions) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}

	// Seal the watermark before writing anything: it is the producer's statement about this
	// file, and a failure to sign must not leave a half-made export behind.
	watermarkDocument, err := sealWatermark(opts.Watermark, opts.WatermarkNote)
	if err != nil {
		return fmt.Errorf("cannot sign the watermark: %w", err)
	}

	sel := ingest.Selection{
		AllPositions:  opts.AllPositions,
		PositionIDs:   opts.PositionIDs,
		TournamentIDs: opts.TournamentIDs,
	}
	// Positions wins over PositionIDs when set: callers holding positions
	// that are not in the database keep working.
	if len(opts.Positions) > 0 {
		sel.AllPositions, sel.PositionIDs = false, nil
		sel.Positions = make([]*domain.Position, len(opts.Positions))
		for i := range opts.Positions {
			sel.Positions[i] = &opts.Positions[i]
		}
	}
	if opts.IncludeCollections {
		sel.CollectionIDs = opts.CollectionIDs
	}
	// IncludeMatches with no ids means every match (the CLI's --match-ids
	// empty=all convention; the GUI collapses an empty pick to
	// IncludeMatches=false itself).
	if opts.IncludeMatches {
		sel.MatchIDs = opts.MatchIDs
		sel.AllMatches = len(opts.MatchIDs) == 0
	}

	_, err = ingest.ExportSQLite(ctx, d.store, "", opts.ExportPath, ingest.ExportOptions{
		Format:        ingest.FormatSQLite,
		Selection:     sel,
		Analysis:      opts.IncludeAnalysis,
		Comments:      opts.IncludeComments,
		PlayedMoves:   opts.IncludePlayedMoves,
		FilterLibrary: opts.IncludeFilterLibrary,
		Metadata:      opts.Metadata,
		Watermark:     watermarkDocument,
		Password:      opts.Password,
	})
	return err
}

// DeleteFile is a helper function to delete a file
func DeleteFile(filePath string) error {
	err := os.Remove(filePath)
	if err != nil {
		return err
	}
	return nil
}

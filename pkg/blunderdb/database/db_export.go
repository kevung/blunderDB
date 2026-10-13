package database

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

// ExportDatabase writes the selection described by opts into a new database
// file. It is the GUI's export dialog and the CLI's `export`; the work is
// ingest.ExportSQLite; this method only seals the watermark with this
// machine's identity and maps the options to a Selection. Uses
// context.Background(); prefer ExportDatabaseCtx when the caller can cancel.
// The report counts what the file holds, family by family.
func (d *Database) ExportDatabase(opts ExportOptions) (ingest.ExportReport, error) {
	return d.ExportDatabaseCtx(context.Background(), opts)
}

// ExportDatabaseCtx is ExportDatabase with a caller-supplied context that
// aborts an in-flight export.
func (d *Database) ExportDatabaseCtx(ctx context.Context, opts ExportOptions) (ingest.ExportReport, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.db == nil {
		return ingest.ExportReport{}, fmt.Errorf("no database is currently open")
	}

	// Seal the watermark before writing anything: it is the producer's statement about this
	// file, and a failure to sign must not leave a half-made export behind.
	watermarkDocument, err := sealWatermark(opts.Watermark, opts.WatermarkNote)
	if err != nil {
		return ingest.ExportReport{}, fmt.Errorf("cannot sign the watermark: %w", err)
	}

	plan, err := d.exportPlan(ctx, opts)
	if err != nil {
		return ingest.ExportReport{}, err
	}
	plan.Metadata = opts.Metadata
	plan.Watermark = watermarkDocument
	plan.Password = opts.Password
	return ingest.ExportSQLite(ctx, d.store, "", opts.ExportPath, plan)
}

// PlanExport counts, family by family, what ExportDatabase would write for the same
// options, and names the living collections that would leave frozen. It writes nothing:
// the export dialog shows it before the user commits to a file.
func (d *Database) PlanExport(opts ExportOptions) (ingest.ExportPlan, error) {
	return d.PlanExportCtx(context.Background(), opts)
}

// PlanExportCtx is PlanExport with a caller-supplied context.
func (d *Database) PlanExportCtx(ctx context.Context, opts ExportOptions) (ingest.ExportPlan, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.db == nil {
		return ingest.ExportPlan{}, fmt.Errorf("no database is currently open")
	}
	plan, err := d.exportPlan(ctx, opts)
	if err != nil {
		return ingest.ExportPlan{}, err
	}
	return ingest.PlanExport(ctx, d.store, "", plan)
}

// PlanExportIDs pages through the source ids behind one count of PlanExport — family is
// one of ingest.Plan* — so the dialog can open what a count counts; a page holds at most
// ingest.MaxPlannedIDs.
func (d *Database) PlanExportIDs(opts ExportOptions, family string, offset, limit int) (ingest.PlannedIDs, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.db == nil {
		return ingest.PlannedIDs{}, fmt.Errorf("no database is currently open")
	}
	plan, err := d.exportPlan(context.Background(), opts)
	if err != nil {
		return ingest.PlannedIDs{}, err
	}
	return ingest.PlanExportIDs(context.Background(), d.store, "", plan, family, offset, limit)
}

// exportPlan maps the GUI's and the CLI's options onto what ingest exports.
//
// The whole library is ingest.WholeTenant, the daemon's export, so the three entry points
// cannot disagree on what "everything" carries; a family the caller turns off or narrows to
// some ids is then taken out of it. Anything else is a list of positions, and the matches
// it carries are, unless named, those its positions were played in.
func (d *Database) exportPlan(ctx context.Context, opts ExportOptions) (ingest.ExportOptions, error) {
	if opts.WholeLibrary {
		return wholeLibraryPlan(opts), nil
	}
	plan := ingest.ExportOptions{
		Format:         ingest.FormatSQLite,
		Analysis:       opts.IncludeAnalysis,
		Comments:       opts.IncludeComments,
		PlayedMoves:    opts.IncludePlayedMoves,
		FilterLibrary:  opts.IncludeFilterLibrary,
		Transcriptions: opts.IncludeTranscriptions,
		Aliases:        opts.IncludeAliases,
		AnkiHistory:    opts.IncludeAnkiHistory,
	}
	sel := ingest.Selection{
		AllPositions:  opts.AllPositions,
		PositionIDs:   opts.PositionIDs,
		TournamentIDs: opts.TournamentIDs,
		// A match leaves with the positions its moves were played in: without them its
		// moves would point at nothing in the file.
		MatchPositions: true,
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
		sel.CollectionPositions = opts.IncludeCollectionPositions
	}
	if opts.IncludeMatches {
		switch {
		case len(opts.MatchIDs) > 0:
			sel.MatchIDs = opts.MatchIDs
		case sel.AllPositions:
			sel.AllMatches = true
		default:
			ids := sel.PositionIDs
			for _, p := range sel.Positions {
				ids = append(ids, p.ID)
			}
			reached, err := d.matchIDsReachedBy(ctx, ids)
			if err != nil {
				return plan, err
			}
			sel.MatchIDs = reached
		}
	}
	// A tournament chosen carries its matches, whatever the matches family says.
	sel.TournamentMatches = len(sel.TournamentIDs) > 0
	sel.DeckIDs, sel.DeckPositions = opts.DeckIDs, len(opts.DeckIDs) > 0
	if opts.IncludeLessons {
		sel.LessonIDs = opts.LessonIDs
		sel.AllLessons = len(opts.LessonIDs) == 0
		sel.LessonContents = true
	}
	plan.Selection = sel
	return plan, nil
}

// wholeLibraryPlan is ingest.WholeTenant less what opts leaves out. Each family follows the
// same rule: off drops it, a list of ids narrows it to them, on with no ids keeps all of it.
func wholeLibraryPlan(opts ExportOptions) ingest.ExportOptions {
	plan := ingest.WholeTenant(ingest.FormatSQLite)
	plan.Analysis = opts.IncludeAnalysis
	plan.Comments = opts.IncludeComments
	plan.PlayedMoves = opts.IncludePlayedMoves
	plan.FilterLibrary = opts.IncludeFilterLibrary
	plan.Transcriptions = opts.IncludeTranscriptions
	plan.Aliases = opts.IncludeAliases
	plan.AnkiHistory = opts.IncludeAnkiHistory
	sel := &plan.Selection
	sel.AllMatches, sel.MatchIDs = narrow(opts.IncludeMatches, opts.MatchIDs)
	sel.AllCollections, sel.CollectionIDs = narrow(opts.IncludeCollections, opts.CollectionIDs)
	sel.AllTournaments, sel.TournamentIDs = narrow(opts.IncludeTournaments, opts.TournamentIDs)
	// A tournament chosen carries its matches, whatever the matches family says.
	sel.TournamentMatches = sel.AllTournaments || len(sel.TournamentIDs) > 0
	sel.AllLessons, sel.LessonIDs = narrow(opts.IncludeLessons, opts.LessonIDs)
	sel.LessonContents = opts.IncludeLessons
	plan.AnkiDecks, sel.DeckIDs = narrow(opts.IncludeAnkiDecks, opts.DeckIDs)
	return plan
}

// narrow reads one family of a whole-library export: all of it, the listed ids, or none.
func narrow(include bool, ids []int64) (all bool, picked []int64) {
	if !include {
		return false, nil
	}
	if len(ids) > 0 {
		return false, ids
	}
	return true, nil
}

// matchIDsReachedBy returns the matches whose moves were played in one of these positions,
// in id order. The caller holds d.mu.
func (d *Database) matchIDsReachedBy(ctx context.Context, positionIDs []int64) ([]int64, error) {
	seen := make(map[int64]bool)
	const chunk = 500
	for start := 0; start < len(positionIDs); start += chunk {
		if err := d.collectMatchIDs(ctx, positionIDs[start:min(start+chunk, len(positionIDs))], seen); err != nil {
			return nil, fmt.Errorf("matches reached by the exported positions: %w", err)
		}
	}
	ids := make([]int64, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids, nil
}

// collectMatchIDs adds to seen the matches whose moves were played in one of positionIDs.
func (d *Database) collectMatchIDs(ctx context.Context, positionIDs []int64, seen map[int64]bool) error {
	args := make([]any, len(positionIDs))
	for i, id := range positionIDs {
		args[i] = id
	}
	rows, err := d.db.QueryContext(ctx, `
		SELECT DISTINCT g.match_id FROM move mv JOIN game g ON g.id = mv.game_id
		WHERE mv.position_id IN (?`+strings.Repeat(",?", len(positionIDs)-1)+`)`, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return err
		}
		seen[id] = true
	}
	return rows.Err()
}

// MatchIDsReachedBy returns the matches whose moves were played in one of these positions:
// what an export of these positions carries when its matches are left unnamed.
func (d *Database) MatchIDsReachedBy(positionIDs []int64) ([]int64, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}
	return d.matchIDsReachedBy(context.Background(), positionIDs)
}

// DeleteFile is a helper function to delete a file
func DeleteFile(filePath string) error {
	err := os.Remove(filePath)
	if err != nil {
		return err
	}
	return nil
}

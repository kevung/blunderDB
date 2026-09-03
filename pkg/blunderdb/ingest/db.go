package ingest

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// DBImporter imports a native blunderDB .db file's position library —
// positions plus their analysis and comments — into the target Storage. It is
// the backend-agnostic counterpart of database.CommitImportDatabase: a
// Storage→Storage merge rather than a parser→MatchGraph map. (The legacy native
// import is position-library only; it does not copy match/game/move rows, and
// neither does this.)
//
// Merge semantics mirror the legacy importer:
//   - positions dedup by content (PositionStore.Save's Zobrist index);
//   - an imported analysis is written only when the target has none, or when
//     the target's analysis is empty-typed and the import's is not;
//   - a comment is appended only when the target doesn't already contain it.
type DBImporter struct{ S storage.Storage }

func (im DBImporter) Import(ctx context.Context, scope string, src Source, prog func(Progress)) (Summary, error) {
	if src.Path == "" {
		return Summary{}, fmt.Errorf("ingest: native .db import requires a file path")
	}

	// sqlite.Open bootstraps a fresh database when the file does not exist —
	// the right thing for a database being created, the wrong thing for one
	// being imported: a mistyped path would leave a new, empty .db on disk
	// and import nothing from it.
	if info, err := os.Stat(src.Path); err != nil {
		return Summary{}, fmt.Errorf("ingest: source .db not found: %w", err)
	} else if info.IsDir() {
		return Summary{}, fmt.Errorf("ingest: source .db not found: %s is a directory", src.Path)
	}
	source, err := sqlite.Open(ctx, src.Path, nil)
	if err != nil {
		return Summary{}, fmt.Errorf("ingest: open source .db: %w", err)
	}
	defer source.Close()

	// Drain the source position list before issuing per-position follow-up
	// queries: nesting them inside the List iterator would hold one pooled
	// connection while grabbing another (see JSONExporter).
	var positions []*domain.Position
	var ids []int64
	for p, err := range source.Positions().List(ctx, scope, storage.ListOpts{}) {
		if err != nil {
			return Summary{}, fmt.Errorf("ingest: list source positions: %w", err)
		}
		pc := *p
		positions = append(positions, &pc)
		ids = append(ids, p.ID)
	}

	// Source analyses and comments, batched (B.11, #179): one round trip per
	// family instead of one Analyses().Load and one Comments().ByPosition per
	// position — a source library of tens of thousands of positions used to
	// mean as many source-side queries again on top of the position list.
	srcAnalyses, err := source.Analyses().LoadMany(ctx, scope, ids)
	if err != nil {
		return Summary{}, fmt.Errorf("ingest: load source analyses: %w", err)
	}
	srcComments, err := source.Comments().ByPositions(ctx, scope, ids)
	if err != nil {
		return Summary{}, fmt.Errorf("ingest: read source comments: %w", err)
	}

	type srcRecord struct {
		pos      *domain.Position
		analysis *domain.PositionAnalysis
		comments []string
	}
	records := make([]srcRecord, 0, len(positions))
	for _, p := range positions {
		rec := srcRecord{pos: p, analysis: srcAnalyses[p.ID]}
		for _, c := range srcComments[p.ID] {
			rec.comments = append(rec.comments, c.Text)
		}
		records = append(records, rec)
	}

	tx, err := im.S.BeginTx(ctx)
	if err != nil {
		return Summary{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	// Every source position is saved first: PositionStore.Save dedups by
	// content (Zobrist), so the target id is not known before the call
	// returns, and a merge decision needs that id to look up what the target
	// already holds. Two source positions never land on the same target id
	// within one import (the source database is itself deduplicated), so a
	// snapshot of the target's existing analyses/comments taken once, before
	// any of this batch's merges run, is equivalent to querying it fresh for
	// each one.
	type savedRecord struct {
		rec *srcRecord
		id  int64
	}
	saved := make([]savedRecord, 0, len(records))
	targetIDs := make([]int64, 0, len(records))
	for i := range records {
		if err := ctx.Err(); err != nil {
			return Summary{}, err
		}
		rec := &records[i]
		pc := *rec.pos
		pc.ID = 0
		id, err := tx.Positions().Save(ctx, scope, &pc)
		if err != nil {
			return Summary{}, err
		}
		saved = append(saved, savedRecord{rec: rec, id: id})
		targetIDs = append(targetIDs, id)
	}

	targetAnalyses, err := tx.Analyses().LoadMany(ctx, scope, targetIDs)
	if err != nil {
		return Summary{}, err
	}
	targetComments, err := tx.Comments().ByPositions(ctx, scope, targetIDs)
	if err != nil {
		return Summary{}, err
	}

	var sum Summary
	for _, s := range saved {
		if err := ctx.Err(); err != nil {
			return sum, err
		}
		if s.rec.analysis != nil {
			if err := mergeDBAnalysisPreloaded(ctx, tx, scope, s.id, targetAnalyses[s.id], s.rec.analysis); err != nil {
				return sum, err
			}
		}
		if err := mergeDBCommentsPreloaded(ctx, tx, scope, s.id, targetComments[s.id], s.rec.comments); err != nil {
			return sum, err
		}
		sum.SavedPositions++
		if prog != nil {
			prog(Progress{Positions: sum.SavedPositions})
		}
	}

	if err := ctx.Err(); err != nil {
		return sum, err
	}
	if err := tx.Commit(); err != nil {
		return sum, err
	}
	committed = true
	return sum, nil
}

// mergeDBAnalysisPreloaded writes an imported analysis for positionID
// following the legacy "prefer existing non-empty analysis" rule. existing is
// the target's current analysis for positionID, already loaded in Import's
// batched pass (AnalysisStore.LoadMany) — this is the batched counterpart of
// what used to be its own Analyses().Load per position (B.11, #179).
func mergeDBAnalysisPreloaded(ctx context.Context, tx storage.Tx, scope string, positionID int64, existing, imported *domain.PositionAnalysis) error {
	if existing == nil || (existing.AnalysisType == "" && imported.AnalysisType != "") {
		return tx.Analyses().Save(ctx, scope, positionID, imported)
	}
	return nil
}

// mergeDBCommentsPreloaded appends each imported comment to positionID unless
// the position's existing comment text already contains it. existing is the
// target's current comment entries for positionID, already loaded in
// Import's batched pass (CommentStore.ByPositions) — the batched counterpart
// of what used to be its own Comments().Text per position (B.11, #179).
func mergeDBCommentsPreloaded(ctx context.Context, tx storage.Tx, scope string, positionID int64, existingEntries []*domain.CommentEntry, comments []string) error {
	if len(comments) == 0 {
		return nil
	}
	parts := make([]string, len(existingEntries))
	for i, e := range existingEntries {
		parts[i] = e.Text
	}
	existing := strings.Join(parts, "\n\n")
	for _, text := range comments {
		trimmed := strings.TrimSpace(text)
		if trimmed == "" || strings.Contains(existing, trimmed) {
			continue
		}
		if _, err := tx.Comments().Add(ctx, scope, positionID, text); err != nil {
			return err
		}
		if existing == "" {
			existing = trimmed
		} else {
			existing = existing + "\n\n" + trimmed
		}
	}
	return nil
}

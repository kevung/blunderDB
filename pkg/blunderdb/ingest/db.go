package ingest

import (
	"context"
	"errors"
	"fmt"
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

	source, err := sqlite.Open(ctx, src.Path, nil)
	if err != nil {
		return Summary{}, fmt.Errorf("ingest: open source .db: %w", err)
	}
	defer source.Close()

	// Drain the source position list before issuing per-position follow-up
	// queries: nesting them inside the List iterator would hold one pooled
	// connection while grabbing another (see JSONExporter).
	var positions []*domain.Position
	for p, err := range source.Positions().List(ctx, scope, storage.ListOpts{}) {
		if err != nil {
			return Summary{}, fmt.Errorf("ingest: list source positions: %w", err)
		}
		pc := *p
		positions = append(positions, &pc)
	}

	// Collect each source position's analysis and comments (source reads only).
	type srcRecord struct {
		pos      *domain.Position
		analysis *domain.PositionAnalysis
		comments []string
	}
	records := make([]srcRecord, 0, len(positions))
	for _, p := range positions {
		rec := srcRecord{pos: p}
		switch a, err := source.Analyses().Load(ctx, scope, p.ID); {
		case err == nil:
			rec.analysis = a
		case errors.Is(err, storage.ErrNotFound):
		default:
			return Summary{}, fmt.Errorf("ingest: load source analysis: %w", err)
		}
		for c, err := range source.Comments().ByPosition(ctx, scope, p.ID) {
			if err != nil {
				return Summary{}, fmt.Errorf("ingest: read source comments: %w", err)
			}
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

	var sum Summary
	for _, rec := range records {
		if err := ctx.Err(); err != nil {
			return sum, err
		}
		pc := *rec.pos
		pc.ID = 0
		id, err := tx.Positions().Save(ctx, scope, &pc)
		if err != nil {
			return sum, err
		}
		if rec.analysis != nil {
			if err := mergeDBAnalysis(ctx, tx, scope, id, rec.analysis); err != nil {
				return sum, err
			}
		}
		if err := mergeDBComments(ctx, tx, scope, id, rec.comments); err != nil {
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

// mergeDBAnalysis writes an imported analysis for positionID following the
// legacy "prefer existing non-empty analysis" rule.
func mergeDBAnalysis(ctx context.Context, tx storage.Tx, scope string, positionID int64, imported *domain.PositionAnalysis) error {
	existing, err := tx.Analyses().Load(ctx, scope, positionID)
	switch {
	case errors.Is(err, storage.ErrNotFound):
		return tx.Analyses().Save(ctx, scope, positionID, imported)
	case err != nil:
		return err
	default:
		if existing.AnalysisType == "" && imported.AnalysisType != "" {
			return tx.Analyses().Save(ctx, scope, positionID, imported)
		}
		return nil
	}
}

// mergeDBComments appends each imported comment to positionID unless the
// position's existing comment text already contains it.
func mergeDBComments(ctx context.Context, tx storage.Tx, scope string, positionID int64, comments []string) error {
	if len(comments) == 0 {
		return nil
	}
	existing, err := tx.Comments().Text(ctx, scope, positionID)
	if err != nil {
		return err
	}
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

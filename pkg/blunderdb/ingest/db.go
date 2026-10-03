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
// positions plus their analysis and comments, and the collections that group
// them — into the target Storage. It is
// the backend-agnostic counterpart of database.CommitImportDatabase: a
// Storage→Storage merge of the position library only, no match/game/move rows.
//
// Merge semantics:
//   - positions dedup by content (PositionStore.Save's Zobrist index);
//   - an imported analysis is written only when the target has none, or when
//     the target's analysis is empty-typed and the import's is not;
//   - a comment is appended only when the target doesn't already contain it;
//   - a collection merges into the target's collection of the same name, or is
//     created; its members are added in the source order, those already there
//     kept where they are. A living collection's query is copied only when the
//     collection is created: the target's own query is the target's.
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

	// Source analyses and comments, batched: one round trip per family, not
	// one per position.
	srcAnalyses, err := source.Analyses().LoadMany(ctx, scope, ids)
	if err != nil {
		return Summary{}, fmt.Errorf("ingest: load source analyses: %w", err)
	}
	srcComments, err := source.Comments().ByPositions(ctx, scope, ids)
	if err != nil {
		return Summary{}, fmt.Errorf("ingest: read source comments: %w", err)
	}

	srcCollections, err := readSourceCollections(ctx, source, scope)
	if err != nil {
		return Summary{}, err
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
	targetOf := make(map[int64]int64, len(records))
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
		targetOf[rec.pos.ID] = id
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

	n, err := mergeDBCollections(ctx, tx, scope, srcCollections, targetOf)
	if err != nil {
		return sum, err
	}
	sum.Collections = n

	if err := ctx.Err(); err != nil {
		return sum, err
	}
	if err := tx.Commit(); err != nil {
		return sum, err
	}
	committed = true
	return sum, nil
}

// srcCollection is a source collection with its members' source position ids,
// in collection order.
type srcCollection struct {
	coll    storage.Collection
	members []int64
}

// readSourceCollections drains the source's collections, then each one's
// members: the List iterator is closed before Members opens another query.
func readSourceCollections(ctx context.Context, source storage.Storage, scope string) ([]srcCollection, error) {
	var out []srcCollection
	for c, err := range source.Collections().List(ctx, scope) {
		if err != nil {
			return nil, fmt.Errorf("ingest: list source collections: %w", err)
		}
		out = append(out, srcCollection{coll: *c})
	}
	for i := range out {
		for m, err := range source.Collections().Members(ctx, scope, out[i].coll.ID) {
			if err != nil {
				return nil, fmt.Errorf("ingest: read source collection %q: %w", out[i].coll.Name, err)
			}
			out[i].members = append(out[i].members, m.PositionID)
		}
	}
	return out, nil
}

// mergeDBCollections writes the source collections into the target, their
// members remapped through targetOf (source position id → id Save returned).
// It reports how many collections it wrote.
func mergeDBCollections(ctx context.Context, tx storage.Tx, scope string, src []srcCollection, targetOf map[int64]int64) (int, error) {
	if len(src) == 0 {
		return 0, nil
	}
	byName := map[string]int64{}
	for c, err := range tx.Collections().List(ctx, scope) {
		if err != nil {
			return 0, err
		}
		byName[c.Name] = c.ID
	}
	for _, sc := range src {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		id, ok := byName[sc.coll.Name]
		if !ok {
			var err error
			if id, err = tx.Collections().Create(ctx, scope, sc.coll.Name, sc.coll.Description); err != nil {
				return 0, err
			}
			if sc.coll.FilterQuery != "" {
				if err := tx.Collections().SetFilterQuery(ctx, scope, id, sc.coll.FilterQuery); err != nil {
					return 0, err
				}
			}
			byName[sc.coll.Name] = id
		}
		ids := make([]int64, 0, len(sc.members))
		for _, m := range sc.members {
			if t, ok := targetOf[m]; ok {
				ids = append(ids, t)
			}
		}
		if len(ids) > 0 {
			if err := tx.Collections().AddPositions(ctx, scope, id, ids); err != nil {
				return 0, err
			}
		}
	}
	return len(src), nil
}

// mergeDBAnalysisPreloaded writes an imported analysis for positionID as
// domain.MergeImportedAnalysis decides. existing is the target's current
// analysis, already loaded in Import's batched pass.
func mergeDBAnalysisPreloaded(ctx context.Context, tx storage.Tx, scope string, positionID int64, existing, imported *domain.PositionAnalysis) error {
	merged, changed := domain.MergeImportedAnalysis(existing, imported)
	if !changed {
		return nil
	}
	return tx.Analyses().Save(ctx, scope, positionID, merged)
}

// mergeDBCommentsPreloaded appends each imported comment to positionID unless
// the position's existing comment text already contains it. existing is the
// target's current comment entries, already loaded in Import's batched pass.
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

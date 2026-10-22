package ingest

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/mets"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// DBImporter imports a native blunderDB .db file's position library —
// positions plus their analysis and comments, and the collections that group
// them, the matches with their games and moves, tournaments, drafts and
// aliases — into the target Storage. It is the backend-agnostic counterpart
// of database.CommitImportDatabase, with which it shares MergeMatches,
// MergeCollections, MergeLessons and MergeDecks.
//
// Merge semantics:
//   - positions dedup by content (PositionStore.Save's Zobrist index);
//   - an imported analysis is written only when the target has none, or when
//     the target's analysis is empty-typed and the import's is not;
//   - a comment is appended only when the target doesn't already contain it;
//   - matches, tournaments, drafts and aliases follow MergeMatches;
//   - collections follow MergeCollections, the rule the desktop import shares.
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
	srcComments, err := source.FileComments(ctx).ByPositions(ctx, scope, ids)
	if err != nil {
		return Summary{}, fmt.Errorf("ingest: read source comments: %w", err)
	}

	srcCollections, err := ReadSourceCollections(ctx, source, scope)
	if err != nil {
		return Summary{}, err
	}
	// A source from before Lessons existed (schema < 2.29.0) has none.
	var srcLessons []*domain.Lesson
	if source.HasTable(ctx, "lesson") {
		if srcLessons, err = ReadSourceLessons(ctx, source, scope); err != nil {
			return Summary{}, err
		}
	}

	var srcDecks []SourceDeck
	if source.HasTable(ctx, "anki_deck") {
		if srcDecks, err = ReadSourceDecks(ctx, source, scope); err != nil {
			return Summary{}, err
		}
	}

	// The tables the source's analyses cite travel with them (ADR-0068); a
	// source from before them (schema < 2.31.0) cites none.
	var srcTables []*domain.MatchEquityTable
	srcMET := map[int64]int64{}
	if source.HasTable(ctx, "match_equity_table") {
		if srcTables, err = mets.ReadTables(ctx, source.MatchEquityTables(), scope); err != nil {
			return Summary{}, fmt.Errorf("ingest: read source match equity tables: %w", err)
		}
		if srcMET, err = source.MatchEquityTables().OfAnalyses(ctx, scope, ids); err != nil {
			return Summary{}, fmt.Errorf("ingest: read source analyses' match equity tables: %w", err)
		}
	}

	type srcRecord struct {
		pos      *domain.Position
		analysis *domain.PositionAnalysis
		comments []*domain.CommentEntry
	}
	srcVersion, err := source.Version(ctx)
	if err != nil {
		return Summary{}, fmt.Errorf("ingest: read source version: %w", err)
	}
	records := make([]srcRecord, 0, len(positions))
	for _, p := range positions {
		rec := srcRecord{pos: p, analysis: srcAnalyses[p.ID]}
		if StaleResponseVerdict(srcVersion, p, rec.analysis) {
			rec.analysis = nil
		}
		rec.comments = srcComments[p.ID]
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
	// snapshot of the target's existing comments taken once, before any of
	// this batch's merges run, is equivalent to querying it fresh for each
	// one. Analyses are not snapshotted: each is merged under its row lock,
	// since a rollout may be written concurrently.
	type savedRecord struct {
		rec *srcRecord
		id  int64
	}
	saved := make([]savedRecord, 0, len(records))
	targetIDs := make([]int64, 0, len(records))
	targetOf := make(map[int64]int64, len(records))
	var created []int64
	for i := range records {
		if err := ctx.Err(); err != nil {
			return Summary{}, err
		}
		rec := &records[i]
		pc := *rec.pos
		pc.ID = 0
		id, inserted, err := tx.Positions().SaveCreated(ctx, scope, &pc)
		if err != nil {
			return Summary{}, err
		}
		if inserted {
			created = append(created, id)
		}
		saved = append(saved, savedRecord{rec: rec, id: id})
		targetIDs = append(targetIDs, id)
		targetOf[rec.pos.ID] = id
	}

	targetComments, err := tx.Comments().ByPositions(ctx, scope, targetIDs)
	if err != nil {
		return Summary{}, err
	}

	carrier := mets.NewCarrier("import", tx.MatchEquityTables(), scope, srcTables)
	sum := Summary{CreatedPositions: created}
	for _, s := range saved {
		if err := ctx.Err(); err != nil {
			return sum, err
		}
		if s.rec.analysis != nil {
			met, err := carrier.Target(ctx, srcMET[s.rec.pos.ID])
			if err != nil {
				return sum, err
			}
			if _, err := MergeDBAnalysis(ctx, tx, scope, s.id, s.rec.analysis, met); err != nil {
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

	matches, err := MergeMatches(ctx, tx, scope, source, targetOf)
	if err != nil {
		return sum, err
	}
	sum.applyMatchMerge(matches)
	maps := RowMaps{Positions: targetOf, Matches: matches.MatchMap, Tournaments: matches.TournamentMap}

	merged, err := MergeCollections(ctx, tx, scope, srcCollections, maps)
	if err != nil {
		return sum, err
	}
	sum.Collections = merged.Changed
	sum.CreatedCollections = merged.Created
	sum.AddedMembers = merged.Added
	sum.LivingCollectionsSkipped = merged.LivingSkipped
	sum.LivingCollectionsFrozen = merged.LivingFrozen
	sum.LivingCollectionsOnManual = merged.LivingOnManual
	if sum.Lessons, err = MergeLessons(ctx, tx, scope, srcLessons, srcCollections, targetOf); err != nil {
		return sum, err
	}
	if sum.Decks, err = MergeDecks(ctx, tx, scope, srcDecks, srcCollections, maps); err != nil {
		return sum, err
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

// SourceCollection is a collection of a native .db being imported, with its
// members' source position ids in collection order.
type SourceCollection struct {
	Coll    storage.Collection
	Members []int64
	// Evaluated is, for a living collection whose query names rows by id,
	// what the query selects at the source (source position ids): the
	// members it freezes into when its query cannot be carried.
	Evaluated []int64
}

// namesRows reports whether a living query names rows by id — `like<n>`,
// `id<n>`, `ma<n>`, `tn<n>` — and so selects something else once those rows
// are numbered by another library. With no row to carry it through, such a
// query cannot be carried; any other comes back as written.
func namesRows(query string) bool {
	_, ok := carriedLivingQuery(query, nil, nil, nil)
	return !ok
}

// ReadSourceCollections drains a source's collections, then each one's
// members: the List iterator is closed before Members opens another query.
func ReadSourceCollections(ctx context.Context, source storage.Stores, scope string) ([]SourceCollection, error) {
	var out []SourceCollection
	for c, err := range source.Collections().List(ctx, scope) {
		if err != nil {
			return nil, fmt.Errorf("ingest: list source collections: %w", err)
		}
		out = append(out, SourceCollection{Coll: *c})
	}
	for i := range out {
		for m, err := range source.Collections().Members(ctx, scope, out[i].Coll.ID) {
			if err != nil {
				return nil, fmt.Errorf("ingest: read source collection %q: %w", out[i].Coll.Name, err)
			}
			out[i].Members = append(out[i].Members, m.PositionID)
		}
		if q := strings.TrimSpace(out[i].Coll.FilterQuery); q != "" && namesRows(q) {
			ev, err := storage.EvaluateCollection(ctx, source, scope, out[i].Coll.ID, 0)
			if err != nil {
				// A query the source cannot evaluate selects nothing there.
				slog.Warn("import: living collection not evaluable at the source",
					"collection", out[i].Coll.Name, "err", err)
				continue
			}
			out[i].Evaluated = ev.PositionIDs
		}
	}
	return out, nil
}

// CollectionMerge says what MergeCollections changed in the target.
type CollectionMerge struct {
	// Changed counts the collections created or given new members.
	Changed int
	// Created are the ids of the collections this merge inserted; Added the
	// members it gave, by target collection id, to those it found held.
	Created []int64
	Added   map[int64][]int64
	// LivingSkipped names the source collections with members whose
	// namesake in the target is living: its membership is its query, so
	// those members are not added.
	LivingSkipped []string
	// LivingFrozen names the living collections created as hand-made lists
	// of what their query selected at the source: it names a row the import
	// did not carry, so in the target it would select something else.
	LivingFrozen []string
	// LivingOnManual names the living source collections whose namesake in
	// the target is a hand-made list: that list is left as it was.
	LivingOnManual []string
}

// MergeCollections is the one rule both native .db imports (the daemon's
// DBImporter and the desktop's Database.CommitImportDatabase) apply: a source
// collection merges into the target's collection of the same name, or is
// created with the source's description and, when living, its query, its row
// ids renumbered through maps — or, when they cannot be, frozen into what it
// selected at the source. Members
// are remapped through maps.Positions (source position id → target id); those the
// target already holds keep their place and are not rewritten, so importing
// the same file twice changes nothing. A living target collection receives no
// members: a living collection stores none (storage.Collection.FilterQuery).
// A living source collection whose namesake is a hand-made list changes
// nothing either, and is named in LivingOnManual.
func MergeCollections(ctx context.Context, tx storage.Stores, scope string, src []SourceCollection, maps RowMaps) (CollectionMerge, error) {
	targetOf := maps.Positions
	var res CollectionMerge
	if len(src) == 0 {
		return res, nil
	}
	byName := map[string]storage.Collection{}
	for c, err := range tx.Collections().List(ctx, scope) {
		if err != nil {
			return res, err
		}
		byName[c.Name] = *c
	}
	for _, sc := range src {
		if err := ctx.Err(); err != nil {
			return res, err
		}
		target, exists := byName[sc.Coll.Name]
		held := map[int64]bool{}
		if exists {
			if target.FilterQuery != "" {
				if len(sc.Members) > 0 {
					res.LivingSkipped = append(res.LivingSkipped, sc.Coll.Name)
				}
				continue
			}
			if strings.TrimSpace(sc.Coll.FilterQuery) != "" {
				// A living source stores no members, and pouring what its
				// query selects into the recipient's own list would rewrite
				// a list the user made: the list is left, the import says so.
				res.LivingOnManual = append(res.LivingOnManual, sc.Coll.Name)
				continue
			}
			for m, err := range tx.Collections().Members(ctx, scope, target.ID) {
				if err != nil {
					return res, err
				}
				held[m.PositionID] = true
			}
		} else {
			id, err := tx.Collections().Create(ctx, scope, sc.Coll.Name, sc.Coll.Description)
			if err != nil {
				return res, err
			}
			target = storage.Collection{ID: id, Name: sc.Coll.Name}
			res.Created = append(res.Created, id)
			members := sc.Members
			if q := strings.TrimSpace(sc.Coll.FilterQuery); q != "" {
				if carried, ok := carriedLivingQuery(q, targetOf, maps.Matches, maps.Tournaments); ok {
					if err := tx.Collections().SetFilterQuery(ctx, scope, id, carried); err != nil {
						return res, err
					}
					target.FilterQuery = carried
				} else {
					members = sc.Evaluated
					res.LivingFrozen = append(res.LivingFrozen, sc.Coll.Name)
				}
			}
			sc.Members = members
			byName[sc.Coll.Name] = target
		}
		var ids []int64
		if target.FilterQuery == "" {
			for _, m := range sc.Members {
				if t, ok := targetOf[m]; ok && !held[t] {
					held[t] = true
					ids = append(ids, t)
				}
			}
		}
		if len(ids) > 0 {
			if err := tx.Collections().AddPositions(ctx, scope, target.ID, ids); err != nil {
				return res, err
			}
			if exists {
				if res.Added == nil {
					res.Added = map[int64][]int64{}
				}
				res.Added[target.ID] = append(res.Added[target.ID], ids...)
			}
		}
		if !exists || len(ids) > 0 {
			res.Changed++
		}
	}
	return res, nil
}

// MergeDBAnalysis writes an imported analysis for positionID as
// domain.MergeImportedAnalysis decides, and reports whether it wrote. Every
// path that brings a library's analyses into another goes through it, so the
// columns derived from the analysis (errors, forced, rates, engine) are the
// store's, whichever path carried them. The read goes through Merge, which
// locks the row (or takes the analysis guard when there is none) until the
// write: a rollout committed between a plain read and the write would be
// overwritten.
func MergeDBAnalysis(ctx context.Context, tx storage.Stores, scope string, positionID int64, imported *domain.PositionAnalysis, importedMET int64) (bool, error) {
	importedSide := mets.SideOf(imported, importedMET)
	var met, existingMET int64
	var readErr error
	wrote, err := tx.Analyses().Merge(ctx, scope, positionID, nil, func(existing *domain.PositionAnalysis) *domain.PositionAnalysis {
		// Read under Merge's lock, so the table belongs to the row merged.
		if existingMET, readErr = tx.MatchEquityTables().OfAnalysis(ctx, scope, positionID); readErr != nil {
			return nil
		}
		existingSide := mets.SideOf(existing, existingMET)
		merged, changed := domain.MergeImportedAnalysis(existing, imported)
		if !changed {
			return nil
		}
		met = mets.AfterMerge(merged, existingSide, importedSide)
		return merged
	})
	if err == nil {
		err = readErr
	}
	if err != nil || !wrote || (met == 0 && existingMET == 0) {
		return wrote, err
	}
	return true, tx.MatchEquityTables().TagAnalyses(ctx, scope, met, []int64{positionID})
}

// mergeDBCommentsPreloaded appends each imported comment to positionID unless
// the position's existing comment text already contains it. existing is the
// target's current comment entries, already loaded in Import's batched pass.
// Each appended comment keeps the source's author, never the importer's, its
// origin — a remark an importer brought the producer does not become the
// recipient's note — and the signature the producer sealed, verbatim: the
// source database states who wrote it (ADR-0007, ADR-0099). Checking it is the reader's business; an
// altered note is kept and shown unverified, never refused here.
func mergeDBCommentsPreloaded(ctx context.Context, tx storage.Tx, scope string, positionID int64, existingEntries []*domain.CommentEntry, comments []*domain.CommentEntry) error {
	if len(comments) == 0 {
		return nil
	}
	parts := make([]string, len(existingEntries))
	for i, e := range existingEntries {
		parts[i] = e.Text
	}
	existing := strings.Join(parts, "\n\n")
	for _, c := range comments {
		text := c.Text
		trimmed := strings.TrimSpace(text)
		if trimmed == "" || strings.Contains(existing, trimmed) {
			continue
		}
		if _, err := tx.Comments().AddFrom(storage.WithCarriedComment(ctx, c.Author, c.Signature), scope, positionID, text, c.Origin); err != nil {
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

package ingest

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// ExportPlan is what an export of the same options would write, counted
// before anything is written: the export dialog shows it, the CLI prints it,
// the daemon answers it. Its counts are those of the ExportReport the export
// would return — the plan walks the same resolved selection (resolveSelection)
// and keeps a row by the rule the writer applies to it.
type ExportPlan struct {
	Positions      PlanFamily `json:"positions"`
	Comments       PlanFamily `json:"comments"`
	Matches        PlanFamily `json:"matches"`
	Tournaments    PlanFamily `json:"tournaments"`
	Collections    PlanFamily `json:"collections"`
	Lessons        PlanFamily `json:"lessons"`
	AnkiDecks      PlanFamily `json:"ankiDecks"`
	Transcriptions PlanFamily `json:"transcriptions"`
	Aliases        PlanFamily `json:"aliases"`
	Filters        PlanFamily `json:"filters"`
	// FrozenCollections are the LIVING collections whose query names a row the
	// file will not hold: they leave as the plain list their query selects now.
	FrozenCollections []PlannedCollection `json:"frozenCollections"`
}

// PlanFamily counts one family of an export. IDs are the source's own ids of
// the rows that leave, so a count can open them; Comments lists the positions
// that carry them. Names stand for the families that have no id worth
// opening: aliases (as "alias → canonical") and saved filters.
type PlanFamily struct {
	Count int      `json:"count"`
	IDs   []int64  `json:"ids"`
	Names []string `json:"names"`
}

// PlannedCollection names a collection of the plan.
type PlannedCollection struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func (f *PlanFamily) addID(id int64) {
	f.Count++
	f.IDs = append(f.IDs, id)
}

// PlanExport counts what ExportSQLite would write for opts, writing nothing.
// The maps from source ids to file ids are identities here: a row "has a
// number in the file" exactly when it is planned, which is all the living
// queries, filters, aliases and drafts ask of them.
func PlanExport(ctx context.Context, src storage.Storage, scope string, opts ExportOptions) (ExportPlan, error) {
	e := &exporter{
		ctx: ctx, src: src, scope: scope, opts: opts,
		posMap: make(map[int64]int64), written: make(map[int64]bool),
		collMap: make(map[int64]int64), tourMap: make(map[int64]int64), matchMap: make(map[int64]int64),
		livingEvals: make(map[int64]*storage.CollectionEvaluation),
	}
	if err := e.resolveSelection(); err != nil {
		return ExportPlan{}, err
	}
	plan := ExportPlan{FrozenCollections: []PlannedCollection{}}
	for _, step := range []func(*ExportPlan) error{
		e.planPositions,
		e.planComments,
		e.planTournaments,
		e.planMatches,
		e.planCollections,
		e.planLessons,
		e.planAliases,
		e.planTranscriptions,
		e.planFilters,
		e.planAnkiDecks,
	} {
		if err := ctx.Err(); err != nil {
			return ExportPlan{}, err
		}
		if err := step(&plan); err != nil {
			return ExportPlan{}, err
		}
	}
	return plan, nil
}

// planPositions keeps the positions that exist, once each, in writePositions'
// order. Positions the caller supplied whole are written whether stored or not;
// one without an id has no number to open but is counted.
func (e *exporter) planPositions(plan *ExportPlan) error {
	sel := e.opts.Selection
	keep := func(id int64) {
		if _, done := e.posMap[id]; done {
			return
		}
		e.posMap[id] = id
		plan.Positions.addID(id)
	}
	existing := func(ids []int64) error {
		for start := 0; start < len(ids); start += positionBatch {
			loaded, err := e.src.Positions().LoadByIDs(e.ctx, e.scope, ids[start:min(start+positionBatch, len(ids))])
			if err != nil {
				return fmt.Errorf("cannot read the positions to export: %w", err)
			}
			for i := range loaded {
				keep(loaded[i].ID)
			}
		}
		return nil
	}
	switch {
	case sel.AllPositions:
		ids, err := e.src.Positions().ListIDs(e.ctx, e.scope, storage.ListOpts{})
		if err != nil {
			return fmt.Errorf("ingest: list positions: %w", err)
		}
		for _, id := range ids {
			keep(id)
		}
	case len(sel.Positions) > 0:
		for _, p := range sel.Positions {
			if p.ID > 0 {
				keep(p.ID)
			} else {
				plan.Positions.Count++
			}
		}
	default:
		if err := existing(sel.PositionIDs); err != nil {
			return err
		}
	}
	return existing(e.extraPositionIDs)
}

func (e *exporter) planComments(plan *ExportPlan) error {
	if !e.opts.Comments {
		return nil
	}
	ids := plan.Positions.IDs
	for start := 0; start < len(ids); start += positionBatch {
		batch := ids[start:min(start+positionBatch, len(ids))]
		comments, err := e.src.Comments().ByPositions(e.ctx, e.scope, batch)
		if err != nil {
			return fmt.Errorf("cannot read comments to export: %w", err)
		}
		for _, id := range batch {
			if n := len(comments[id]); n > 0 {
				plan.Comments.Count += n
				plan.Comments.IDs = append(plan.Comments.IDs, id)
			}
		}
	}
	return nil
}

func (e *exporter) planTournaments(plan *ExportPlan) error {
	for _, id := range e.tournamentIDs {
		if _, err := e.src.Tournaments().Get(e.ctx, e.scope, id); err != nil {
			continue
		}
		e.tourMap[id] = id
		plan.Tournaments.addID(id)
	}
	return nil
}

func (e *exporter) planMatches(plan *ExportPlan) error {
	for _, id := range e.matchIDs {
		m, err := e.src.Matches().Get(e.ctx, e.scope, id)
		if err != nil {
			continue
		}
		e.matchMap[id] = id
		e.noteMatchNames(m)
		plan.Matches.addID(id)
	}
	return nil
}

// planCollections also settles each LIVING collection as writeLivingQueries
// does: once every family has its numbers, a query that cannot say in the
// file what it says here marks the collection frozen.
func (e *exporter) planCollections(plan *ExportPlan) error {
	for _, id := range e.collectionIDs {
		c, err := e.src.Collections().Get(e.ctx, e.scope, id)
		if err != nil {
			continue
		}
		e.collMap[id] = id
		plan.Collections.addID(id)
		if query := strings.TrimSpace(c.FilterQuery); query != "" {
			if _, ok := carriedLivingQuery(query, e.posMap, e.matchMap, e.tourMap); !ok {
				plan.FrozenCollections = append(plan.FrozenCollections, PlannedCollection{ID: id, Name: c.Name})
			}
		}
	}
	return nil
}

func (e *exporter) planLessons(plan *ExportPlan) error {
	for _, id := range e.lessonIDs {
		if _, err := e.src.Lessons().Get(e.ctx, e.scope, id); err != nil {
			continue
		}
		plan.Lessons.addID(id)
	}
	return nil
}

func (e *exporter) planAliases(plan *ExportPlan) error {
	if !e.opts.Aliases || len(e.matchNames) == 0 {
		return nil
	}
	for _, kind := range []storage.AliasKind{storage.AliasPlayer, storage.AliasEvent} {
		aliases, err := e.src.Aliases().List(e.ctx, e.scope, kind)
		if err != nil {
			return fmt.Errorf("ingest: list %s aliases: %w", kind, err)
		}
		names := e.matchNames[kind]
		for _, a := range aliases {
			if !names[a.Alias] && !names[a.Canonical] {
				continue
			}
			plan.Aliases.Count++
			plan.Aliases.Names = append(plan.Aliases.Names, a.Alias+" → "+a.Canonical)
		}
	}
	return nil
}

func (e *exporter) planTranscriptions(plan *ExportPlan) error {
	if !e.opts.Transcriptions {
		return nil
	}
	sel := e.opts.Selection
	unattached := sel.AllMatches && sel.AllPositions
	var drafts []*storage.Transcription
	for tr, err := range e.src.Transcriptions().List(e.ctx, e.scope) {
		if err != nil {
			return fmt.Errorf("ingest: list transcriptions: %w", err)
		}
		d := *tr
		drafts = append(drafts, &d)
	}
	for _, d := range drafts {
		if _, exported := e.matchMap[d.MatchID]; !exported && !unattached {
			continue
		}
		if _, err := carriedTranscriptDocument(d.Document); err != nil {
			continue
		}
		plan.Transcriptions.addID(d.ID)
	}
	return nil
}

func (e *exporter) planFilters(plan *ExportPlan) error {
	if !e.opts.FilterLibrary {
		return nil
	}
	for f, err := range e.src.Filters().List(e.ctx, e.scope) {
		if err != nil {
			return fmt.Errorf("ingest: list filters: %w", err)
		}
		if _, ok := carriedLivingQuery(f.Command, e.posMap, e.matchMap, e.tourMap); !ok {
			continue
		}
		plan.Filters.Count++
		plan.Filters.Names = append(plan.Filters.Names, f.Name)
	}
	return nil
}

func (e *exporter) planAnkiDecks(plan *ExportPlan) error {
	if !e.opts.AnkiDecks && len(e.deckIDs) == 0 {
		return nil
	}
	picked := toSet(e.deckIDs)
	var ids []int64
	for d, err := range e.src.Anki().ListDecks(e.ctx, e.scope) {
		if err != nil {
			return fmt.Errorf("ingest: list decks: %w", err)
		}
		if e.opts.AnkiDecks || picked[d.ID] {
			ids = append(ids, d.ID)
		}
	}
	for _, id := range ids {
		plan.AnkiDecks.addID(id)
	}
	return nil
}

// planned gives the report's counts of the families a plan counts.
func (r ExportReport) planned() map[string]int {
	return map[string]int{
		"positions": r.Positions, "comments": r.Comments, "matches": r.Matches,
		"tournaments": r.Tournaments, "collections": r.Collections, "lessons": r.Lessons,
		"ankiDecks": r.AnkiDecks, "transcriptions": r.Transcriptions, "aliases": r.Aliases,
		"filters": r.Filters,
	}
}

// Counts gives the plan's counts under the names ExportReport.planned uses.
func (p ExportPlan) Counts() map[string]int {
	return map[string]int{
		"positions": p.Positions.Count, "comments": p.Comments.Count, "matches": p.Matches.Count,
		"tournaments": p.Tournaments.Count, "collections": p.Collections.Count, "lessons": p.Lessons.Count,
		"ankiDecks": p.AnkiDecks.Count, "transcriptions": p.Transcriptions.Count, "aliases": p.Aliases.Count,
		"filters": p.Filters.Count,
	}
}

// Mismatch names the first family whose planned count differs from what the
// export wrote, or "" when the plan held. It is how the parity test and a
// caller that wants to check its preview read the two side by side.
func (p ExportPlan) Mismatch(r ExportReport) string {
	want := r.planned()
	keys := make([]string, 0, len(want))
	for k := range want {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		if got := p.Counts()[k]; got != want[k] {
			return fmt.Sprintf("%s: planned %d, wrote %d", k, got, want[k])
		}
	}
	return ""
}

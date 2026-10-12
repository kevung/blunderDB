package ingest

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// ExportPlan is what an export of the same options would write, counted
// before anything is written: the export dialog shows it, the CLI prints it,
// the daemon answers it. Its counts are those of the ExportReport the export
// would return — the plan walks the same resolved selection (resolveSelection)
// and keeps a row by the rule the writer applies to it.
//
// It holds counts only, so its size does not grow with the library; the ids
// behind one count are asked for with PlanExportIDs, a page at a time.
type ExportPlan struct {
	Positions   int `json:"positions"`
	Comments    int `json:"comments"`
	Matches     int `json:"matches"`
	Tournaments int `json:"tournaments"`
	// Directions are the tournaments, among those planned, whose Direction
	// leaves with them.
	Directions     int `json:"directions"`
	Collections    int `json:"collections"`
	Lessons        int `json:"lessons"`
	AnkiDecks      int `json:"ankiDecks"`
	Transcriptions int `json:"transcriptions"`
	Aliases        int `json:"aliases"`
	Filters        int `json:"filters"`
	// FrozenCollections are the LIVING collections whose query names a row the
	// file will not hold: they leave as the plain list their query selects now.
	FrozenCollections []PlannedRow `json:"frozenCollections"`
	// DegradedAnkiDecks are the search decks whose source loses part of what it
	// said on the way (ExportReport.DegradedAnkiDecks).
	DegradedAnkiDecks []PlannedRow `json:"degradedAnkiDecks"`
}

// PlannedRow names a row of the plan. Both of its lists are bounded by the
// number of collections or decks, never by the number of positions.
type PlannedRow struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// The families whose ids PlanExportIDs pages through. Aliases have no id; a
// comment is opened through the position that carries it.
const (
	PlanPositions      = "positions"
	PlanComments       = "comments" // the positions that carry the comments
	PlanMatches        = "matches"
	PlanTournaments    = "tournaments"
	PlanDirections     = "directions" // the tournaments whose Direction leaves
	PlanCollections    = "collections"
	PlanLessons        = "lessons"
	PlanAnkiDecks      = "ankiDecks"
	PlanTranscriptions = "transcriptions"
	PlanFilters        = "filters"
)

// MaxPlannedIDs bounds one page of PlanExportIDs.
const MaxPlannedIDs = 1000

// PlannedIDs is one page of the source ids behind a count of the plan.
type PlannedIDs struct {
	Family string  `json:"family"`
	Total  int     `json:"total"`
	Offset int     `json:"offset"`
	IDs    []int64 `json:"ids"`
}

// PlanExport counts what ExportSQLite would write for opts, writing nothing.
func PlanExport(ctx context.Context, src storage.Storage, scope string, opts ExportOptions) (ExportPlan, error) {
	p, err := newPlanner(ctx, src, scope, opts)
	if err != nil {
		return ExportPlan{}, err
	}
	return p.plan, nil
}

// PlanExportIDs pages through the source ids of one family of opts' plan, so
// a count can open what it counts. limit is capped at MaxPlannedIDs (and
// defaults to it).
func PlanExportIDs(ctx context.Context, src storage.Storage, scope string, opts ExportOptions, family string, offset, limit int) (PlannedIDs, error) {
	if _, ok := plannedFamilies[family]; !ok {
		return PlannedIDs{}, fmt.Errorf("%w: no ids for export family %q", storage.ErrInvalid, family)
	}
	p, err := newPlanner(ctx, src, scope, opts)
	if err != nil {
		return PlannedIDs{}, err
	}
	ids := p.ids[family]
	if limit <= 0 || limit > MaxPlannedIDs {
		limit = MaxPlannedIDs
	}
	offset = max(0, min(offset, len(ids)))
	page := slices.Clone(ids[offset:min(offset+limit, len(ids))])
	if page == nil {
		page = []int64{}
	}
	return PlannedIDs{Family: family, Total: len(ids), Offset: offset, IDs: page}, nil
}

var plannedFamilies = map[string]struct{}{
	PlanPositions: {}, PlanComments: {}, PlanMatches: {}, PlanTournaments: {}, PlanDirections: {}, PlanCollections: {},
	PlanLessons: {}, PlanAnkiDecks: {}, PlanTranscriptions: {}, PlanFilters: {},
}

// planner is one plan's walk. The exporter's maps from source ids to file ids
// are identities here: a row "has a number in the file" exactly when it is
// planned, which is all the living queries, filters, aliases, drafts and deck
// sources ask of them.
type planner struct {
	*exporter
	plan ExportPlan
	ids  map[string][]int64
}

func newPlanner(ctx context.Context, src storage.Storage, scope string, opts ExportOptions) (*planner, error) {
	p := &planner{
		exporter: &exporter{
			ctx: ctx, src: src, scope: scope, opts: opts,
			posMap: make(map[int64]int64), written: make(map[int64]bool),
			collMap: make(map[int64]int64), tourMap: make(map[int64]int64), matchMap: make(map[int64]int64),
			livingEvals: make(map[int64]*storage.CollectionEvaluation),
		},
		plan: ExportPlan{FrozenCollections: []PlannedRow{}, DegradedAnkiDecks: []PlannedRow{}},
		ids:  make(map[string][]int64),
	}
	if err := p.resolveSelection(); err != nil {
		return nil, err
	}
	for _, step := range []func() error{
		p.positions,
		p.comments,
		p.tournaments,
		p.matches,
		p.directions,
		p.collections,
		p.lessons,
		p.aliases,
		p.transcriptions,
		p.filters,
		p.ankiDecks,
	} {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := step(); err != nil {
			return nil, err
		}
	}
	return p, nil
}

func (p *planner) add(family string, count *int, id int64) {
	*count++
	p.ids[family] = append(p.ids[family], id)
}

// positions keeps what writePositions writes: each position once per Zobrist
// hash, as the file's Save deduplicates. The positions of the whole library
// are taken by id alone: a library already holds one row per hash.
func (p *planner) positions() error {
	sel := p.opts.Selection
	hashes := make(map[uint64]bool)
	keep := func(pos *domain.Position) {
		if pos.ID > 0 {
			if _, done := p.posMap[pos.ID]; done {
				return
			}
			p.posMap[pos.ID] = pos.ID
		}
		h := engine.PopulatePositionColumns(pos).ZobristHash
		if hashes[h] {
			return
		}
		hashes[h] = true
		p.plan.Positions++
		if pos.ID > 0 {
			p.ids[PlanPositions] = append(p.ids[PlanPositions], pos.ID)
		}
	}
	existing := func(ids []int64) error {
		for start := 0; start < len(ids); start += positionBatch {
			loaded, err := p.src.Positions().LoadByIDs(p.ctx, p.scope, ids[start:min(start+positionBatch, len(ids))])
			if err != nil {
				return fmt.Errorf("cannot read the positions to export: %w", err)
			}
			for i := range loaded {
				keep(&loaded[i])
			}
		}
		return nil
	}
	switch {
	case sel.AllPositions:
		ids, err := p.src.Positions().ListIDs(p.ctx, p.scope, storage.ListOpts{})
		if err != nil {
			return fmt.Errorf("ingest: list positions: %w", err)
		}
		for _, id := range ids {
			if _, done := p.posMap[id]; !done {
				p.posMap[id] = id
				p.add(PlanPositions, &p.plan.Positions, id)
			}
		}
	case len(sel.Positions) > 0:
		for _, pos := range sel.Positions {
			keep(pos)
		}
	default:
		if err := existing(sel.PositionIDs); err != nil {
			return err
		}
	}
	return existing(p.extraPositionIDs)
}

func (p *planner) comments() error {
	if !p.opts.Comments {
		return nil
	}
	ids := p.ids[PlanPositions]
	for start := 0; start < len(ids); start += positionBatch {
		batch := ids[start:min(start+positionBatch, len(ids))]
		comments, err := p.src.Comments().ByPositions(p.ctx, p.scope, batch)
		if err != nil {
			return fmt.Errorf("cannot read comments to export: %w", err)
		}
		for _, id := range batch {
			if n := len(comments[id]); n > 0 {
				p.plan.Comments += n
				p.ids[PlanComments] = append(p.ids[PlanComments], id)
			}
		}
	}
	return nil
}

func (p *planner) tournaments() error {
	for _, id := range p.tournamentIDs {
		if _, err := p.src.Tournaments().Get(p.ctx, p.scope, id); err != nil {
			continue
		}
		p.tourMap[id] = id
		p.add(PlanTournaments, &p.plan.Tournaments, id)
	}
	return nil
}

// directions keeps what writeDirections writes: the Direction of each planned
// tournament that has one.
func (p *planner) directions() error {
	for _, id := range slices.Sorted(maps.Keys(p.tourMap)) {
		_, err := p.src.Directions().Get(p.ctx, p.scope, id)
		if errors.Is(err, direction.ErrNoDirection) || errors.Is(err, storage.ErrNotFound) {
			continue
		}
		if err != nil {
			return fmt.Errorf("ingest: reading the direction of tournament %d: %w", id, err)
		}
		p.add(PlanDirections, &p.plan.Directions, id)
	}
	return nil
}

func (p *planner) matches() error {
	for _, id := range p.matchIDs {
		m, err := p.src.Matches().Get(p.ctx, p.scope, id)
		if err != nil {
			continue
		}
		p.matchMap[id] = id
		p.noteMatchNames(m)
		p.add(PlanMatches, &p.plan.Matches, id)
	}
	return nil
}

// collections also settles each LIVING collection as writeLivingQueries
// does: once every family has its numbers, a query that cannot say in the
// file what it says here marks the collection frozen.
func (p *planner) collections() error {
	for _, id := range p.collectionIDs {
		c, err := p.src.Collections().Get(p.ctx, p.scope, id)
		if err != nil {
			continue
		}
		p.collMap[id] = id
		p.add(PlanCollections, &p.plan.Collections, id)
		if query := strings.TrimSpace(c.FilterQuery); query != "" {
			if _, ok := carriedLivingQuery(query, p.posMap, p.matchMap, p.tourMap); !ok {
				p.plan.FrozenCollections = append(p.plan.FrozenCollections, PlannedRow{ID: id, Name: c.Name})
			}
		}
	}
	return nil
}

func (p *planner) lessons() error {
	for _, id := range p.lessonIDs {
		if _, err := p.src.Lessons().Get(p.ctx, p.scope, id); err != nil {
			continue
		}
		p.add(PlanLessons, &p.plan.Lessons, id)
	}
	return nil
}

func (p *planner) aliases() error {
	if !p.opts.Aliases || len(p.matchNames) == 0 {
		return nil
	}
	for _, kind := range []storage.AliasKind{storage.AliasPlayer, storage.AliasEvent} {
		aliases, err := p.src.Aliases().List(p.ctx, p.scope, kind)
		if err != nil {
			return fmt.Errorf("ingest: list %s aliases: %w", kind, err)
		}
		names := p.matchNames[kind]
		for _, a := range aliases {
			if names[a.Alias] || names[a.Canonical] {
				p.plan.Aliases++
			}
		}
	}
	return nil
}

func (p *planner) transcriptions() error {
	if !p.opts.Transcriptions {
		return nil
	}
	sel := p.opts.Selection
	unattached := sel.AllMatches && sel.AllPositions
	var drafts []*storage.Transcription
	for tr, err := range p.src.Transcriptions().List(p.ctx, p.scope) {
		if err != nil {
			return fmt.Errorf("ingest: list transcriptions: %w", err)
		}
		d := *tr
		drafts = append(drafts, &d)
	}
	for _, d := range drafts {
		if _, exported := p.matchMap[d.MatchID]; !exported && !unattached {
			continue
		}
		if _, err := carriedTranscriptDocument(d.Document); err != nil {
			continue
		}
		p.add(PlanTranscriptions, &p.plan.Transcriptions, d.ID)
	}
	return nil
}

func (p *planner) filters() error {
	if !p.opts.FilterLibrary {
		return nil
	}
	for f, err := range p.src.Filters().List(p.ctx, p.scope) {
		if err != nil {
			return fmt.Errorf("ingest: list filters: %w", err)
		}
		if _, ok := carriedLivingQuery(f.Command, p.posMap, p.matchMap, p.tourMap); ok {
			p.add(PlanFilters, &p.plan.Filters, f.ID)
		}
	}
	return nil
}

func (p *planner) ankiDecks() error {
	if !p.opts.AnkiDecks && len(p.deckIDs) == 0 {
		return nil
	}
	picked := toSet(p.deckIDs)
	var decks []domain.AnkiDeck
	for d, err := range p.src.Anki().ListDecks(p.ctx, p.scope) {
		if err != nil {
			return fmt.Errorf("ingest: list decks: %w", err)
		}
		if p.opts.AnkiDecks || picked[d.ID] {
			decks = append(decks, *d)
		}
	}
	for _, d := range decks {
		p.add(PlanAnkiDecks, &p.plan.AnkiDecks, d.ID)
		if d.SourceType == domain.AnkiSourceSearch &&
			searchSourceLost(d.SourceCommand, carriedSearchSource(d.SourceCommand, p.posMap, p.matchMap, p.tourMap)) {
			p.plan.DegradedAnkiDecks = append(p.plan.DegradedAnkiDecks, PlannedRow{ID: d.ID, Name: d.Name})
		}
	}
	return nil
}

// Counts gives the plan's counts by family name, the names ExportReport.Counts
// uses.
func (p ExportPlan) Counts() map[string]int {
	return map[string]int{
		PlanPositions: p.Positions, PlanComments: p.Comments, PlanMatches: p.Matches,
		PlanTournaments: p.Tournaments, PlanDirections: p.Directions, PlanCollections: p.Collections, PlanLessons: p.Lessons,
		PlanAnkiDecks: p.AnkiDecks, PlanTranscriptions: p.Transcriptions, "aliases": p.Aliases,
		PlanFilters: p.Filters, "degradedAnkiDecks": len(p.DegradedAnkiDecks),
	}
}

// Counts gives the report's counts of the families a plan counts.
func (r ExportReport) Counts() map[string]int {
	return map[string]int{
		PlanPositions: r.Positions, PlanComments: r.Comments, PlanMatches: r.Matches,
		PlanTournaments: r.Tournaments, PlanDirections: r.Directions, PlanCollections: r.Collections, PlanLessons: r.Lessons,
		PlanAnkiDecks: r.AnkiDecks, PlanTranscriptions: r.Transcriptions, "aliases": r.Aliases,
		PlanFilters: r.Filters, "degradedAnkiDecks": r.DegradedAnkiDecks,
	}
}

// Mismatch names the first family whose planned count differs from what the
// export wrote, or "" when the plan held.
func (p ExportPlan) Mismatch(r ExportReport) string {
	planned, wrote := p.Counts(), r.Counts()
	keys := make([]string, 0, len(wrote))
	for k := range wrote {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		if planned[k] != wrote[k] {
			return fmt.Sprintf("%s: planned %d, wrote %d", k, planned[k], wrote[k])
		}
	}
	return ""
}

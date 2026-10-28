package ingest

import (
	"context"
	"fmt"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// CapPosition is a source position as a capped import plans it: its source
// id, its Zobrist hash, and whether it holds itself in the target once
// carried — individually imported, flagged by the source tool, or bearing a
// comment the user wrote (the self-standing holders of positionIsHeldSQL).
type CapPosition struct {
	ID       int64
	Zobrist  uint64
	SelfHeld bool
}

// CapPlan is what a capped native .db import leaves out, by source id. A row
// is left out whole or carried whole, so that importing the same file again
// without the cap brings it complete instead of finding a truncated namesake
// already held.
type CapPlan struct {
	Positions   map[int64]bool
	Matches     map[int64]bool
	Collections map[int64]bool
	Lessons     map[int64]bool
	Decks       map[int64]bool
}

// Empty reports whether the plan leaves nothing out.
func (p CapPlan) Empty() bool { return len(p.Positions) == 0 }

// PlanCap decides what an import capped at max positions leaves out:
//
//  1. the source positions past the first max, in ascending source id;
//  2. every match, collection, Lesson and Anki deck that reaches one of
//     them — a member, a move, a step, a card, a living query or search
//     source naming one, or a collection or match left out by this rule;
//  3. every position of the first max that nothing would hold in the target
//     once the rest is carried (the retention predicate positionIsHeldSQL):
//     not already stored there, not self-held, and reached by no carried
//     match move, collection membership, Lesson step or deck card. Carried, it
//     would be purged at the next match deletion.
//
// So a capped import carries at most max positions. positions must be sorted
// by ascending id. target is read, never written: the import's transaction.
// max <= 0, or a source within it, leaves nothing out.
func PlanCap(ctx context.Context, source, target storage.Stores, scope string, positions []CapPosition, max int,
	colls []SourceCollection, lessons []*domain.Lesson, decks []SourceDeck) (CapPlan, error) {
	plan := CapPlan{Positions: map[int64]bool{}, Matches: map[int64]bool{}, Collections: map[int64]bool{}, Lessons: map[int64]bool{}, Decks: map[int64]bool{}}
	if max <= 0 || len(positions) <= max {
		return plan, nil
	}
	cut := map[int64]bool{}
	kept := map[int64]int64{}
	for i, p := range positions {
		if i >= max {
			cut[p.ID] = true
		} else {
			kept[p.ID] = p.ID
		}
	}
	all := make(map[int64]int64, len(positions))
	for _, p := range positions {
		all[p.ID] = p.ID
	}

	// Matches: a move on a cut position leaves the match out.
	var matchIDs []int64
	for m, err := range source.Matches().List(ctx, scope, storage.MatchListOpts{}) {
		if err != nil {
			// A source too old for matches carries none (MergeMatches).
			matchIDs = nil
			break
		}
		matchIDs = append(matchIDs, m.ID)
	}
	matchPositions := map[int64][]int64{}
	allMatches, keptMatches := map[int64]int64{}, map[int64]int64{}
	for _, id := range matchIDs {
		if err := ctx.Err(); err != nil {
			return plan, err
		}
		for mv, err := range source.Matches().MovesByMatch(ctx, scope, id) {
			if err != nil {
				return plan, fmt.Errorf("ingest: read moves of source match %d: %w", id, err)
			}
			if mv.PositionID != 0 {
				matchPositions[id] = append(matchPositions[id], mv.PositionID)
			}
			if cut[mv.PositionID] {
				plan.Matches[id] = true
			}
		}
		allMatches[id] = id
		if !plan.Matches[id] {
			keptMatches[id] = id
		}
	}
	tournaments := map[int64]int64{}
	for t, err := range source.Tournaments().List(ctx, scope, storage.ListOpts{}) {
		if err != nil {
			break
		}
		tournaments[t.ID] = t.ID
	}
	// A living query or search source that names a row the cut leaves out
	// would be carried frozen, or emptied, for good.
	losesRow := func(query string) bool {
		_, okAll := carriedLivingQuery(query, all, allMatches, tournaments)
		_, okKept := carriedLivingQuery(query, kept, keptMatches, tournaments)
		return okAll && !okKept
	}
	touches := func(ids []int64) bool {
		for _, id := range ids {
			if cut[id] {
				return true
			}
		}
		return false
	}

	for _, c := range colls {
		q := strings.TrimSpace(c.Coll.FilterQuery)
		if touches(c.Members) || touches(c.Evaluated) || (q != "" && losesRow(q)) {
			plan.Collections[c.Coll.ID] = true
		}
	}
	for _, l := range lessons {
		for _, st := range l.Steps {
			if cut[st.PositionID] || plan.Collections[st.CollectionID] {
				plan.Lessons[l.ID] = true
				break
			}
		}
	}
	for _, d := range decks {
		switch {
		case touches(d.PositionIDs):
		case d.Deck.SourceType == domain.AnkiSourceCollection && plan.Collections[d.Deck.SourceID]:
		case d.Deck.SourceType == domain.AnkiSourceSearch &&
			carriedSearchSource(d.Deck.SourceCommand, all, allMatches, tournaments) != carriedSearchSource(d.Deck.SourceCommand, kept, keptMatches, tournaments):
		default:
			continue
		}
		plan.Decks[d.Deck.ID] = true
	}

	held, err := capHolders(ctx, target, scope, plan, matchPositions, colls, lessons, decks, all, allMatches, tournaments)
	if err != nil {
		return plan, err
	}
	for _, p := range positions {
		if cut[p.ID] {
			plan.Positions[p.ID] = true
			continue
		}
		if p.SelfHeld || held[p.ID] {
			continue
		}
		_, stored, err := target.Positions().Exists(ctx, scope, p.Zobrist)
		if err != nil {
			return plan, err
		}
		if !stored {
			plan.Positions[p.ID] = true
		}
	}
	return plan, nil
}

// capHolders returns the source positions that a carried match, collection,
// Lesson or deck will hold in the target, by MergeMatches', MergeCollections',
// MergeLessons' and MergeDecks' own rules: a namesake the target already has
// takes no Lesson step or deck card, a living target collection no member.
func capHolders(ctx context.Context, target storage.Stores, scope string, plan CapPlan, matchPositions map[int64][]int64,
	colls []SourceCollection, lessons []*domain.Lesson, decks []SourceDeck, all, allMatches, tournaments map[int64]int64) (map[int64]bool, error) {
	held := map[int64]bool{}
	hold := func(ids []int64) {
		for _, id := range ids {
			held[id] = true
		}
	}
	for id, ps := range matchPositions {
		if !plan.Matches[id] {
			hold(ps)
		}
	}

	if len(colls) > 0 {
		targetColl := map[string]storage.Collection{}
		for c, err := range target.Collections().List(ctx, scope) {
			if err != nil {
				return nil, err
			}
			targetColl[c.Name] = *c
		}
		for _, c := range colls {
			if plan.Collections[c.Coll.ID] {
				continue
			}
			q := strings.TrimSpace(c.Coll.FilterQuery)
			t, exists := targetColl[c.Coll.Name]
			switch {
			case exists && (t.FilterQuery != "" || q != ""):
				// A living namesake takes no member; a living source leaves
				// a hand-made namesake as it was.
			case q == "":
				hold(c.Members)
			default:
				if _, ok := carriedLivingQuery(q, all, allMatches, tournaments); !ok {
					hold(c.Evaluated) // frozen into what it selected
				}
			}
		}
	}

	if len(lessons) > 0 {
		names, err := target.Lessons().List(ctx, scope)
		if err != nil {
			return nil, err
		}
		have := map[string]bool{}
		for _, l := range names {
			have[l.Name] = true
		}
		for _, l := range lessons {
			if plan.Lessons[l.ID] || have[l.Name] {
				continue
			}
			for _, st := range l.Steps {
				if st.PositionID != 0 {
					held[st.PositionID] = true
				}
			}
		}
	}

	if len(decks) > 0 {
		have := map[string]bool{}
		for d, err := range target.Anki().ListDecks(ctx, scope) {
			if err != nil {
				return nil, err
			}
			have[d.Name] = true
		}
		for _, d := range decks {
			if !plan.Decks[d.Deck.ID] && !have[d.Deck.Name] {
				hold(d.PositionIDs)
			}
		}
	}
	return held, nil
}

// Keep* return the rows of a family the plan carries.

func (p CapPlan) KeepCollections(colls []SourceCollection) []SourceCollection {
	return keepBy(colls, func(c SourceCollection) bool { return !p.Collections[c.Coll.ID] })
}

func (p CapPlan) KeepLessons(lessons []*domain.Lesson) []*domain.Lesson {
	return keepBy(lessons, func(l *domain.Lesson) bool { return !p.Lessons[l.ID] })
}

func (p CapPlan) KeepDecks(decks []SourceDeck) []SourceDeck {
	return keepBy(decks, func(d SourceDeck) bool { return !p.Decks[d.Deck.ID] })
}

func keepBy[T any](rows []T, keep func(T) bool) []T {
	out := make([]T, 0, len(rows))
	for _, r := range rows {
		if keep(r) {
			out = append(out, r)
		}
	}
	return out
}

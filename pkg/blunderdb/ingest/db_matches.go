package ingest

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// RowMaps are the source → target ids of the rows a native .db import wrote
// or found already held. A living query names rows by id, so it is carried
// through them (carriedLivingQuery).
type RowMaps struct {
	Positions   map[int64]int64
	Matches     map[int64]int64
	Tournaments map[int64]int64
}

// MatchMerge says what MergeMatches changed in the target.
type MatchMerge struct {
	// Matches counts the matches created, each with its games, moves and
	// move analyses; Duplicates those the target already held by fingerprint.
	Matches    int
	Duplicates int
	// Tournaments counts the tournaments created; one whose name the target
	// holds receives the imported matches instead.
	Tournaments    int
	Transcriptions int
	Aliases        int
	// Unreadable names the families the source could not be read for — a
	// source older than them — which the import leaves behind.
	Unreadable []string

	MatchMap      map[int64]int64
	TournamentMap map[int64]int64
}

// MergeMatches is the second half of the native .db rule both imports apply
// (DBImporter and Database.CommitImportDatabase), run once the positions are
// saved and targetOf known:
//   - a tournament merges into the target's tournament of the same name, or
//     is created with the source's date, location and comment;
//   - a match the target already holds — same match_hash, else same
//     canonical_hash — is not written again; any other is created with its
//     games, moves (their positions remapped through targetOf, NULL when the
//     file did not carry them), move analyses and origin, and filed in its
//     tournament in the source's order;
//   - a draft is written unless the target holds the same document, or a
//     draft of the match it produced;
//   - an alias is set unless the target already decides that spelling, or
//     the name is a canonical there: the recipient's choice wins.
//
// Tournaments, drafts and aliases keep the dates their source rows bore.
// Importing the same file twice therefore changes nothing. Nothing is written
// that the file did not carry: no import batch, no date of this import other
// than the stores' own (ADR-0007).
func MergeMatches(ctx context.Context, tx storage.Stores, scope string, source storage.Stores, targetOf map[int64]int64) (MatchMerge, error) {
	res := MatchMerge{MatchMap: map[int64]int64{}, TournamentMap: map[int64]int64{}}
	if err := mergeTournaments(ctx, tx, scope, source, &res); err != nil {
		return res, err
	}
	if err := mergeMatchRows(ctx, tx, scope, source, targetOf, &res); err != nil {
		return res, err
	}
	if err := mergeTranscriptions(ctx, tx, scope, source, &res); err != nil {
		return res, err
	}
	if err := mergeAliases(ctx, tx, scope, source, &res); err != nil {
		return res, err
	}
	return res, nil
}

// ImportMatchPlan is what MergeMatches would do with a source's matches, counted
// without writing: the import preview's figures.
type ImportMatchPlan struct {
	// ToImport counts the matches the import would create; Known those the
	// target already holds by fingerprint.
	ToImport int
	Known    int
	// Unreadable is true when the source has no readable matches family: a
	// source older than it, which the import leaves behind.
	Unreadable bool
}

// PlanImportMatches counts the source's matches by MergeMatches' rule — same
// match_hash, else same canonical_hash, in the target — and nothing else: it
// writes nothing.
func PlanImportMatches(ctx context.Context, target storage.Stores, scope string, source storage.Stores) (ImportMatchPlan, error) {
	var plan ImportMatchPlan
	var ids []int64
	for m, err := range source.Matches().List(ctx, scope, storage.MatchListOpts{}) {
		if err != nil {
			slog.Warn("import preview: source matches unreadable", "err", err)
			plan.Unreadable = true
			return plan, nil
		}
		ids = append(ids, m.ID)
	}
	for _, id := range ids {
		if err := ctx.Err(); err != nil {
			return plan, err
		}
		m, err := source.Matches().Get(ctx, scope, id)
		if err != nil {
			return plan, fmt.Errorf("ingest: read source match %d: %w", id, err)
		}
		// The source's own fingerprints are unique (schema), so only the
		// target can already hold one.
		if m.MatchHash != "" || m.CanonicalHash != "" {
			_, found, err := target.Matches().FindByHash(ctx, scope, m.MatchHash, m.CanonicalHash)
			if err != nil {
				return plan, err
			}
			if found {
				plan.Known++
				continue
			}
		}
		plan.ToImport++
	}
	return plan, nil
}

// unreadable records that the source cannot be read for family: a source
// older than the family is imported without it, never refused for it.
func (res *MatchMerge) unreadable(family string, err error) {
	slog.Warn("import: source family unreadable; left behind", "family", family, "err", err)
	res.Unreadable = append(res.Unreadable, family)
}

func mergeTournaments(ctx context.Context, tx storage.Stores, scope string, source storage.Stores, res *MatchMerge) error {
	var src []*domain.Tournament
	for t, err := range source.Tournaments().List(ctx, scope, storage.ListOpts{}) {
		if err != nil {
			res.unreadable("tournaments", err)
			return nil
		}
		tc := *t
		src = append(src, &tc)
	}
	if len(src) == 0 {
		return nil
	}
	byName := map[string]int64{}
	for t, err := range tx.Tournaments().List(ctx, scope, storage.ListOpts{}) {
		if err != nil {
			return err
		}
		byName[t.Name] = t.ID
	}
	for _, t := range src {
		if id, ok := byName[t.Name]; ok {
			res.TournamentMap[t.ID] = id
			continue
		}
		id, err := tx.Tournaments().Create(ctx, scope, t.Name, t.Date, t.Location)
		if err != nil {
			return fmt.Errorf("ingest: import tournament %q: %w", t.Name, err)
		}
		if t.Comment != "" {
			if err := tx.Tournaments().UpdateComment(ctx, scope, id, t.Comment); err != nil {
				return fmt.Errorf("ingest: import tournament %q comment: %w", t.Name, err)
			}
		}
		if err := tx.Tournaments().KeepDates(ctx, scope, id, t.CreatedAt, t.UpdatedAt); err != nil {
			return fmt.Errorf("ingest: import tournament %q dates: %w", t.Name, err)
		}
		byName[t.Name] = id
		res.TournamentMap[t.ID] = id
		res.Tournaments++
	}
	return nil
}

func mergeMatchRows(ctx context.Context, tx storage.Stores, scope string, source storage.Stores, targetOf map[int64]int64, res *MatchMerge) error {
	var ids []int64
	for m, err := range source.Matches().List(ctx, scope, storage.MatchListOpts{}) {
		if err != nil {
			res.unreadable("matches", err)
			return nil
		}
		ids = append(ids, m.ID)
	}
	src := make([]*domain.Match, 0, len(ids))
	for _, id := range ids {
		m, err := source.Matches().Get(ctx, scope, id)
		if err != nil {
			return fmt.Errorf("ingest: read source match %d: %w", id, err)
		}
		src = append(src, m)
	}
	// AddMatch files a match last in its tournament: writing them in the
	// source's order keeps that order.
	sort.SliceStable(src, func(i, j int) bool {
		return src[i].TournamentSortOrder < src[j].TournamentSortOrder
	})

	originsReadable := true
	if _, err := source.Duels().Origin(ctx, scope, 0); err != nil && !errors.Is(err, storage.ErrNotFound) {
		originsReadable = false
	}

	var created []int64
	for _, m := range src {
		if err := ctx.Err(); err != nil {
			return err
		}
		if m.MatchHash != "" || m.CanonicalHash != "" {
			id, found, err := tx.Matches().FindByHash(ctx, scope, m.MatchHash, m.CanonicalHash)
			if err != nil {
				return err
			}
			if found {
				res.MatchMap[m.ID] = id
				res.Duplicates++
				continue
			}
		}
		id, err := copyMatch(ctx, tx, scope, source, m, targetOf)
		if err != nil {
			return err
		}
		if originsReadable {
			o, err := source.Duels().Origin(ctx, scope, m.ID)
			switch {
			case err == nil:
				oc := *o
				oc.MatchID = id
				if err := tx.Duels().SetOrigin(ctx, scope, &oc); err != nil {
					return fmt.Errorf("ingest: import origin of match %d: %w", m.ID, err)
				}
			case !errors.Is(err, storage.ErrNotFound):
				return fmt.Errorf("ingest: read origin of source match %d: %w", m.ID, err)
			}
		}
		if m.TournamentID != nil {
			if tid, ok := res.TournamentMap[*m.TournamentID]; ok {
				if err := tx.Tournaments().AddMatch(ctx, scope, tid, id); err != nil {
					return fmt.Errorf("ingest: file match %d in its tournament: %w", m.ID, err)
				}
			}
		}
		res.MatchMap[m.ID] = id
		res.Matches++
		created = append(created, id)
	}
	// The per-seat tallies, in the same transaction as the moves: a library
	// never holds a match whose stats a reader must wait for.
	if len(created) > 0 {
		if err := tx.Stats().RefreshMatchStats(ctx, scope, created); err != nil {
			return err
		}
	}
	return nil
}

// copyMatch writes one source match, its games, moves and move analyses into
// the target and returns its new id. Each source stream is drained before the
// next opens: a nested iterator would hold the source's connection.
func copyMatch(ctx context.Context, tx storage.Stores, scope string, source storage.Stores, m *domain.Match, targetOf map[int64]int64) (int64, error) {
	mc := *m
	mc.ID = 0
	mc.TournamentID = nil // filed by AddMatch, through the tournament map
	mc.ImportBatchID = 0  // names an import of the source library
	mc.DiceHash = ""      // derived; the recipient computes it again
	mc.LastVisitedPosition = 0
	id, err := tx.Matches().Save(ctx, scope, &mc)
	if err != nil {
		return 0, fmt.Errorf("ingest: import match %d: %w", m.ID, err)
	}

	var games []*domain.Game
	for g, err := range source.Matches().Games(ctx, scope, m.ID) {
		if err != nil {
			return 0, fmt.Errorf("ingest: read games of source match %d: %w", m.ID, err)
		}
		gc := *g
		games = append(games, &gc)
	}
	gameMap := make(map[int64]int64, len(games))
	for _, g := range games {
		gc := *g
		gc.ID = 0
		gc.MatchID = id
		gid, err := tx.Matches().CreateGame(ctx, scope, &gc)
		if err != nil {
			return 0, fmt.Errorf("ingest: import game of match %d: %w", m.ID, err)
		}
		gameMap[g.ID] = gid
	}

	var moves []*domain.Move
	for mv, err := range source.Matches().MovesByMatch(ctx, scope, m.ID) {
		if err != nil {
			return 0, fmt.Errorf("ingest: read moves of source match %d: %w", m.ID, err)
		}
		mvc := *mv
		moves = append(moves, &mvc)
	}
	moveMap := make(map[int64]int64, len(moves))
	for _, mv := range moves {
		gid, ok := gameMap[mv.GameID]
		if !ok {
			continue
		}
		mvc := *mv
		mvc.ID = 0
		mvc.GameID = gid
		mvc.PositionID = targetOf[mv.PositionID] // 0 → NULL: the file did not carry it
		nid, err := tx.Matches().CreateMove(ctx, scope, &mvc)
		if err != nil {
			return 0, fmt.Errorf("ingest: import move of match %d: %w", m.ID, err)
		}
		moveMap[mv.ID] = nid
	}

	var analyses []*domain.MoveAnalysis
	for ma, err := range source.Matches().MoveAnalysesByMatch(ctx, scope, m.ID) {
		if err != nil {
			return 0, fmt.Errorf("ingest: read move analyses of source match %d: %w", m.ID, err)
		}
		ac := *ma
		analyses = append(analyses, &ac)
	}
	for _, ma := range analyses {
		mid, ok := moveMap[ma.MoveID]
		if !ok {
			continue
		}
		ac := *ma
		ac.ID = 0
		ac.MoveID = mid
		if _, err := tx.Matches().CreateMoveAnalysis(ctx, scope, &ac); err != nil {
			return 0, fmt.Errorf("ingest: import move analysis of match %d: %w", m.ID, err)
		}
	}
	return id, nil
}

func mergeTranscriptions(ctx context.Context, tx storage.Stores, scope string, source storage.Stores, res *MatchMerge) error {
	var src []*storage.Transcription
	for t, err := range source.Transcriptions().List(ctx, scope) {
		if err != nil {
			res.unreadable("transcriptions", err)
			return nil
		}
		tc := *t
		src = append(src, &tc)
	}
	if len(src) == 0 {
		return nil
	}
	heldDoc := map[string]bool{}
	heldMatch := map[int64]bool{}
	for t, err := range tx.Transcriptions().List(ctx, scope) {
		if err != nil {
			return err
		}
		heldDoc[t.Document] = true
		if t.MatchID != 0 {
			heldMatch[t.MatchID] = true
		}
	}
	for _, t := range src {
		// List may leave the document out; Get reads the whole draft.
		full, err := source.Transcriptions().Get(ctx, scope, t.ID)
		if err != nil {
			return fmt.Errorf("ingest: read source transcription %d: %w", t.ID, err)
		}
		match := res.MatchMap[full.MatchID] // 0 when its match did not travel
		if heldDoc[full.Document] || (match != 0 && heldMatch[match]) {
			continue
		}
		d := *full
		d.ID, d.Revision, d.MatchID = 0, 0, match
		id, err := tx.Transcriptions().Save(ctx, scope, &d)
		if err != nil {
			return fmt.Errorf("ingest: import transcription %q: %w", d.Label, err)
		}
		if err := tx.Transcriptions().KeepDates(ctx, scope, id, full.CreatedAt, full.UpdatedAt); err != nil {
			return fmt.Errorf("ingest: import transcription %q dates: %w", d.Label, err)
		}
		heldDoc[d.Document] = true
		if match != 0 {
			heldMatch[match] = true
		}
		res.Transcriptions++
	}
	return nil
}

func mergeAliases(ctx context.Context, tx storage.Stores, scope string, source storage.Stores, res *MatchMerge) error {
	for _, kind := range []storage.AliasKind{storage.AliasPlayer, storage.AliasEvent} {
		src, err := source.Aliases().List(ctx, scope, kind)
		if err != nil {
			res.unreadable(string(kind)+" aliases", err)
			continue
		}
		if len(src) == 0 {
			continue
		}
		held, err := tx.Aliases().List(ctx, scope, kind)
		if err != nil {
			return err
		}
		spelled := map[string]bool{}
		canonical := map[string]bool{}
		for _, a := range held {
			spelled[strings.ToLower(a.Alias)] = true
			canonical[strings.ToLower(a.Canonical)] = true
		}
		for _, a := range src {
			// A spelling the target already resolves, or a name it holds as
			// canonical: setting it would overrule the recipient. A canonical
			// the target spells otherwise is followed by Set itself.
			if spelled[strings.ToLower(a.Alias)] || canonical[strings.ToLower(a.Alias)] {
				continue
			}
			if err := tx.Aliases().Set(ctx, scope, kind, a.Alias, a.Canonical); err != nil {
				return fmt.Errorf("ingest: import %s alias %q: %w", kind, a.Alias, err)
			}
			if err := tx.Aliases().KeepCreatedAt(ctx, scope, kind, a.Alias, a.CreatedAt); err != nil {
				return fmt.Errorf("ingest: import %s alias %q date: %w", kind, a.Alias, err)
			}
			spelled[strings.ToLower(a.Alias)] = true
			canonical[strings.ToLower(a.Canonical)] = true
			res.Aliases++
		}
	}
	return nil
}

// applyMatchMerge reports a MergeMatches in the import's Summary.
func (s *Summary) applyMatchMerge(m MatchMerge) {
	s.Matches = m.Matches
	s.SkippedDuplicates = m.Duplicates
	s.Tournaments = m.Tournaments
	s.Transcriptions = m.Transcriptions
	s.Aliases = m.Aliases
	s.Unreadable = m.Unreadable
}

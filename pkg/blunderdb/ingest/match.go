package ingest

import (
	"context"
	"fmt"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// MatchGraph is the backend-independent, fully-parsed representation of a match
// ready to be written through Storage. Format parsers (XG, GnuBG, BGF, …) map
// their output into a MatchGraph; WriteMatch persists it.
type MatchGraph struct {
	Match domain.Match
	Games []GameGraph
	// CommentOrigin is the provenance stamped on every comment this graph
	// carries. Each format mapper sets its own; the zero value reads as
	// "unknown".
	CommentOrigin domain.CommentOrigin
	// ImportBatchID stamps the match with the import it came in with, 0 for
	// none. WriteMatch copies it onto Match, overriding whatever the mapper
	// left in Match.ImportBatchID.
	ImportBatchID int64
	// ReplaceMatchID names an existing match this graph REPLACES rather than
	// creates, 0 for the ordinary "this match is arriving" case. It is how a
	// transcription draft is saved again after a correction (ADR-0045 §2): the
	// match keeps its id, which other rows point at, while its games, moves
	// and positions are rewritten. Importers never set it.
	ReplaceMatchID int64
	// SkipDuplicates restores the plain skip of an exact duplicate: by default
	// a match already stored still hands over the analyses it holds deeper
	// than the stored ones (deepenAnalyses). Set by the caller, like
	// ImportBatchID.
	SkipDuplicates bool
}

// GameGraph is one game with its ordered moves.
type GameGraph struct {
	Game  domain.Game
	Moves []MoveGraph
}

// MoveGraph is one move plus the position it was played from and that
// position's analysis. Analyses holds the analysis fragments to apply to the
// position, in order: each is merged into whatever is already stored for the
// (deduplicated) position, mirroring the sequence of saveAnalysisInTx calls the
// legacy importer makes (e.g. a checker move with a preceding cube decision
// contributes a checker fragment then a cube fragment). It is empty for a move
// with no engine analysis. Comments are the free-text notes attached to the
// position in the source file (one per entry).
type MoveGraph struct {
	Move     domain.Move
	Position *domain.Position
	Analyses []*domain.PositionAnalysis
	Comments []string
}

// WriteResult summarises a WriteMatch call.
type WriteResult struct {
	MatchID        int64
	Skipped        bool // true when an exact same-format duplicate was found (nothing written but flags)
	FlagsApplied   int  // study marks a skipped duplicate newly raised on already-stored positions
	Deepened       int  // positions of a skipped duplicate whose stored analysis this import deepened
	Enriched       bool // true when a cross-format (canonical) duplicate was enriched in place
	Replaced       bool // true when an existing match was rewritten in place (MatchGraph.ReplaceMatchID)
	SavedPositions int
	// Tournament is the event name the match was filed under, empty when the
	// file named none or when the match already existed.
	Tournament string
	// ProbableDuplicate is set when the match this call created has the dice
	// of a match already stored under other player names: a signal for the
	// report, never a merge.
	ProbableDuplicate *domain.DuplicateSuspect
}

// WriteMatch persists a MatchGraph through tx. It is the single Storage-based
// sink shared by every import format. The whole graph is written inside the
// caller-provided transaction, so an import of several matches is atomic and
// cancellable as a unit.
//
// Duplicate detection has two levels, mirroring the legacy importers:
//   - Exact same-format duplicate (MatchHash already present): no match, game
//     or move row is written, WriteResult.Skipped is true. The analyses it
//     carries are still offered to the stored positions, and only a deeper
//     one replaces what is stored (deepenAnalyses): a 3-ply then a Roller++
//     analysis of the same match keep the Roller++ whatever the import order.
//     MatchGraph.SkipDuplicates turns this off.
//   - Cross-format duplicate (CanonicalHash present from another format, e.g.
//     the same match imported from XG and then GnuBG): the match/game/move rows
//     are NOT recreated, but the graph's positions and analyses are still
//     written — deduplicating by Zobrist they land on the existing positions,
//     and mergeAnalysis combines this format's analysis with what is already
//     stored (cross-engine cube analyses, extra checker moves). Enriched is
//     true.
//
// A third mode, replacement, is asked for explicitly by MatchGraph.ReplaceMatchID
// and short-circuits both: the caller is not offering a match that may already be
// here, it is rewriting one it owns (ADR-0045 §2).
//
// Positions dedup independently by Zobrist hash inside PositionStore.Save.
func WriteMatch(ctx context.Context, tx storage.Tx, scope string, g *MatchGraph, prog func(Progress)) (WriteResult, error) {
	var res WriteResult

	// Replacement: drop the old games first — their moves are what still names
	// the outgoing positions, and the purge at the end needs that list. The
	// match row itself stays, id and all.
	replace := g.ReplaceMatchID != 0
	var orphanCandidates []int64
	if replace {
		ids, err := tx.Matches().DeleteGames(ctx, scope, g.ReplaceMatchID)
		if err != nil {
			return res, err
		}
		orphanCandidates = ids
	}

	// Exact same-format duplicate → skip entirely. A replacement never asks:
	// the caller named the match, and a draft re-saved unchanged would
	// otherwise find its own hash and skip its own rewrite.
	if !replace && g.Match.MatchHash != "" {
		id, found, err := tx.Matches().FindByHash(ctx, scope, g.Match.MatchHash, "")
		if err != nil {
			return res, err
		}
		if found {
			// Still deliver the source-tool study marks: a flag added in XG
			// after the import does not change the match hash, and re-importing
			// would mean deleting the match. Save ORs the flag, so this can
			// only add (ADR-0006).
			n, err := applyFlags(ctx, tx, scope, g)
			if err != nil {
				return res, err
			}
			res = WriteResult{MatchID: id, Skipped: true, FlagsApplied: n}
			if !g.SkipDuplicates {
				if res.Deepened, err = deepenAnalyses(ctx, tx, scope, g); err != nil {
					return res, err
				}
			}
			return res, nil
		}
	}

	// Cross-format (canonical) duplicate → enrich the existing match's positions.
	enrich := false
	var matchID int64
	if !replace && g.Match.CanonicalHash != "" {
		id, found, err := tx.Matches().FindByHash(ctx, scope, "", g.Match.CanonicalHash)
		if err != nil {
			return res, err
		}
		if found {
			enrich = true
			matchID = id
		}
	}

	// The aliases rename what is stored, after both fingerprints: the hashes
	// keep the names the file wrote, so a file imported before its alias
	// existed still finds itself on its next import (storage.AliasStore).
	var aliases aliasMaps
	fileNames := [2]string{g.Match.Player1Name, g.Match.Player2Name}
	if !enrich && !replace {
		var err error
		if aliases, err = loadAliases(ctx, tx, scope); err != nil {
			return res, err
		}
		aliases.apply(&g.Match)
	}

	// The names are not in the dice hash: a match stored under other names
	// is found here, and only signalled.
	if !enrich && g.Match.DiceHash == "" {
		initial, dice := graphDice(g)
		g.Match.DiceHash = DiceMatchHash(int(g.Match.MatchLength), initial, dice)
	}
	if !enrich && !replace {
		suspect, sameID, err := probableDuplicate(ctx, tx, scope, &g.Match, fileNames, aliases.players)
		if err != nil {
			return res, err
		}
		res.ProbableDuplicate = suspect
		// The same dice under names the aliases say are the same people: a
		// known spelling of a stored match, enriched like a cross-format copy.
		if sameID != 0 {
			enrich = true
			matchID = sameID
		}
	}

	switch {
	case replace:
		// The header is re-stated, never re-inserted: ReplaceHeader leaves the
		// id, the import date, the import batch, the tournament, the match
		// comment and the last-visited position exactly where they were. The
		// tournament is not re-filed: correcting a die is not a request to
		// move the match.
		matchID = g.ReplaceMatchID
		g.Match.ID = matchID
		if err := tx.Matches().ReplaceHeader(ctx, scope, matchID, &g.Match); err != nil {
			return res, err
		}

	case !enrich:
		// The batch the caller opened wins over whatever a mapper left on the
		// match: only the caller knows what the user meant as one import.
		g.Match.ImportBatchID = g.ImportBatchID
		id, err := tx.Matches().Save(ctx, scope, &g.Match)
		if err != nil {
			return res, err
		}
		matchID = id

		// The file names its event; make it a tournament. Only on a match this
		// import created: a match already stored may have been filed by hand,
		// and re-importing its file must not move it. Date and location start
		// empty.
		if name := strings.TrimSpace(g.Match.Event); name != "" {
			if err := tx.Tournaments().SetMatchByName(ctx, scope, matchID, name); err != nil {
				return res, err
			}
			res.Tournament = name
		}
	}
	res.MatchID = matchID
	if res.ProbableDuplicate != nil {
		res.ProbableDuplicate.MatchID = matchID
	}
	res.Enriched = enrich
	res.Replaced = replace

	counter := Progress{Matches: 1}
	for gi := range g.Games {
		gg := &g.Games[gi]
		var gameID int64
		if !enrich {
			gg.Game.MatchID = matchID
			id, err := tx.Matches().CreateGame(ctx, scope, &gg.Game)
			if err != nil {
				return res, err
			}
			gameID = id
		}
		counter.Games++

		for mi := range gg.Moves {
			mg := &gg.Moves[mi]
			if mg.Position != nil {
				played := &storage.PlayedActions{CheckerMove: mg.Move.CheckerMove, CubeAction: mg.Move.CubeAction}
				posID, err := savePositionWithAnalyses(ctx, tx, scope, mg.Position, played, mg.Analyses, mg.Comments, g.CommentOrigin)
				if err != nil {
					return res, err
				}
				mg.Move.PositionID = posID
				res.SavedPositions++
				counter.Positions++
			}
			if !enrich {
				mg.Move.GameID = gameID
				if _, err := tx.Matches().CreateMove(ctx, scope, &mg.Move); err != nil {
					return res, err
				}
			}
			if prog != nil {
				prog(counter)
			}
		}
	}

	// Only now, with the new moves written, is the retention predicate asked
	// about the positions the old ones referenced: a position both versions
	// share is held by its new move and survives on its existing row, with its
	// analysis and its id; one only the corrected Action reached goes unless
	// the retention predicate holds it. The ordinary purge, no trash
	// (ADR-0045 §3).
	if replace && len(orphanCandidates) > 0 {
		if err := tx.Matches().PurgeOrphanPositions(ctx, scope, orphanCandidates); err != nil {
			return res, err
		}
	}
	return res, nil
}

// savePositionWithAnalyses saves pos (deduplicated by Zobrist), folds every
// analysis fragment into whatever is stored for it, then adds the comments.
// It is shared by WriteMatch (per move) and the single-position importers.
// played is the decision the graph records at pos (nil for a position
// imported on its own): the analysis columns take it instead of reading the
// move table, which on a common position holds thousands of rows.
//
// The fragments are merged in memory and stored with one AnalysisStore.Merge:
// one read, one encode, one write — and no write at all when the result is
// what is already stored. Between two fragments the partial result is rounded
// as storage would round it, so equity errors are recomputed from rounded
// equities exactly as when each fragment was saved on its own.
func savePositionWithAnalyses(ctx context.Context, tx storage.Tx, scope string, pos *domain.Position, played *storage.PlayedActions, analyses []*domain.PositionAnalysis, comments []string, origin domain.CommentOrigin) (int64, error) {
	posID, err := tx.Positions().Save(ctx, scope, pos)
	if err != nil {
		return 0, err
	}
	frags := make([]*domain.PositionAnalysis, 0, len(analyses))
	for _, frag := range analyses {
		if frag != nil {
			frags = append(frags, frag)
		}
	}
	if len(frags) > 0 {
		if _, err := tx.Analyses().Merge(ctx, scope, posID, played, func(existing *domain.PositionAnalysis) *domain.PositionAnalysis {
			cur := existing
			for i, frag := range frags {
				merged := mergeAnalysis(cur, *frag)
				if i < len(frags)-1 {
					merged.PositionID = int(posID)
					engine.RoundAnalysisForStorage(&merged)
				}
				cur = &merged
			}
			return cur
		}); err != nil {
			return posID, err
		}
	}
	// Add only comments not already on the position: an enrich re-import, or
	// the same file imported twice, revisits a deduplicated position.
	if len(comments) > 0 {
		existing := map[string]bool{}
		for c, err := range tx.Comments().ByPosition(ctx, scope, posID) {
			if err != nil {
				return posID, err
			}
			existing[c.Text] = true
		}
		for _, c := range comments {
			if c == "" || existing[c] {
				continue
			}
			if _, err := tx.Comments().AddFrom(ctx, scope, posID, c, origin); err != nil {
				return posID, err
			}
			existing[c] = true
		}
	}
	return posID, nil
}

// applyFlags raises the source-tool study mark on the positions of a graph
// whose match is already stored, and returns how many marks it actually
// raised — 0 when every mark is already in the database, so the caller can
// tell "duplicate, nothing to do" from "duplicate, N new marks".
//
// It is the one thing an exact duplicate still writes. Only flagged positions
// are visited, and PositionStore.RaiseFlag only ever sets the mark on the
// stored row: nothing is duplicated or cleared.
func applyFlags(ctx context.Context, tx storage.Tx, scope string, g *MatchGraph) (int, error) {
	n := 0
	for gi := range g.Games {
		for mi := range g.Games[gi].Moves {
			pos := g.Games[gi].Moves[mi].Position
			if pos == nil || !pos.Flagged {
				continue
			}
			raised, err := tx.Positions().RaiseFlag(ctx, scope, pos)
			if err != nil {
				return n, fmt.Errorf("ingest: apply flag to duplicate match position: %w", err)
			}
			if raised {
				n++
			}
		}
	}
	return n, nil
}

// deepenAnalyses offers the analyses of a graph whose match is already stored
// to the positions it reaches, and returns how many stored analyses it
// changed. Unlike the cross-format enrichment, the incoming analysis is the
// same engine's view of the same decisions, so it replaces what is stored only
// where it is strictly deeper (deepenAnalysis): re-importing the same file,
// or a shallower version of it, writes nothing. A position no longer stored
// is left alone rather than recreated.
func deepenAnalyses(ctx context.Context, tx storage.Tx, scope string, g *MatchGraph) (int, error) {
	n := 0
	for gi := range g.Games {
		for mi := range g.Games[gi].Moves {
			mg := &g.Games[gi].Moves[mi]
			if mg.Position == nil || len(mg.Analyses) == 0 {
				continue
			}
			posID, found, err := tx.Positions().Exists(ctx, scope, engine.PopulatePositionColumns(mg.Position).ZobristHash)
			if err != nil {
				return n, err
			}
			if !found {
				continue
			}
			played := &storage.PlayedActions{CheckerMove: mg.Move.CheckerMove, CubeAction: mg.Move.CubeAction}
			written, err := tx.Analyses().Merge(ctx, scope, posID, played, func(existing *domain.PositionAnalysis) *domain.PositionAnalysis {
				cur := existing
				for _, frag := range mg.Analyses {
					if frag == nil {
						continue
					}
					next := deepenAnalysis(cur, *frag)
					if next != cur {
						next.PositionID = int(posID)
						engine.RoundAnalysisForStorage(next)
					}
					cur = next
				}
				return cur
			})
			if err != nil {
				return n, fmt.Errorf("ingest: deepen duplicate match analysis: %w", err)
			}
			if written {
				n++
			}
		}
	}
	return n, nil
}

// probableDuplicate looks for a stored match with m's dice under other
// player names, and returns it as a suspect whose MatchID the caller fills
// once m is saved; nil when there is none. When the names differ only by
// aliases, the stored match is the same one: its id comes back as sameID and
// no suspect is raised.
//
// fileNames are the names the file wrote, before the aliases renamed m's.
func probableDuplicate(ctx context.Context, tx storage.Tx, scope string, m *domain.Match, fileNames [2]string, players storage.AliasMap) (suspect *domain.DuplicateSuspect, sameID int64, err error) {
	others, err := tx.Matches().ListByDiceHash(ctx, scope, m.DiceHash)
	if err != nil {
		return nil, 0, err
	}
	for _, o := range others {
		if samePlayers(o.Player1Name, o.Player2Name, fileNames[0], fileNames[1]) {
			continue
		}
		if samePlayers(players.Canonical(o.Player1Name), players.Canonical(o.Player2Name), m.Player1Name, m.Player2Name) {
			return nil, o.ID, nil
		}
		return &domain.DuplicateSuspect{
			Kind: domain.DuplicateSameDice, OtherID: o.ID,
			Players:      m.Player1Name + " – " + m.Player2Name,
			OtherPlayers: o.Player1Name + " – " + o.Player2Name,
		}, 0, nil
	}
	return nil, 0, nil
}

// aliasMaps holds the player and event aliases an import renames through.
type aliasMaps struct {
	players, events storage.AliasMap
}

func loadAliases(ctx context.Context, tx storage.Tx, scope string) (aliasMaps, error) {
	var a aliasMaps
	p, err := tx.Aliases().List(ctx, scope, storage.AliasPlayer)
	if err != nil {
		return a, err
	}
	e, err := tx.Aliases().List(ctx, scope, storage.AliasEvent)
	if err != nil {
		return a, err
	}
	return aliasMaps{players: storage.NewAliasMap(p), events: storage.NewAliasMap(e)}, nil
}

// apply stores the canonical names in place of the aliases the file wrote.
func (a aliasMaps) apply(m *domain.Match) {
	if len(a.players) > 0 {
		m.Player1Name = a.players.Canonical(m.Player1Name)
		m.Player2Name = a.players.Canonical(m.Player2Name)
	}
	if len(a.events) > 0 && strings.TrimSpace(m.Event) != "" {
		m.Event = a.events.Canonical(m.Event)
	}
}

// samePlayers reports whether two matches name the same two players, in
// either seat, ignoring case and surrounding spaces.
func samePlayers(a1, a2, b1, b2 string) bool {
	n := func(s string) string { return strings.ToLower(strings.TrimSpace(s)) }
	x1, x2, y1, y2 := n(a1), n(a2), n(b1), n(b2)
	return (x1 == y1 && x2 == y2) || (x1 == y2 && x2 == y1)
}

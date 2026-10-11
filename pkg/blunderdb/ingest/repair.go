package ingest

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// The refusals of RepairMatch. Each matches storage.ErrInvalid or
// storage.ErrNotFound, so every mode maps them as it maps its other refusals.
var (
	// ErrRepairNoSource — the match names no source file (a Duel, a
	// Transcription), or the file it names is no longer on disk.
	ErrRepairNoSource = fmt.Errorf("%w: the source file of the match cannot be read", storage.ErrNotFound)

	// ErrRepairSourceChanged — the source file no longer holds this match:
	// neither its format hash nor its canonical hash is the stored one.
	// Rewriting the match from it would replace one match with another.
	ErrRepairSourceChanged = fmt.Errorf("%w: the source file no longer holds this match", storage.ErrInvalid)

	// ErrRepairSeatsUnclear — the stored moves do not say which way round
	// the stored match names the seats: too few of them line up with the
	// file's, their players split, or they contradict the names. Guessing
	// would give each player the other's figures.
	ErrRepairSeatsUnclear = fmt.Errorf("%w: the seats of the stored match cannot be matched with its source file", storage.ErrInvalid)
)

// RepairCounts is what a repair report counts on one side of the repair:
// the match's moves, those that stand on a position, and those whose play
// is scored by their position's analysis.
type RepairCounts struct {
	Moves      int `json:"moves"`
	Positioned int `json:"positioned"`
	Scored     int `json:"scored"`
}

// RepairReport says what RepairMatch changed, or would change under a dry
// run.
type RepairReport struct {
	MatchID  int64  `json:"match_id"`
	FilePath string `json:"file_path"`
	// Swapped is true when the stored match names the seats the other way
	// round from its file: the repair writes the file's games in the stored
	// seat order, as MatchStore.SwapPlayers would have left them.
	Swapped bool         `json:"swapped"`
	Before  RepairCounts `json:"before"`
	After   RepairCounts `json:"after"`
	// ChangedMoves counts the moves whose position the repair changes,
	// matched by game and move number.
	ChangedMoves int `json:"changed_moves"`
	// RehomedComments counts the user's comments moved from a position no
	// move holds any more onto the one the file names.
	RehomedComments int `json:"rehomed_comments"`
	// KeptComments counts the user's comments left on a position the match
	// no longer reaches but something else holds (a collection, an Anki
	// card, a lesson, a mark): the position stays, and its comments with it.
	KeptComments int  `json:"kept_comments"`
	DryRun       bool `json:"dry_run"`
}

// Differs reports whether the stored match differs from its source file.
func (r RepairReport) Differs() bool {
	return r.ChangedMoves > 0 || r.Before != r.After
}

// MapMatchFile reads a match file through the mapper its extension names,
// as an import does. A position file is refused.
func MapMatchFile(path string) (*MatchGraph, error) {
	g, _, err := mapFile(path)
	if err != nil {
		return nil, err
	}
	if g == nil {
		return nil, fmt.Errorf("%w: %s holds no match", storage.ErrInvalid, filepath.Base(path))
	}
	return g, nil
}

// MapMatchSource reads the source file a stored match names (its
// file_path). A relative path was relative to the directory the import ran
// from, which nothing recorded: it is looked for under each of dirs in turn
// (the library's directory, the working directory), and the hash check of
// RepairMatch refuses a file that is not the match. ErrRepairNoSource when
// the match names no file or none is found.
func MapMatchSource(path string, dirs ...string) (*MatchGraph, error) {
	if strings.TrimSpace(path) == "" {
		return nil, ErrRepairNoSource
	}
	candidates := []string{path}
	if !filepath.IsAbs(path) {
		candidates = candidates[:0]
		for _, d := range dirs {
			candidates = append(candidates, filepath.Join(d, path))
		}
		candidates = append(candidates, path)
	}
	// A candidate that cannot be stat'ed (a directory not readable, a path
	// too long) is not the file: the next one may be.
	var firstErr error
	for _, c := range candidates {
		_, err := os.Stat(c)
		if err == nil {
			return MapMatchFile(c)
		}
		if firstErr == nil && !errors.Is(err, os.ErrNotExist) {
			firstErr = err
		}
	}
	if firstErr != nil {
		return nil, fmt.Errorf("%w (%s): %w", ErrRepairNoSource, path, firstErr)
	}
	return nil, fmt.Errorf("%w (%s)", ErrRepairNoSource, path)
}

// moveKey names a move across a rewrite: a double and its answer share a
// move number, and their action tells them apart.
type moveKey struct {
	game, move   int32
	kind, action string
}

func keyOf(game int32, mv *domain.Move) moveKey {
	return moveKey{game, mv.MoveNumber, mv.MoveType, mv.CubeAction}
}

type storedMove struct {
	player   int32
	position int64
	scored   bool
	rollTick *int64
	tick     *int64
}

// RepairMatch rewrites the games, moves, positions and analyses of the
// stored match matchID from g, the graph of its source file, in one
// transaction: what a fresh import of the file writes, under the match's
// existing row. What the user brought is kept: the id, the tournament, the
// comment, the import batch, the names as they are stored (aliases, edits),
// the date, event, location and round, the video and its Repères, the
// Transcription that points at the match, and the comments the user wrote on
// a position the repair leaves (Rehome). A match whose seats are stored the
// other way round from the file is written in the stored seat order
// (swapGraphSeats), so each person keeps their seat.
//
// The file must still hold the match: its format or canonical hash must be
// the stored one, otherwise ErrRepairSourceChanged. With dryRun the
// transaction is rolled back and the report says what would change; a match
// that already is its file is rolled back too, so a repair run twice writes
// nothing the second time.
//
// Whether a position the match leaves is purged is the retention
// predicate's call, and a user comment is one of the things that hold a
// position: the comments are moved first, and a position still held after
// the purge (a collection, a card) had to keep them. The repair is then
// rolled back and run again, leaving those comments where they are.
func RepairMatch(ctx context.Context, store TxBeginner, scope string, matchID int64, g *MatchGraph, dryRun bool) (RepairReport, error) {
	keep := map[int64]bool{}
	// Each pass adds at least one position to keep, and a position is kept
	// at most once: the bound is a guard, not a budget.
	for range 8 {
		rep, held, err := repairOnce(ctx, store, scope, matchID, cloneGraph(g), dryRun, keep)
		if err != nil || len(held) == 0 {
			return rep, err
		}
		for _, id := range held {
			keep[id] = true
		}
	}
	return RepairReport{MatchID: matchID, DryRun: dryRun}, fmt.Errorf("ingest: repair of match %d does not settle which positions keep their comments", matchID)
}

// cloneGraph copies what a repair pass rewrites in g (the header, the seats,
// the moves' ids), so a pass rolled back leaves g as the file mapped it.
func cloneGraph(g *MatchGraph) *MatchGraph {
	c := *g
	c.Games = make([]GameGraph, len(g.Games))
	for i, gg := range g.Games {
		c.Games[i] = GameGraph{Game: gg.Game, Moves: append([]MoveGraph(nil), gg.Moves...)}
	}
	return &c
}

// repairOnce is one pass of RepairMatch. It returns, without committing,
// the positions whose comments it moved but that the purge kept; keep names
// the positions whose comments stay.
func repairOnce(ctx context.Context, store TxBeginner, scope string, matchID int64, g *MatchGraph, dryRun bool, keep map[int64]bool) (RepairReport, []int64, error) {
	rep := RepairReport{MatchID: matchID, DryRun: dryRun}
	tx, err := store.BeginTx(ctx)
	if err != nil {
		return rep, nil, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	ms := tx.Matches()
	stored, err := ms.Get(ctx, scope, matchID)
	if err != nil {
		return rep, nil, err
	}
	rep.FilePath = stored.FilePath
	sameMatch := (stored.MatchHash != "" && stored.MatchHash == g.Match.MatchHash) ||
		(stored.CanonicalHash != "" && stored.CanonicalHash == g.Match.CanonicalHash)
	if !sameMatch {
		return rep, nil, ErrRepairSourceChanged
	}

	before, counts, err := readStoredMoves(ctx, ms, scope, matchID)
	if err != nil {
		return rep, nil, err
	}
	rep.Before = counts

	// The seats: a move's player is what a swap changes and nothing else
	// does, so the moves vote. Most of the file's moves must take part, the
	// vote must be clear, and names that read the same on both sides (no
	// alias, no edit) must not say otherwise.
	agree, disagree, fileMoves := 0, 0, 0
	for _, gg := range g.Games {
		for _, mg := range gg.Moves {
			fileMoves++
			sm, ok := before[keyOf(gg.Game.GameNumber, &mg.Move)]
			if !ok {
				continue
			}
			if sm.player == mg.Move.Player {
				agree++
			} else {
				disagree++
			}
		}
	}
	votes := agree + disagree
	if votes == 0 || 2*votes < fileMoves || 4*min(agree, disagree) > votes {
		return rep, nil, ErrRepairSeatsUnclear
	}
	rep.Swapped = disagree > agree
	if byNames, ok := seatsByNames(stored, &g.Match); ok && byNames != rep.Swapped {
		return rep, nil, ErrRepairSeatsUnclear
	}

	// The dice hash is the file's, whatever the seats.
	if g.Match.DiceHash == "" {
		initial, dice := graphDice(g)
		g.Match.DiceHash = DiceMatchHash(int(g.Match.MatchLength), initial, dice)
	}
	if rep.Swapped {
		swapGraphSeats(g)
	}

	// The header the user may have edited stays as stored.
	h := &g.Match
	h.Player1Name, h.Player2Name = stored.Player1Name, stored.Player2Name
	h.MatchDate = stored.MatchDate
	h.Event, h.Location, h.Round = stored.Event, stored.Location, stored.Round
	h.FilePath = stored.FilePath
	h.VideoSource = nil // kept by ReplaceHeader

	// The Repères are typed by hand against the video: carried by move.
	for gi := range g.Games {
		gg := &g.Games[gi]
		for mi := range gg.Moves {
			mv := &gg.Moves[mi].Move
			if sm, ok := before[keyOf(gg.Game.GameNumber, mv)]; ok {
				mv.RollTickMS, mv.TickMS = sm.rollTick, sm.tick
			}
		}
	}

	g.ReplaceMatchID = matchID
	if _, err := WriteMatch(ctx, tx, scope, g, nil); err != nil {
		return rep, nil, err
	}

	after, counts, err := readStoredMoves(ctx, ms, scope, matchID)
	if err != nil {
		return rep, nil, err
	}
	rep.After = counts

	// A position the repair left that no move holds any more hands the
	// user's comments to the position the file names for the same move, then
	// goes the way of any orphan.
	left := map[int64]int64{}
	for k, sm := range before {
		am, ok := after[k]
		if !ok || am.position == sm.position {
			continue
		}
		rep.ChangedMoves++
		if sm.position != 0 && am.position != 0 {
			left[sm.position] = am.position
		}
	}
	if len(left) > 0 {
		ids := make([]int64, 0, len(left))
		for id := range left {
			ids = append(ids, id)
		}
		held, err := ms.MovesByPositions(ctx, scope, ids)
		if err != nil {
			return rep, nil, err
		}
		var moved, kept []int64
		for from, to := range left {
			if len(held[from]) > 0 {
				continue
			}
			if keep[from] {
				kept = append(kept, from)
				continue
			}
			n, err := tx.Comments().Rehome(ctx, scope, from, to)
			if err != nil {
				return rep, nil, err
			}
			if n > 0 {
				moved = append(moved, from)
			}
			rep.RehomedComments += n
		}
		if err := ms.PurgeOrphanPositions(ctx, scope, ids); err != nil {
			return rep, nil, err
		}
		if len(moved) > 0 {
			survivors, err := tx.Positions().LoadByIDs(ctx, scope, moved)
			if err != nil {
				return rep, nil, err
			}
			if len(survivors) > 0 {
				out := make([]int64, len(survivors))
				for i := range survivors {
					out[i] = survivors[i].ID
				}
				return rep, out, nil
			}
		}
		if len(kept) > 0 {
			comments, err := tx.Comments().ByPositions(ctx, scope, kept)
			if err != nil {
				return rep, nil, err
			}
			for _, cs := range comments {
				for _, c := range cs {
					if c.Origin == domain.CommentOriginUser {
						rep.KeptComments++
					}
				}
			}
		}
	}

	if dryRun || !rep.Differs() {
		return rep, nil, nil
	}
	if err := tx.Commit(); err != nil {
		return rep, nil, err
	}
	committed = true
	return rep, nil, nil
}

// seatsByNames reads the seats from the names when both sides spell them
// alike: ok is false when they do not (an alias, an edit, two equal names),
// and the moves alone decide.
func seatsByNames(stored, file *domain.Match) (swapped, ok bool) {
	norm := func(s string) string { return strings.ToLower(strings.TrimSpace(s)) }
	s1, s2 := norm(stored.Player1Name), norm(stored.Player2Name)
	f1, f2 := norm(file.Player1Name), norm(file.Player2Name)
	if s1 == s2 || f1 == f2 {
		return false, false
	}
	switch {
	case s1 == f1 && s2 == f2:
		return false, true
	case s1 == f2 && s2 == f1:
		return true, true
	}
	return false, false
}

// swapGraphSeats turns g into the match MatchStore.SwapPlayers leaves: the
// seat-bound header fields, every game's initial score and winner, and every
// move's player. Positions are stored from the side on roll: none changes.
func swapGraphSeats(g *MatchGraph) {
	m := &g.Match
	m.Player1Elo, m.Player2Elo = m.Player2Elo, m.Player1Elo
	m.Player1Experience, m.Player2Experience = m.Player2Experience, m.Player1Experience
	for gi := range g.Games {
		gm := &g.Games[gi].Game
		gm.InitialScore[0], gm.InitialScore[1] = gm.InitialScore[1], gm.InitialScore[0]
		gm.Winner = -gm.Winner
		for mi := range g.Games[gi].Moves {
			mv := &g.Games[gi].Moves[mi].Move
			mv.Player = -mv.Player
		}
	}
}

// readStoredMoves reads the match's moves keyed by game and move number,
// with the counts a report shows.
func readStoredMoves(ctx context.Context, ms storage.MatchStore, scope string, matchID int64) (map[moveKey]storedMove, RepairCounts, error) {
	var c RepairCounts
	gameNumber := map[int64]int32{}
	for gm, err := range ms.Games(ctx, scope, matchID) {
		if err != nil {
			return nil, c, err
		}
		gameNumber[gm.ID] = gm.GameNumber
	}
	out := map[moveKey]storedMove{}
	for mv, err := range ms.MovesByMatch(ctx, scope, matchID) {
		if err != nil {
			return nil, c, err
		}
		c.Moves++
		if mv.PositionID != 0 {
			c.Positioned++
		}
		if mv.ErrorMP != nil {
			c.Scored++
		}
		out[keyOf(gameNumber[mv.GameID], mv)] = storedMove{
			player: mv.Player, position: mv.PositionID, scored: mv.ErrorMP != nil,
			rollTick: mv.RollTickMS, tick: mv.TickMS,
		}
	}
	return out, c, nil
}

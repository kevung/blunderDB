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
	RehomedComments int  `json:"rehomed_comments"`
	DryRun          bool `json:"dry_run"`
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
// file_path). ErrRepairNoSource when it names none or the file is gone.
func MapMatchSource(path string) (*MatchGraph, error) {
	if strings.TrimSpace(path) == "" {
		return nil, ErrRepairNoSource
	}
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%w (%s)", ErrRepairNoSource, path)
		}
		return nil, err
	}
	return MapMatchFile(path)
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
// transaction is rolled back and the report says what would change.
func RepairMatch(ctx context.Context, store TxBeginner, scope string, matchID int64, g *MatchGraph, dryRun bool) (RepairReport, error) {
	rep := RepairReport{MatchID: matchID, DryRun: dryRun}
	tx, err := store.BeginTx(ctx)
	if err != nil {
		return rep, err
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
		return rep, err
	}
	rep.FilePath = stored.FilePath
	sameMatch := (stored.MatchHash != "" && stored.MatchHash == g.Match.MatchHash) ||
		(stored.CanonicalHash != "" && stored.CanonicalHash == g.Match.CanonicalHash)
	if !sameMatch {
		return rep, ErrRepairSourceChanged
	}

	before, counts, err := readStoredMoves(ctx, ms, scope, matchID)
	if err != nil {
		return rep, err
	}
	rep.Before = counts

	// The seats: a move's player is what a swap changes and nothing else
	// does, so the moves vote.
	agree, disagree := 0, 0
	for _, gg := range g.Games {
		for _, mg := range gg.Moves {
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
	rep.Swapped = disagree > agree

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
		return rep, err
	}

	after, counts, err := readStoredMoves(ctx, ms, scope, matchID)
	if err != nil {
		return rep, err
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
			return rep, err
		}
		for from, to := range left {
			if len(held[from]) > 0 {
				continue
			}
			n, err := tx.Comments().Rehome(ctx, scope, from, to)
			if err != nil {
				return rep, err
			}
			rep.RehomedComments += n
		}
		if err := ms.PurgeOrphanPositions(ctx, scope, ids); err != nil {
			return rep, err
		}
	}

	if dryRun {
		return rep, nil
	}
	if err := tx.Commit(); err != nil {
		return rep, err
	}
	committed = true
	return rep, nil
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

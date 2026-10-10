package storagetest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// repoTestdata is the repository's testdata directory, whichever backend's
// package runs the suite.
func repoTestdata(name string) string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "..", "testdata", name)
}

// repairMove is what a repair must restore of one move: the position (by
// Zobrist), its analysis, the play's score, and the seat.
type repairMove struct {
	Zobrist  uint64
	Analysis string
	ErrorMP  string
	Player   int32
	RollTick string
}

type repairSnapshot struct {
	Moves    map[string]repairMove
	Analysed int
}

func takeRepairSnapshot(t *testing.T, s storage.Storage, matchID int64) repairSnapshot {
	t.Helper()
	ctx := context.Background()
	ms := s.Matches()
	gameNumber := map[int64]int32{}
	for g, err := range ms.Games(ctx, "", matchID) {
		if err != nil {
			t.Fatalf("Games: %v", err)
		}
		gameNumber[g.ID] = g.GameNumber
	}
	var moves []*domain.Move
	var ids []int64
	for mv, err := range ms.MovesByMatch(ctx, "", matchID) {
		if err != nil {
			t.Fatalf("MovesByMatch: %v", err)
		}
		moves = append(moves, mv)
		if mv.PositionID != 0 {
			ids = append(ids, mv.PositionID)
		}
	}
	positions, err := s.Positions().LoadByIDs(ctx, "", ids)
	if err != nil {
		t.Fatalf("LoadByIDs: %v", err)
	}
	hash := map[int64]uint64{}
	for i := range positions {
		hash[positions[i].ID] = engine.ZobristHash(&positions[i])
	}
	analyses, err := s.Analyses().LoadMany(ctx, "", ids)
	if err != nil {
		t.Fatalf("LoadMany: %v", err)
	}
	sn := repairSnapshot{Moves: map[string]repairMove{}}
	for _, mv := range moves {
		rm := repairMove{Zobrist: hash[mv.PositionID], ErrorMP: fmt.Sprint(repairDeref(mv.ErrorMP)), Player: mv.Player, RollTick: fmt.Sprint(repairDeref(mv.RollTickMS))}
		if a := analyses[mv.PositionID]; a != nil {
			c := *a
			c.PositionID, c.CreationDate, c.LastModifiedDate = 0, time.Time{}, time.Time{}
			// A merge onto a stored row may write the same play with its
			// parts in another order: the plays compare as sets.
			c.PlayedMoves, c.PlayedCubeActions = playSet(c.PlayedMoves), playSet(c.PlayedCubeActions)
			b, _ := json.Marshal(c)
			rm.Analysis = string(b)
			sn.Analysed++
		}
		sn.Moves[repairKey(gameNumber[mv.GameID], mv.MoveNumber, mv.MoveType, mv.CubeAction)] = rm
	}
	return sn
}

// damageLikeTheOldSwap rewrites the match the way inverting its players did
// before positions were known to be stored from the side on roll: every
// position copied with its score and cube owner exchanged (a new Zobrist, no
// analysis), and the seats exchanged everywhere else.
func damageLikeTheOldSwap(t *testing.T, s storage.Storage, matchID int64, path string, tick int64) {
	t.Helper()
	ctx := context.Background()
	g, err := ingest.MapMatchFile(path)
	if err != nil {
		t.Fatalf("map %s: %v", path, err)
	}
	stored, err := s.Matches().Get(ctx, "", matchID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	g.Match.Player1Name, g.Match.Player2Name = stored.Player2Name, stored.Player1Name
	g.Match.FilePath = stored.FilePath
	for gi := range g.Games {
		gm := &g.Games[gi].Game
		gm.InitialScore[0], gm.InitialScore[1] = gm.InitialScore[1], gm.InitialScore[0]
		gm.Winner = -gm.Winner
		for mi := range g.Games[gi].Moves {
			mg := &g.Games[gi].Moves[mi]
			mg.Move.Player = -mg.Move.Player
			mg.Analyses, mg.Comments = nil, nil
			if p := mg.Position; p != nil {
				p.Score[0], p.Score[1] = p.Score[1], p.Score[0]
				if p.Cube.Owner != domain.None {
					p.Cube.Owner = 1 - p.Cube.Owner
				}
			}
			if gi == 0 && mi == 2 {
				mg.Move.RollTickMS, mg.Move.TickMS = &tick, &tick
			}
		}
	}
	g.ReplaceMatchID = matchID
	tx, err := s.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	if _, err := ingest.WriteMatch(ctx, tx, "", g, nil); err != nil {
		_ = tx.Rollback()
		t.Fatalf("damage: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}
}

func importForRepair(t *testing.T, s storage.Storage, path string) int64 {
	t.Helper()
	ctx := context.Background()
	g, err := ingest.MapMatchFile(path)
	if err != nil {
		t.Fatalf("map %s: %v", path, err)
	}
	g.Match.FilePath = path
	tx, err := s.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	res, err := ingest.WriteMatch(ctx, tx, "", g, nil)
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("import: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	return res.MatchID
}

// testMatchRepairFromSource: a match damaged by the old swap is restored,
// from its source file, to what a fresh import holds — positions, analyses,
// the plays' scores — in the seat order the user chose, and everything the
// user brought to it stays.
func testMatchRepairFromSource(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ms := s.Matches()
	path := repoTestdata("test.xg")
	matchID := importForRepair(t, s, path)
	fresh := takeRepairSnapshot(t, s, matchID)
	if fresh.Analysed == 0 {
		t.Fatalf("fixture %s carries no analysis", path)
	}
	freshMatch, err := ms.Get(ctx, "", matchID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	// Before the swap: a comment on a position of the match, which holds it.
	var firstPos int64
	freshPos := map[int64]bool{}
	for mv, err := range ms.MovesByMatch(ctx, "", matchID) {
		if err != nil {
			t.Fatalf("MovesByMatch: %v", err)
		}
		freshPos[mv.PositionID] = true
		if firstPos == 0 && mv.PositionID != 0 {
			firstPos = mv.PositionID
		}
	}
	if _, err := s.Comments().Add(ctx, "", firstPos, "before the swap"); err != nil {
		t.Fatalf("Add comment: %v", err)
	}

	const tick = int64(4242)
	damageLikeTheOldSwap(t, s, matchID, path, tick)

	// After it, what the user brought.
	alias := "Alias of " + freshMatch.Player1Name
	if err := s.Aliases().Set(ctx, "", storage.AliasPlayer, alias, freshMatch.Player1Name); err != nil {
		t.Fatalf("alias: %v", err)
	}
	if err := ms.Update(ctx, "", matchID, "Edited Two", "Edited One", "2024-05-06"); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if err := ms.UpdateComment(ctx, "", matchID, "my match note"); err != nil {
		t.Fatalf("UpdateComment: %v", err)
	}
	if err := ms.SetVideoSource(ctx, "", matchID, "https://example.org/v"); err != nil {
		t.Fatalf("SetVideoSource: %v", err)
	}
	if err := s.Tournaments().SetMatchByName(ctx, "", matchID, "Repair Open"); err != nil {
		t.Fatalf("SetMatchByName: %v", err)
	}
	transID, err := s.Transcriptions().Save(ctx, "", &storage.Transcription{FormatVersion: "1", MatchID: matchID, Document: "{}"})
	if err != nil {
		t.Fatalf("transcription: %v", err)
	}
	var damagedPos int64
	for mv, err := range ms.MovesByMatch(ctx, "", matchID) {
		if err != nil {
			t.Fatalf("MovesByMatch: %v", err)
		}
		if mv.PositionID != 0 && !freshPos[mv.PositionID] {
			damagedPos = mv.PositionID
			break
		}
	}
	if _, err := s.Comments().Add(ctx, "", damagedPos, "after the swap"); err != nil {
		t.Fatalf("Add comment: %v", err)
	}
	damaged := takeRepairSnapshot(t, s, matchID)
	if damaged.Analysed >= fresh.Analysed {
		t.Fatalf("damaged match still has %d analysed moves of %d", damaged.Analysed, fresh.Analysed)
	}
	headerBefore, err := ms.Get(ctx, "", matchID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	graph := func(p string) *ingest.MatchGraph {
		g, err := ingest.MapMatchFile(p)
		if err != nil {
			t.Fatalf("map %s: %v", p, err)
		}
		return g
	}

	// A file that holds another match is refused.
	if _, err := ingest.RepairMatch(ctx, s, "", matchID, graph(repoTestdata("charlot1-charlot2_7p_2025-11-08-2305.xg")), false); !errors.Is(err, ingest.ErrRepairSourceChanged) {
		t.Fatalf("other file: err = %v, want ErrRepairSourceChanged", err)
	}

	// The dry run says what would change and writes nothing.
	rep, err := ingest.RepairMatch(ctx, s, "", matchID, graph(path), true)
	if err != nil {
		t.Fatalf("dry run: %v", err)
	}
	if !rep.Differs() || !rep.Swapped || rep.After.Scored <= rep.Before.Scored {
		t.Fatalf("dry run report = %+v", rep)
	}
	if got := takeRepairSnapshot(t, s, matchID); fmt.Sprint(got) != fmt.Sprint(damaged) {
		t.Fatalf("dry run wrote")
	}

	rep, err = ingest.RepairMatch(ctx, s, "", matchID, graph(path), false)
	if err != nil {
		t.Fatalf("repair: %v", err)
	}
	if rep.RehomedComments != 1 {
		t.Errorf("rehomed comments = %d, want 1", rep.RehomedComments)
	}

	repaired := takeRepairSnapshot(t, s, matchID)
	if len(repaired.Moves) != len(fresh.Moves) || repaired.Analysed != fresh.Analysed {
		t.Fatalf("repaired: %d moves / %d analysed, fresh: %d / %d", len(repaired.Moves), repaired.Analysed, len(fresh.Moves), fresh.Analysed)
	}
	for k, want := range fresh.Moves {
		got := repaired.Moves[k]
		if got.Zobrist != want.Zobrist || got.Analysis != want.Analysis || got.ErrorMP != want.ErrorMP {
			t.Fatalf("move %v: repaired %+v, fresh %+v", k, got, want)
		}
		if got.Player != -want.Player {
			t.Fatalf("move %v: player %d, want the swapped seat %d", k, got.Player, -want.Player)
		}
	}
	if got := repaired.Moves[damagedMoveKey(t, s, matchID)].RollTick; got != fmt.Sprint(tick) {
		t.Errorf("the Repère of the move was lost: %v", got)
	}

	m, err := ms.Get(ctx, "", matchID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if m.Player1Name != "Edited Two" || m.Player2Name != "Edited One" || m.Comment != "my match note" ||
		m.TournamentName != "Repair Open" || m.VideoSource == nil || *m.VideoSource != "https://example.org/v" ||
		!m.MatchDate.Equal(headerBefore.MatchDate) || m.MatchHash != freshMatch.MatchHash || m.DiceHash != freshMatch.DiceHash {
		t.Errorf("header after repair = %+v", m)
	}
	if tr, err := s.Transcriptions().Get(ctx, "", transID); err != nil || tr.MatchID != matchID {
		t.Errorf("transcription after repair: %+v, %v", tr, err)
	}
	if as, err := s.Aliases().List(ctx, "", storage.AliasPlayer); err != nil || !slices.ContainsFunc(as, func(a storage.Alias) bool { return a.Alias == alias }) {
		t.Errorf("aliases after repair: %+v, %v", as, err)
	}
	var texts []string
	var ids []int64
	for mv, err := range ms.MovesByMatch(ctx, "", matchID) {
		if err != nil {
			t.Fatalf("MovesByMatch: %v", err)
		}
		ids = append(ids, mv.PositionID)
	}
	byPos, err := s.Comments().ByPositions(ctx, "", ids)
	if err != nil {
		t.Fatalf("ByPositions: %v", err)
	}
	for _, es := range byPos {
		for _, e := range es {
			texts = append(texts, e.Text)
		}
	}
	for _, want := range []string{"before the swap", "after the swap"} {
		if !slices.Contains(texts, want) {
			t.Errorf("comment %q is not on a position of the repaired match (%q)", want, texts)
		}
	}

	// Repaired, the match no longer differs from its file.
	rep, err = ingest.RepairMatch(ctx, s, "", matchID, graph(path), true)
	if err != nil {
		t.Fatalf("second dry run: %v", err)
	}
	if rep.Differs() || !rep.Swapped {
		t.Errorf("repaired match still differs: %+v", rep)
	}
}

// repairKey names a move in a snapshot: a double and its answer share a
// move number.
func repairKey(game, move int32, kind, action string) string {
	return fmt.Sprint(game, "/", move, "/", kind, "/", action)
}

// damagedMoveKey is the key of game 1's third move, the one
// damageLikeTheOldSwap gave a Repère.
func damagedMoveKey(t *testing.T, s storage.Storage, matchID int64) string {
	t.Helper()
	ctx := context.Background()
	var game1 int64
	for g, err := range s.Matches().Games(ctx, "", matchID) {
		if err != nil {
			t.Fatalf("Games: %v", err)
		}
		if g.GameNumber == 1 {
			game1 = g.ID
		}
	}
	i := 0
	for mv, err := range s.Matches().Moves(ctx, "", game1) {
		if err != nil {
			t.Fatalf("Moves: %v", err)
		}
		if i == 2 {
			return repairKey(1, mv.MoveNumber, mv.MoveType, mv.CubeAction)
		}
		i++
	}
	t.Fatalf("game 1 has fewer than three moves")
	return ""
}

func playSet(plays []string) []string {
	out := make([]string, 0, len(plays))
	for _, p := range plays {
		parts := strings.Fields(p)
		slices.Sort(parts)
		out = append(out, strings.Join(parts, " "))
	}
	slices.Sort(out)
	return slices.Compact(out)
}

func repairDeref[T int32 | int64](p *T) any {
	if p == nil {
		return nil
	}
	return *p
}

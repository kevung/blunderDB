// Contract cases for positions and analyses: save/load, dedup, provenance, blob
// compression and the repair of denormalised columns.
// The table that runs them lives in contract.go.
package storagetest

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strconv"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func testPositionSaveLoad(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	p := checkerPos()
	id, err := s.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if id == 0 {
		t.Fatal("Save returned id 0")
	}
	if p.ID != id {
		t.Errorf("Save did not set p.ID: got %d, want %d", p.ID, id)
	}

	got, err := s.Positions().Load(ctx, "", id)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.ID != id {
		t.Errorf("Load id: got %d, want %d", got.ID, id)
	}
	if got.DecisionType != domain.CheckerAction {
		t.Errorf("Load DecisionType: got %d, want %d", got.DecisionType, domain.CheckerAction)
	}
	if got.Board != p.Board {
		t.Errorf("Load board mismatch:\n got %+v\nwant %+v", got.Board, p.Board)
	}
}

func testPositionDedup(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	p1 := checkerPos()
	id1, err := s.Positions().Save(ctx, "", &p1)
	if err != nil {
		t.Fatalf("first Save: %v", err)
	}
	p2 := checkerPos()
	id2, err := s.Positions().Save(ctx, "", &p2)
	if err != nil {
		t.Fatalf("second Save: %v", err)
	}
	if id1 != id2 {
		t.Errorf("dedup failed: first Save id %d, second Save id %d", id1, id2)
	}

	n := 0
	for _, err := range s.Positions().List(ctx, "", storage.ListOpts{}) {
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		n++
	}
	if n != 1 {
		t.Errorf("after dedup expected 1 stored position, got %d", n)
	}
}

func testPositionUpdatePreservesID(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	p := checkerPos()
	id, err := s.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Mutate the board (15 checkers preserved) and update in place.
	p.Board.Points[19] = domain.Point{Checkers: 4, Color: domain.White}
	p.Board.Points[20] = domain.Point{Checkers: 1, Color: domain.White}
	if err := s.Positions().Update(ctx, "", &p); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := s.Positions().Load(ctx, "", id)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.ID != id {
		t.Errorf("Update changed id: got %d, want %d", got.ID, id)
	}
	if got.Board != p.Board {
		t.Errorf("Update did not persist board change:\n got %+v\nwant %+v", got.Board, p.Board)
	}
}

// testPositionUpdateRefusesDuplicate: editing a position until it is the same
// as another stored one must not create a second row for that position, nor
// surface the backend's constraint message. The store refuses with a
// DuplicatePositionError naming the existing row, and the edited row keeps
// its previous content.
func testPositionUpdateRefusesDuplicate(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	a := checkerPos()
	idA, err := s.Positions().Save(ctx, "", &a)
	if err != nil {
		t.Fatalf("Save a: %v", err)
	}
	b := checkerPos()
	b.Dice = [2]int{6, 5}
	idB, err := s.Positions().Save(ctx, "", &b)
	if err != nil {
		t.Fatalf("Save b: %v", err)
	}
	if idA == idB {
		t.Fatal("fixtures are not distinct positions")
	}

	edited := b
	edited.Dice = a.Dice // now identical to a
	err = s.Positions().Update(ctx, "", &edited)
	var dup *storage.DuplicatePositionError
	if !errors.As(err, &dup) {
		t.Fatalf("Update to an existing position: got %v, want *storage.DuplicatePositionError", err)
	}
	if dup.ExistingID != idA {
		t.Errorf("ExistingID = %d, want %d", dup.ExistingID, idA)
	}
	if !errors.Is(err, storage.ErrConflict) {
		t.Errorf("a duplicate position is not reported as ErrConflict: %v", err)
	}

	got, err := s.Positions().Load(ctx, "", idB)
	if err != nil {
		t.Fatalf("Load b after refused update: %v", err)
	}
	if got.Dice != b.Dice {
		t.Errorf("refused update changed the row: dice %v, want %v", got.Dice, b.Dice)
	}
}

// testPositionProvenanceSticky pins the rule that makes the individually
// imported flag usable: it is ORed into the stored value, never assigned.
// Both orderings below are ordinary user behaviour, and the flag must mean the
// same thing in each — see docs/adr/0001.
func testPositionProvenanceSticky(t *testing.T, s storage.Storage) {
	ctx := context.Background()

	flag := func(id int64) bool {
		p, err := s.Positions().Load(ctx, "", id)
		if err != nil {
			t.Fatalf("Load position %d: %v", id, err)
		}
		return p.IndividuallyImported
	}
	save := func(p domain.Position, individual bool) int64 {
		p.IndividuallyImported = individual
		id, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("Save position: %v", err)
		}
		return id
	}

	// S1 — the user imports a position on its own, then imports the match it
	// came from. The match import must not clear the flag.
	solo := save(provenancePos(1), true)
	if got := save(provenancePos(1), false); got != solo {
		t.Fatalf("dedup failed: match import created id %d, want %d", got, solo)
	}
	if !flag(solo) {
		t.Error("S1: importing the match cleared individually_imported")
	}

	// S2 — the match is imported first, then the user imports one of its
	// positions on its own. The insert is a no-op, so the flag has to be raised
	// on the row that is already there.
	fromMatch := save(provenancePos(2), false)
	if flag(fromMatch) {
		t.Error("a match-sourced position came back individually imported")
	}
	if got := save(provenancePos(2), true); got != fromMatch {
		t.Fatalf("dedup failed: individual import created id %d, want %d", got, fromMatch)
	}
	if !flag(fromMatch) {
		t.Error("S2: individually importing an already-stored position did not mark it")
	}

	// A position only ever seen inside a match stays unmarked.
	if only := save(provenancePos(3), false); flag(only) {
		t.Error("a position seen only in a match was marked individually imported")
	}
}

// testPositionListIDsAndLoadByIDs pins the pair the GUI browses a library
// with: ListIDs is List's order under List's bounds, and LoadByIDs hands the
// positions back in the caller's order, silently dropping an id that no
// longer exists.
func testPositionListIDsAndLoadByIDs(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ps := s.Positions()

	var saved []int64
	for n := 1; n <= 5; n++ {
		p := provenancePos(n)
		id, err := ps.Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("Save %d: %v", n, err)
		}
		saved = append(saved, id)
	}

	ids, err := ps.ListIDs(ctx, "", storage.ListOpts{})
	if err != nil {
		t.Fatalf("ListIDs: %v", err)
	}
	var listed []int64
	for p, err := range ps.List(ctx, "", storage.ListOpts{}) {
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		listed = append(listed, p.ID)
	}
	if !equalIDs(ids, listed) {
		t.Errorf("ListIDs order differs from List: ids %v, list %v", ids, listed)
	}

	window, err := ps.ListIDs(ctx, "", storage.ListOpts{Limit: 2, Offset: 1})
	if err != nil {
		t.Fatalf("ListIDs window: %v", err)
	}
	if !equalIDs(window, listed[1:3]) {
		t.Errorf("ListIDs{Limit:2,Offset:1}: got %v, want %v", window, listed[1:3])
	}

	// Caller's order, not id order; an unknown id is skipped, not an error.
	want := []int64{saved[3], saved[0], saved[4]}
	got, err := ps.LoadByIDs(ctx, "", []int64{saved[3], 987654321, saved[0], saved[4]})
	if err != nil {
		t.Fatalf("LoadByIDs: %v", err)
	}
	var gotIDs []int64
	for _, p := range got {
		gotIDs = append(gotIDs, p.ID)
	}
	if !equalIDs(gotIDs, want) {
		t.Errorf("LoadByIDs order: got %v, want %v", gotIDs, want)
	}
	if got[1].Score != provenancePos(1).Score {
		t.Errorf("LoadByIDs did not reconstruct the position: score %v", got[1].Score)
	}

	empty, err := ps.LoadByIDs(ctx, "", nil)
	if err != nil {
		t.Fatalf("LoadByIDs(nil): %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("LoadByIDs(nil): got %d positions, want 0", len(empty))
	}
}

func testAnalysisSaveAndCompress(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	p := checkerPos()
	posID, err := s.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}

	a := domain.PositionAnalysis{
		AnalysisType: "CheckerMove",
		CheckerAnalysis: &domain.CheckerAnalysis{
			Moves: []domain.CheckerMove{
				{Index: 0, Move: "13/11 24/23", Equity: 0.123, PlayerWinChance: 54.32},
			},
		},
	}
	if err := s.Analyses().Save(ctx, "", posID, &a); err != nil {
		t.Fatalf("Save analysis: %v", err)
	}

	// Load round-trips through the compressed data column (zstd, see engine.CompressAnalysisData).
	got, err := s.Analyses().Load(ctx, "", posID)
	if err != nil {
		t.Fatalf("Load analysis: %v", err)
	}
	if got.AnalysisType != "CheckerMove" {
		t.Errorf("AnalysisType: got %q, want %q", got.AnalysisType, "CheckerMove")
	}
	if got.PositionID != int(posID) {
		t.Errorf("PositionID: got %d, want %d", got.PositionID, posID)
	}
	if got.CheckerAnalysis == nil || len(got.CheckerAnalysis.Moves) != 1 {
		t.Fatalf("CheckerAnalysis not round-tripped: %+v", got.CheckerAnalysis)
	}
	if got.CheckerAnalysis.Moves[0].Move != "13/11 24/23" {
		t.Errorf("move: got %q, want %q", got.CheckerAnalysis.Moves[0].Move, "13/11 24/23")
	}
}

// testRepairDenormalisedColumns pins the repair on the very defect that made it
// necessary: a column left holding the error of an action that was not played.
//
// The check that matters is the SECOND run returning 0. A repair that rewrites
// every row every time cannot tell "something was wrong" from "it ran", and the
// count is the only thing an operator has to decide whether to worry.
func testRepairDenormalisedColumns(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	as := s.Analyses()

	pos := statsDecisionPos(t, 7)
	pos.DecisionType = domain.CubeAction
	posID, err := s.Positions().Save(ctx, "", &pos)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}
	a := domain.PositionAnalysis{
		PlayedCubeActions: []string{"No Double"},
		DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{
			BestCubeAction:          "No Double",
			CubefulNoDoubleEquity:   0.40,
			CubefulDoubleTakeEquity: 0.20,
			CubefulDoublePassEquity: 1.00,
			CubefulNoDoubleError:    0,
			CubefulDoubleTakeError:  -0.200,
			CubefulDoublePassError:  -0.600,
		},
	}
	if err := as.Save(ctx, "", posID, &a); err != nil {
		if errors.Is(err, storage.ErrInternal) {
			t.Skip("Analyses not implemented on this backend")
		}
		t.Fatalf("Save analysis: %v", err)
	}

	// Une base saine ne bouge pas : c'est ce qui rend le compteur lisible.
	n, err := as.RepairDenormalisedColumns(ctx, "")
	if err != nil {
		t.Fatalf("Repair (sain): %v", err)
	}
	if n != 0 {
		t.Errorf("réparation d'une base saine : %d lignes touchées, want 0", n)
	}

	// La réparation d'une colonne RÉELLEMENT abîmée se teste là où l'on peut
	// l'abîmer — en SQL, dans analyses_repair_sqlite_test.go. Le contrat, lui,
	// n'a que l'interface : il vérifie ce qu'il peut, et ce qu'il vérifie est
	// justement ce qui rend le compteur lisible.
}

// testAnalysisSaveIsAnUpsert pins the fix for issue #173: a position has ONE
// analysis, and saving twice replaces it rather than adding a second row.
//
// Save used to SELECT an existing row and then INSERT or UPDATE. Two saves
// racing on the same position both read "no row" and both inserted, and Load —
// a plain `WHERE position_id = ?` — then returned whichever the planner reached
// first, so a position could keep showing an analysis that had been superseded.
// Save is a single upsert now, over a UNIQUE index on analysis(position_id).
//
// The check that matters is the SECOND save: what Load returns afterwards must
// be the second analysis, every time, on any backend.
func testAnalysisSaveIsAnUpsert(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	p := checkerPos()
	posID, err := s.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}

	analysisWith := func(move string) *domain.PositionAnalysis {
		return &domain.PositionAnalysis{
			AnalysisType: "CheckerMove",
			CheckerAnalysis: &domain.CheckerAnalysis{
				Moves: []domain.CheckerMove{{Index: 0, Move: move, Equity: 0.1}},
			},
		}
	}

	for _, move := range []string{"13/11 24/23", "8/5 6/5", "24/18"} {
		if err := s.Analyses().Save(ctx, "", posID, analysisWith(move)); err != nil {
			t.Fatalf("Save analysis %q: %v", move, err)
		}
	}

	got, err := s.Analyses().Load(ctx, "", posID)
	if err != nil {
		t.Fatalf("Load analysis: %v", err)
	}
	if got.CheckerAnalysis == nil || len(got.CheckerAnalysis.Moves) != 1 {
		t.Fatalf("CheckerAnalysis not round-tripped: %+v", got.CheckerAnalysis)
	}
	if last := got.CheckerAnalysis.Moves[0].Move; last != "24/18" {
		t.Errorf("Load after three saves returned %q, want the last one written (%q)", last, "24/18")
	}

	// And there is exactly one row to load: Delete leaves nothing behind.
	if err := s.Analyses().Delete(ctx, "", posID); err != nil {
		t.Fatalf("Delete analysis: %v", err)
	}
	if _, err := s.Analyses().Load(ctx, "", posID); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Load after Delete: got %v, want ErrNotFound — a second row survived the upsert", err)
	}
}

// testRepairCrawfordSentinel pins the repair of issue #338 on both backends.
//
// The importers wrote away `1` on every 1-away position, Crawford game or not,
// so a post-Crawford position already stored reads as cube-dead (CONTEXT.md,
// « Away score »). Correcting it changes the Zobrist hash — the away score is
// part of the identity — so the row is rehashed, which is why this cannot be
// an UPDATE and be done with it: when the corrected position is ALREADY
// stored, the two must become one row rather than collide.
//
// Both halves are checked, plus what must not move: the Crawford game's own
// position, and a position no game points at.
func testRepairCrawfordSentinel(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ps, ms := s.Positions(), s.Matches()

	m := domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 7, MatchHash: "crawford-repair"}
	matchID, err := ms.Save(ctx, "", &m)
	if err != nil {
		t.Fatalf("Save match: %v", err)
	}
	// The Crawford game (6-2), then the game after it (6-3).
	var gameIDs [2]int64
	for i, initial := range [2][2]int32{{6, 2}, {6, 3}} {
		g := domain.Game{MatchID: matchID, GameNumber: int32(i + 1), InitialScore: initial}
		id, err := ms.CreateGame(ctx, "", &g)
		if err != nil {
			t.Fatalf("CreateGame %d: %v", i+1, err)
		}
		gameIDs[i] = id
	}
	play := func(gameID, positionID int64, n int32) {
		t.Helper()
		mv := domain.Move{GameID: gameID, MoveNumber: n, MoveType: "checker",
			PositionID: positionID, Player: 1, Dice: [2]int32{3, 1}, CheckerMove: "8/5 6/5"}
		if _, err := ms.CreateMove(ctx, "", &mv); err != nil {
			t.Fatalf("CreateMove: %v", err)
		}
	}
	save := func(what string, p domain.Position) int64 {
		t.Helper()
		id, err := ps.Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("Save %s: %v", what, err)
		}
		return id
	}

	// In the Crawford game, away [1, 5] is right and must not move.
	inCrawford := statsDecisionPos(t, 0)
	inCrawford.Score = [2]int{domain.Crawford, 5}
	crawfordID := save("the Crawford game's position", inCrawford)
	play(gameIDs[0], crawfordID, 1)

	// In the game after it, the same away score is the importers' bug.
	stale := statsDecisionPos(t, 1)
	stale.Score = [2]int{domain.Crawford, 4}
	staleID := save("the stale post-Crawford position", stale)
	play(gameIDs[1], staleID, 1)

	// A second stale one whose corrected twin is already stored: the merge.
	twinBoard := statsDecisionPos(t, 2)
	twinBoard.Score = [2]int{domain.PostCrawford, 4}
	twinID := save("the correct twin", twinBoard)
	staleTwin := statsDecisionPos(t, 2)
	staleTwin.Score = [2]int{domain.Crawford, 4}
	staleTwinID := save("the stale twin", staleTwin)
	if staleTwinID == twinID {
		t.Fatal("the two away scores hashed to one row; the case proves nothing")
	}
	play(gameIDs[1], staleTwinID, 2)

	// And one no game points at: away 1 is then nobody's mistake.
	loose := statsDecisionPos(t, 3)
	loose.Score = [2]int{domain.Crawford, 3}
	looseID := save("the loose position", loose)

	n, err := ps.RepairCrawfordSentinel(ctx, "")
	if err != nil {
		t.Fatalf("RepairCrawfordSentinel: %v", err)
	}
	if n != 2 {
		t.Errorf("repaired %d positions, want 2 (the two post-Crawford ones)", n)
	}

	got, err := ps.Load(ctx, "", staleID)
	if err != nil {
		t.Fatalf("Load the repaired position: %v", err)
	}
	if got.Score != [2]int{domain.PostCrawford, 4} {
		t.Errorf("away score after repair = %v, want [0 4]", got.Score)
	}
	// The merged one is gone, and its move now names the survivor.
	if _, err := ps.Load(ctx, "", staleTwinID); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("the merged position %d is still stored (err=%v)", staleTwinID, err)
	}
	for mv, err := range ms.Moves(ctx, "", gameIDs[1]) {
		if err != nil {
			t.Fatalf("Moves: %v", err)
		}
		if mv.MoveNumber == 2 && mv.PositionID != twinID {
			t.Errorf("the merged move names position %d, want the survivor %d", mv.PositionID, twinID)
		}
	}
	// What must not move.
	for id, want := range map[int64][2]int{
		crawfordID: {domain.Crawford, 5},
		looseID:    {domain.Crawford, 3},
	} {
		p, err := ps.Load(ctx, "", id)
		if err != nil {
			t.Fatalf("Load %d: %v", id, err)
		}
		if p.Score != want {
			t.Errorf("position %d was rewritten to %v, want %v", id, p.Score, want)
		}
	}

	// Idempotent: everything now says what it means.
	if again, err := ps.RepairCrawfordSentinel(ctx, ""); err != nil || again != 0 {
		t.Errorf("second pass: repaired=%d err=%v, want 0 and no error", again, err)
	}
}

// testRepairCrawfordMergeCarriesDependents pins what the Crawford repair's
// merge carries when a stale row folds into its correct twin: everything that
// hangs off a position today, on both backends.
//
// The list is the schema's, not the September one: moves and comments,
// collection memberships (one kept where both rows were members), Anki cards
// with the KEY that names the position inside its deck (ADR-0042), the review
// journal — including the reviews of a card dropped because the deck already
// held the twin's — and the trash entries that name the stale row by id.
func testRepairCrawfordMergeCarriesDependents(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ps, ms, cs, anki := s.Positions(), s.Matches(), s.Collections(), s.Anki()

	m := domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 7, MatchHash: "crawford-merge"}
	matchID, err := ms.Save(ctx, "", &m)
	if err != nil {
		t.Fatalf("Save match: %v", err)
	}
	// The Crawford game (6-2), then the post-Crawford one (6-3) the stale row
	// was played in.
	var postCrawfordGame int64
	for i, initial := range [2][2]int32{{6, 2}, {6, 3}} {
		g := domain.Game{MatchID: matchID, GameNumber: int32(i + 1), InitialScore: initial}
		gameID, err := ms.CreateGame(ctx, "", &g)
		if err != nil {
			t.Fatalf("CreateGame %d: %v", i+1, err)
		}
		postCrawfordGame = gameID
	}

	twin := statsDecisionPos(t, 0)
	twin.Score = [2]int{domain.PostCrawford, 4}
	twinID, err := ps.Save(ctx, "", &twin)
	if err != nil {
		t.Fatalf("Save twin: %v", err)
	}
	stale := statsDecisionPos(t, 0)
	stale.Score = [2]int{domain.Crawford, 4}
	staleID, err := ps.Save(ctx, "", &stale)
	if err != nil {
		t.Fatalf("Save stale: %v", err)
	}
	if staleID == twinID {
		t.Fatal("the two away scores hashed to one row; the case proves nothing")
	}
	mv := domain.Move{GameID: postCrawfordGame, MoveNumber: 1, MoveType: "checker",
		PositionID: staleID, Player: 1, Dice: [2]int32{3, 1}, CheckerMove: "8/5 6/5"}
	if _, err := ms.CreateMove(ctx, "", &mv); err != nil {
		t.Fatalf("CreateMove: %v", err)
	}

	// Collections: one holds both rows, one only the stale row.
	both, err := cs.Create(ctx, "", "both", "")
	if err != nil {
		t.Fatalf("Create collection: %v", err)
	}
	if err := cs.AddPositions(ctx, "", both, []int64{staleID, twinID}); err != nil {
		t.Fatalf("AddPositions: %v", err)
	}
	staleOnly, err := cs.Create(ctx, "", "stale only", "")
	if err != nil {
		t.Fatalf("Create collection: %v", err)
	}
	if err := cs.AddPosition(ctx, "", staleOnly, staleID); err != nil {
		t.Fatalf("AddPosition: %v", err)
	}

	// Decks: the same shape, and the stale card is reviewed in each, so its
	// journal has to survive both the re-pointing and the drop.
	deckBoth, err := anki.CreateDeck(ctx, "", "both", "", domain.AnkiSourceSearch, 0, "")
	if err != nil {
		t.Fatalf("CreateDeck: %v", err)
	}
	if err := anki.SyncWithPositions(ctx, "", deckBoth, []int64{twinID, staleID}); err != nil {
		t.Fatalf("SyncWithPositions: %v", err)
	}
	deckStale, err := anki.CreateDeck(ctx, "", "stale only", "", domain.AnkiSourceSearch, 0, "")
	if err != nil {
		t.Fatalf("CreateDeck: %v", err)
	}
	if err := anki.SyncWithPositions(ctx, "", deckStale, []int64{staleID}); err != nil {
		t.Fatalf("SyncWithPositions: %v", err)
	}
	reviewStaleCard := func(deckID int64) {
		t.Helper()
		for range 2 {
			next, err := anki.NextCard(ctx, "", deckID)
			if err != nil {
				t.Fatalf("NextCard deck %d: %v", deckID, err)
			}
			if next.Card.PositionID != staleID {
				if err := anki.BuryCard(ctx, "", next.Card.ID); err != nil {
					t.Fatalf("BuryCard: %v", err)
				}
				continue
			}
			if _, err := anki.ReviewCard(ctx, "", next.Card.ID, 3); err != nil {
				t.Fatalf("ReviewCard: %v", err)
			}
			return
		}
		t.Fatalf("deck %d never served the stale card", deckID)
	}
	reviewStaleCard(deckBoth)
	reviewStaleCard(deckStale)

	if _, err := s.Comments().Add(ctx, "", staleID, "the trailer doubles here"); err != nil {
		t.Fatalf("Add comment: %v", err)
	}

	// Trash entries naming the stale row by id.
	putTrash := func(kind domain.TrashKind, payload any) int64 {
		t.Helper()
		blob, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		id, err := s.Trash().Put(ctx, "", kind, "entry", blob)
		if err != nil {
			t.Fatalf("Trash Put: %v", err)
		}
		return id
	}
	trashedComment := putTrash(domain.TrashComment, domain.TrashCommentPayload{
		Comment: domain.CommentEntry{PositionID: staleID, Text: "deleted note"}})
	trashedCollection := putTrash(domain.TrashCollection, domain.TrashCollectionPayload{
		Name: "deleted", PositionIDs: []int64{staleID, twinID}})

	n, err := ps.RepairCrawfordSentinel(ctx, "")
	if err != nil {
		t.Fatalf("RepairCrawfordSentinel: %v", err)
	}
	if n != 1 {
		t.Fatalf("repaired %d positions, want 1", n)
	}
	if _, err := ps.Load(ctx, "", staleID); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("the merged position %d is still stored (err=%v)", staleID, err)
	}

	for mv, err := range ms.Moves(ctx, "", postCrawfordGame) {
		if err != nil {
			t.Fatalf("Moves: %v", err)
		}
		if mv.PositionID != twinID {
			t.Errorf("the move names position %d, want the survivor %d", mv.PositionID, twinID)
		}
	}

	members := func(collectionID int64) []int64 {
		t.Helper()
		var ids []int64
		for cp, err := range cs.Members(ctx, "", collectionID) {
			if err != nil {
				t.Fatalf("Members: %v", err)
			}
			ids = append(ids, cp.PositionID)
		}
		return ids
	}
	if got := members(both); !slices.Equal(got, []int64{twinID}) {
		t.Errorf("collection holding both rows = %v, want the survivor once [%d]", got, twinID)
	}
	if got := members(staleOnly); !slices.Equal(got, []int64{twinID}) {
		t.Errorf("collection holding the stale row = %v, want [%d]", got, twinID)
	}

	for _, deckID := range []int64{deckBoth, deckStale} {
		assertDeckFollowsSurvivor(t, s, deckID, twinID)
	}

	var comments []string
	for c, err := range s.Comments().ByPosition(ctx, "", twinID) {
		if err != nil {
			t.Fatalf("ByPosition: %v", err)
		}
		comments = append(comments, c.Text)
	}
	if !slices.Contains(comments, "the trailer doubles here") {
		t.Errorf("the survivor's comments = %q, want the stale row's note", comments)
	}

	assertTrashFollowsSurvivor(t, s, trashedComment, trashedCollection, twinID)
}

// assertDeckFollowsSurvivor checks one deck after the merge: a single card for
// the survivor, a journal of one review naming it by id and by key, and a key
// that a later sync recognises instead of adding a second card.
func assertDeckFollowsSurvivor(t *testing.T, s storage.Storage, deckID, twinID int64) {
	t.Helper()
	ctx := context.Background()
	anki := s.Anki()
	deckPositions := func() []int64 {
		t.Helper()
		var ids []int64
		for p, err := range anki.DeckPositions(ctx, "", deckID) {
			if err != nil {
				t.Fatalf("DeckPositions: %v", err)
			}
			ids = append(ids, p.ID)
		}
		return ids
	}
	if got := deckPositions(); !slices.Equal(got, []int64{twinID}) {
		t.Errorf("deck %d holds %v, want one card for the survivor [%d]", deckID, got, twinID)
	}
	var logs []*domain.AnkiReviewLog
	for l, err := range anki.ReviewLog(ctx, "", deckID, 10) {
		if err != nil {
			t.Fatalf("ReviewLog: %v", err)
		}
		logs = append(logs, l)
	}
	if len(logs) != 1 {
		t.Fatalf("deck %d journal has %d reviews, want the stale card's 1", deckID, len(logs))
	}
	twinKey := strconv.FormatInt(twinID, 10)
	if logs[0].PositionID != twinID || logs[0].Key != twinKey {
		t.Errorf("deck %d journal names position %d key %q, want %d %q",
			deckID, logs[0].PositionID, logs[0].Key, twinID, twinKey)
	}
	// The key moved with the card: a sync naming the survivor finds its card
	// instead of adding a second one.
	if err := anki.SyncWithPositions(ctx, "", deckID, []int64{twinID}); err != nil {
		t.Fatalf("SyncWithPositions after repair: %v", err)
	}
	if got := deckPositions(); !slices.Equal(got, []int64{twinID}) {
		t.Errorf("deck %d after a sync holds %v, want [%d]: the card key did not follow", deckID, got, twinID)
	}
}

// assertTrashFollowsSurvivor checks that the trashed comment and the trashed
// collection that named the merged row now name the survivor, once.
func assertTrashFollowsSurvivor(t *testing.T, s storage.Storage, commentEntry, collectionEntry, twinID int64) {
	t.Helper()
	ctx := context.Background()
	entry, err := s.Trash().Load(ctx, "", commentEntry)
	if err != nil {
		t.Fatalf("Trash Load: %v", err)
	}
	var cp domain.TrashCommentPayload
	if err := json.Unmarshal(entry.Payload, &cp); err != nil {
		t.Fatalf("decode comment payload: %v", err)
	}
	if cp.Comment.PositionID != twinID || cp.Comment.Text != "deleted note" {
		t.Errorf("trashed comment names position %d (%q), want %d", cp.Comment.PositionID, cp.Comment.Text, twinID)
	}
	entry, err = s.Trash().Load(ctx, "", collectionEntry)
	if err != nil {
		t.Fatalf("Trash Load: %v", err)
	}
	var colp domain.TrashCollectionPayload
	if err := json.Unmarshal(entry.Payload, &colp); err != nil {
		t.Fatalf("decode collection payload: %v", err)
	}
	if !slices.Equal(colp.PositionIDs, []int64{twinID}) || colp.Name != "deleted" {
		t.Errorf("trashed collection = %q %v, want \"deleted\" [%d]", colp.Name, colp.PositionIDs, twinID)
	}
}

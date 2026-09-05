package migrate

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// TestRunRemapsIdentifiers exercises the actual id-remap logic of Run — not
// just the tenant-validation guard tenant_test.go covers — on two in-memory
// SQLite storages, no Docker required. It builds a small src database that
// exercises every family Run copies (positions with an analysis and a
// comment, a tournament, a match with a game and a move referencing one of
// the positions, and a collection holding the other position) then checks
// that the destination's foreign keys were rewritten to the destination's own
// ids rather than blindly carried over from the source.
func TestRunRemapsIdentifiers(t *testing.T) {
	ctx := context.Background()

	src, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	if err := src.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	// Two distinct positions (different dice, so distinct Zobrist hashes —
	// see engine.ZobristHash — and no accidental dedup). posAnalysed carries
	// the analysis, posCollected ends up in a collection and is the one the
	// match's move references.
	posAnalysed := domain.InitializePosition()
	posAnalysed.DecisionType = domain.CheckerAction
	idAnalysedSrc, err := src.Positions().Save(ctx, "", &posAnalysed)
	if err != nil {
		t.Fatalf("save posAnalysed: %v", err)
	}

	posCollected := domain.InitializePosition()
	posCollected.DecisionType = domain.CheckerAction
	posCollected.Dice = [2]int{6, 5}
	idCollectedSrc, err := src.Positions().Save(ctx, "", &posCollected)
	if err != nil {
		t.Fatalf("save posCollected: %v", err)
	}
	if idAnalysedSrc == idCollectedSrc {
		t.Fatal("the two fixture positions collapsed to the same id — they must be distinct for this test to mean anything")
	}

	analysis := domain.PositionAnalysis{
		AnalysisType: "CheckerMove",
		CheckerAnalysis: &domain.CheckerAnalysis{
			Moves: []domain.CheckerMove{{Index: 0, Move: "13/11 24/23", Equity: 0.123, PlayerWinChance: 54.32}},
		},
	}
	if err := src.Analyses().Save(ctx, "", idAnalysedSrc, &analysis); err != nil {
		t.Fatalf("save analysis: %v", err)
	}
	if _, err := src.Comments().Add(ctx, "", idCollectedSrc, "a note"); err != nil {
		t.Fatalf("add comment: %v", err)
	}

	tournID, err := src.Tournaments().Create(ctx, "", "Open", "2026-01-01", "Paris")
	if err != nil {
		t.Fatalf("create tournament: %v", err)
	}

	match := domain.Match{
		Player1Name:  "Player One",
		Player2Name:  "Player Two",
		MatchLength:  7,
		TournamentID: &tournID,
	}
	matchID, err := src.Matches().Save(ctx, "", &match)
	if err != nil {
		t.Fatalf("save match: %v", err)
	}

	game := domain.Game{MatchID: matchID, GameNumber: 1, InitialScore: [2]int32{0, 0}}
	gameID, err := src.Matches().CreateGame(ctx, "", &game)
	if err != nil {
		t.Fatalf("create game: %v", err)
	}

	move := domain.Move{
		GameID:      gameID,
		MoveNumber:  0,
		MoveType:    "checker",
		PositionID:  idAnalysedSrc,
		Player:      1,
		Dice:        [2]int32{3, 1},
		CheckerMove: "13/11 24/23",
	}
	if _, err := src.Matches().CreateMove(ctx, "", &move); err != nil {
		t.Fatalf("create move: %v", err)
	}

	collID, err := src.Collections().Create(ctx, "", "Favourites", "kept for later")
	if err != nil {
		t.Fatalf("create collection: %v", err)
	}
	if err := src.Collections().AddPositions(ctx, "", collID, []int64{idCollectedSrc}); err != nil {
		t.Fatalf("add position to collection: %v", err)
	}

	dst, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer dst.Close()
	if err := dst.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	rep, err := Run(ctx, src, dst, "", Options{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	want := Report{Positions: 2, Analyses: 1, Comments: 1, Tournaments: 1, Matches: 1, Games: 1, Moves: 1, Collections: 1}
	if rep != want {
		t.Fatalf("Report = %+v, want %+v", rep, want)
	}

	// Find the two positions in dst by the dice that told them apart in src —
	// their ids are necessarily different from the source's (a fresh
	// destination starts its own autoincrement), which is exactly what a
	// remap bug (an id blindly carried over, or the two swapped) would hide
	// if this test compared old and new ids instead of content.
	var idAnalysedDst, idCollectedDst int64
	for p, err := range dst.Positions().List(ctx, "", storage.ListOpts{}) {
		if err != nil {
			t.Fatalf("list dst positions: %v", err)
		}
		switch p.Dice {
		case [2]int{3, 1}:
			idAnalysedDst = p.ID
		case [2]int{6, 5}:
			idCollectedDst = p.ID
		default:
			t.Fatalf("unexpected position in dst: dice %v", p.Dice)
		}
	}
	if idAnalysedDst == 0 || idCollectedDst == 0 {
		t.Fatalf("did not find both fixture positions in dst (analysed=%d, collected=%d)", idAnalysedDst, idCollectedDst)
	}

	gotAnalysis, err := dst.Analyses().Load(ctx, "", idAnalysedDst)
	if err != nil {
		t.Fatalf("load analysis in dst: %v", err)
	}
	if gotAnalysis.CheckerAnalysis == nil || len(gotAnalysis.CheckerAnalysis.Moves) != 1 ||
		gotAnalysis.CheckerAnalysis.Moves[0].Move != "13/11 24/23" {
		t.Errorf("analysis in dst does not match: %+v", gotAnalysis)
	}
	if _, err := dst.Analyses().Load(ctx, "", idCollectedDst); err == nil {
		t.Error("posCollected should carry no analysis in dst, but Load succeeded")
	}

	var comments []string
	for c, err := range dst.Comments().ByPosition(ctx, "", idCollectedDst) {
		if err != nil {
			t.Fatalf("list comments in dst: %v", err)
		}
		comments = append(comments, c.Text)
	}
	if len(comments) != 1 || comments[0] != "a note" {
		t.Errorf("comments on posCollected in dst = %v, want [\"a note\"]", comments)
	}

	var dstMatchID int64
	for m, err := range dst.Matches().List(ctx, "", storage.MatchListOpts{}) {
		if err != nil {
			t.Fatalf("list dst matches: %v", err)
		}
		dstMatchID = m.ID
	}
	if dstMatchID == 0 {
		t.Fatal("no match found in dst")
	}

	tr, err := dst.Tournaments().TournamentOf(ctx, "", dstMatchID)
	if err != nil {
		t.Fatalf("TournamentOf: %v", err)
	}
	if tr.Name != "Open" {
		t.Errorf("dst match's tournament = %q, want %q", tr.Name, "Open")
	}

	var dstGameID int64
	gameCount := 0
	for g, err := range dst.Matches().Games(ctx, "", dstMatchID) {
		if err != nil {
			t.Fatalf("list dst games: %v", err)
		}
		dstGameID = g.ID
		gameCount++
	}
	if gameCount != 1 {
		t.Fatalf("dst match has %d games, want 1", gameCount)
	}

	moveCount := 0
	for mv, err := range dst.Matches().Moves(ctx, "", dstGameID) {
		if err != nil {
			t.Fatalf("list dst moves: %v", err)
		}
		moveCount++
		if mv.PositionID != idAnalysedDst {
			t.Errorf("dst move references position %d, want the remapped %d (posAnalysed)", mv.PositionID, idAnalysedDst)
		}
	}
	if moveCount != 1 {
		t.Fatalf("dst game has %d moves, want 1", moveCount)
	}

	// Collect the collection's id from a fully-drained List() before issuing
	// the nested Positions() query: the sqlite backend pins a :memory: DSN to
	// a single pooled connection (storage/sqlite's ConfigurePool — see
	// CLAUDE.md "Notes & Gotchas"), and List's iterator holds that connection
	// open for as long as its range loop runs. Calling Positions() — a second
	// query — from inside that loop would need a second connection that the
	// pool can never hand out, deadlocking forever (storagetest's own helpers,
	// e.g. contract_collections.go's collectionIDs/collectionPositionIDs,
	// follow the same two-pass shape for the same reason).
	var dstCollID, dstCollCount int64
	var dstCollName string
	for coll, err := range dst.Collections().List(ctx, "") {
		if err != nil {
			t.Fatalf("list dst collections: %v", err)
		}
		if coll.Name == "Favourites" {
			dstCollID, dstCollName = coll.ID, coll.Name
			dstCollCount++
		}
	}
	if dstCollCount != 1 {
		t.Fatalf(`collection "Favourites" found %d times in dst, want 1`, dstCollCount)
	}

	var members []int64
	for p, err := range dst.Collections().Positions(ctx, "", dstCollID, storage.ListOpts{}) {
		if err != nil {
			t.Fatalf("list dst collection positions: %v", err)
		}
		members = append(members, p.ID)
	}
	if len(members) != 1 || members[0] != idCollectedDst {
		t.Errorf("dst collection %q members = %v, want [%d]", dstCollName, members, idCollectedDst)
	}
}

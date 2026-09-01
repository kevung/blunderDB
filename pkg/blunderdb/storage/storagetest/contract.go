// Package storagetest provides a backend-agnostic contract test suite for
// implementations of storage.Storage. Each backend (SQLite now, PostgreSQL
// later) runs the same suite against a fresh instance:
//
//	func TestContract_SQLite(t *testing.T) {
//	    storagetest.RunContractTests(t, func() storage.Storage {
//	        s, _ := sqlite.Open(context.Background(), ":memory:", nil)
//	        return s
//	    })
//	}
//
// It lives in a regular (non-_test.go) file so it can be imported by the
// backend packages' test binaries — an exported helper in a _test.go file
// would not be visible across packages.
//
// This file holds the case table and the fixtures several families share;
// the cases themselves live in one contract_<family>.go per family
// (positions, match, match_delete, tournament, search, collections,
// comments, anki, stats, stats_detail, misc). A new case is registered here and written in
// its family's file.
package storagetest

import (
	"context"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// RunContractTests exercises a storage.Storage implementation. factory must
// return a fresh, empty, migrated Storage on every call.
func RunContractTests(t *testing.T, factory func() storage.Storage) {
	t.Helper()

	cases := []struct {
		name string
		fn   func(t *testing.T, s storage.Storage)
	}{
		{"Position/Save+Load", testPositionSaveLoad},
		{"Position/DedupByZobrist", testPositionDedup},
		{"Position/UpdatePreservesId", testPositionUpdatePreservesId},
		{"Position/ProvenanceIsSticky", testPositionProvenanceSticky},
		{"Search/FilterByIndividuallyImported", testSearchFilterByIndividuallyImported},
		{"Search/FilterByCommentPresence", testSearchFilterByCommentPresence},
		{"Search/FilterByFlagged", testSearchFilterByFlagged},
		{"Analysis/SaveAndCompress", testAnalysisSaveAndCompress},
		{"Match/CreateGameMoveCascade", testMatchCreateGameMove},
		{"Match/DeleteCascade", testMatchDeleteCascade},
		{"Match/DeleteCascadeRetention", testMatchDeleteCascadeRetention},
		{"Match/FindByHash", testMatchFindByHash},
		{"Match/SwapCopyOnWrite", testMatchSwapCopyOnWrite},
		{"Match/ListFilterSortPaginate", testMatchListFilterSortPaginate},
		{"Tournament/AddRemoveMatch", testTournamentAddRemoveMatch},
		{"Collection/MoveBetweenCollections", testCollectionMoveBetween},
		{"Collection/CopyPosition", testCollectionCopyPosition},
		{"Collection/ReorderPositions", testCollectionReorderPositions},
		{"Collection/RenameAndDelete", testCollectionRenameAndDelete},
		{"Collection/PositionIndexMap", testCollectionPositionIndexMap},
		{"Comment/CRUD", testCommentCRUD},
		{"Comment/SearchAcrossPositions", testCommentSearchAcrossPositions},
		{"Comment/PositionDeleteCascades", testCommentPositionDeleteCascades},
		{"Anki/ReviewUpdatesScheduling", testAnkiReviewUpdatesScheduling},
		{"Filter/SaveAndList", testFilterSaveAndList},
		{"History/SaveLoadClear", testCommandHistory},
		{"SearchHistory/SaveListDelete", testSearchHistory},
		{"Scope/HistoryAndFilterIsolation", testScopeIsolation},
		{"Metadata/Counts", testMetadataCounts},
		{"Session/SaveLoadEmpty", testSessionSaveLoad},
		{"Session/MultiScopeIsolation", testSessionMultiScope},
		{"Search/FilterByDecisionType", testSearchFilterByDecisionType},
		{"Search/FilterByCubeResponse", testSearchFilterByCubeResponse},
		{"Search/FilterByAnalysisDecodesCompressedBlob", testSearchFilterByAnalysisDecodesCompressedBlob},
		{"Match/MoveLuckRoundTrip", testMoveLuckRoundTrip},
		{"Stats/AggregateCounts", testStatsAggregateCounts},
		{"Stats/CubeDirections", testStatsCubeDirections},
		{"Analyses/RepairDenormalisedColumns", testRepairDenormalisedColumns},
		{"Stats/MatchDetail", testStatsMatchDetail},
		{"Stats/SnowieDenominatorCountsBothPlayers", testStatsSnowieDenominator},
		{"Stats/PlayerTable", testStatsPlayerTable},
		{"Stats/PositionIDsByMatch", testStatsPositionIDsByMatch},
		{"Stats/PositionIDsByTournament", testStatsPositionIDsByTournament},
		{"Errors/DanglingReferenceIsNotFound", testDanglingReferenceIsNotFound},
		{"Errors/MergePlayersRejectsEmptyAsInvalid", testMergePlayersRejectsEmptyAsInvalid},
		{"Tx/RollbackUndoes", testTxRollbackUndoes},
		{"Tx/CommitPersists", testTxCommitPersists},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.fn == nil {
				t.Skip("contract case pending storage implementation (P2 PRs 3-6)")
			}
			s := factory()
			defer s.Close()
			tc.fn(t, s)
		})
	}
}

// checkerPos returns a fresh starting position flagged as a checker decision.
func checkerPos() domain.Position {
	p := domain.InitializePosition()
	p.DecisionType = domain.CheckerAction
	return p
}

// cubePos returns a cube-decision position with a board distinct from
// checkerPos so it hashes (and therefore stores) as its own row.
func cubePos() domain.Position {
	p := domain.InitializePosition()
	p.DecisionType = domain.CubeAction
	p.Board.Points[1] = domain.Point{Checkers: 1, Color: domain.White}
	p.Board.Points[3] = domain.Point{Checkers: 1, Color: domain.White}
	return p
}

// provenancePos returns a position unique to n (the score is part of a
// position's identity, so each n hashes to its own row).
func provenancePos(n int) domain.Position {
	p := domain.InitializePosition()
	p.DecisionType = domain.CheckerAction
	p.Score = [2]int{n, 0}
	return p
}

func statsDecisionPos(t *testing.T, slot int) domain.Position {
	t.Helper()
	if slot < 0 || slot >= len(statsDicePairs) {
		t.Fatalf("statsDecisionPos: slot %d out of range (only %d fixture decisions supported)", slot, len(statsDicePairs))
	}
	p := domain.InitializePosition()
	p.DecisionType = domain.CheckerAction
	p.Score = [2]int{4, 4}
	d := statsDicePairs[slot]
	p.Dice = [2]int{d[0], d[1]}
	return p
}

// statsFixtureMatch saves a match with one non-forced, counted checker
// decision per player. slotBase and slotBase+1 pick each decision's dice (see
// statsDecisionPos), so callers passing disjoint slot ranges can build
// several fixtures in the same storage instance without their positions
// colliding. Each decision carries an analysis with two candidate moves and a
// known equity error on the one actually played, which is what
// statsCountedExpr/MatchDetail need to treat it as real data instead of an
// empty aggregate. It returns the match id and the two decisions' position
// ids (index 0 = player 1, index 1 = player 2).
func statsFixtureMatch(t *testing.T, s storage.Storage, slotBase int, p1, p2 string) (matchID int64, posIDs [2]int64) {
	t.Helper()
	ctx := context.Background()

	m := domain.Match{Player1Name: p1, Player2Name: p2, MatchLength: 7,
		MatchDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)}
	matchID, err := s.Matches().Save(ctx, "", &m)
	if err != nil {
		t.Fatalf("Save match: %v", err)
	}
	g := domain.Game{MatchID: matchID, GameNumber: 1, Winner: 1, PointsWon: 1}
	gameID, err := s.Matches().CreateGame(ctx, "", &g)
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}

	mkDecision := func(slot int, player int32) int64 {
		pos := statsDecisionPos(t, slot)
		posID, err := s.Positions().Save(ctx, "", &pos)
		if err != nil {
			t.Fatalf("Save position (slot %d): %v", slot, err)
		}
		mv := domain.Move{GameID: gameID, MoveNumber: int32(slot), MoveType: "checker",
			PositionID: posID, Player: player, CheckerMove: "13/11 24/23"}
		if _, err := s.Matches().CreateMove(ctx, "", &mv); err != nil {
			t.Fatalf("CreateMove (slot %d): %v", slot, err)
		}
		equityError := 0.05
		a := domain.PositionAnalysis{
			PlayedMoves: []string{"13/11 24/23"},
			CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
				{Move: "8/6 6/4", Equity: 0.50},
				{Move: "13/11 24/23", Equity: 0.45, EquityError: &equityError},
			}},
		}
		if err := s.Analyses().Save(ctx, "", posID, &a); err != nil {
			t.Fatalf("Save analysis (slot %d): %v", slot, err)
		}
		return posID
	}

	posIDs[0] = mkDecision(slotBase, 1)
	posIDs[1] = mkDecision(slotBase+1, -1)
	return matchID, posIDs
}

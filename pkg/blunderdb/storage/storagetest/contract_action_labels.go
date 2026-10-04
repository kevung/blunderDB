package storagetest

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// unlistedActionLabels are labels domain.ActionCode lacks: the storage must
// register them and read them back verbatim.
var unlistedActionLabels = []string{"Unknown(-1)", "Doppel, Annahme", "it's a pass"}

// testActionLabelsReadBackVerbatim writes every action label — the fixed
// list, "" and labels outside it — into move.move_type and move.cube_action,
// and requires each to come back as written (ADR-0071).
func testActionLabelsReadBackVerbatim(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ms := s.Matches()
	matchID, err := ms.Save(ctx, "", &domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 7})
	if err != nil {
		t.Fatal(err)
	}
	gameID, err := ms.CreateGame(ctx, "", &domain.Game{MatchID: matchID, GameNumber: 1})
	if err != nil {
		t.Fatal(err)
	}
	labels := append(slices.Clone(domain.ActionLabels()), unlistedActionLabels...)
	want := map[int64]string{}
	var posIDs []int64
	for i, label := range labels {
		p := provenancePos(i + 1)
		pid, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("Save position %d: %v", i, err)
		}
		posIDs = append(posIDs, pid)
		// The reversed label in move_type proves the two columns are coded
		// independently.
		other := labels[len(labels)-1-i]
		mv := domain.Move{GameID: gameID, MoveNumber: int32(i + 1), MoveType: other, PositionID: pid, Player: 1, CubeAction: label}
		id, err := ms.CreateMove(ctx, "", &mv)
		if err != nil {
			t.Fatalf("CreateMove(%q): %v", label, err)
		}
		want[id] = label
	}
	got, err := ms.MovesByPositions(ctx, "", posIDs)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for i, pid := range posIDs {
		for _, mv := range got[pid] {
			n++
			if mv.CubeAction != want[mv.ID] || mv.MoveType != labels[len(labels)-1-i] {
				t.Errorf("move %d: cube_action %q, move_type %q; want %q, %q",
					mv.ID, mv.CubeAction, mv.MoveType, want[mv.ID], labels[len(labels)-1-i])
			}
		}
	}
	if n != len(labels) {
		t.Errorf("read back %d moves, want %d", n, len(labels))
	}
}

// testActionLabelSelectsByBestCubeAction stores an analysis whose best cube
// action is outside the fixed list and requires the statistics selection by
// that label to find its position, and no other label to.
func testActionLabelSelectsByBestCubeAction(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ms := s.Matches()
	matchID, err := ms.Save(ctx, "", &domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 7,
		MatchDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	gameID, err := ms.CreateGame(ctx, "", &domain.Game{MatchID: matchID, GameNumber: 1, Winner: 1, PointsWon: 1})
	if err != nil {
		t.Fatal(err)
	}
	byLabel := map[string]int64{}
	for i, best := range []string{"No Double", unlistedActionLabels[0]} {
		pos := statsDecisionPos(t, i)
		pos.DecisionType = domain.CubeAction
		pid, err := s.Positions().Save(ctx, "", &pos)
		if err != nil {
			t.Fatal(err)
		}
		mv := domain.Move{GameID: gameID, MoveNumber: int32(i + 1), MoveType: "cube", PositionID: pid, Player: 1, CubeAction: "Double"}
		if _, err := ms.CreateMove(ctx, "", &mv); err != nil {
			t.Fatal(err)
		}
		a := domain.PositionAnalysis{AnalysisType: "DoublingCube",
			DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{AnalysisDepth: "4-ply", BestCubeAction: best}}
		if err := s.Analyses().Save(ctx, "", pid, &a); err != nil {
			if errors.Is(err, storage.ErrInternal) {
				t.Skip("Analyses not implemented on this backend")
			}
			t.Fatal(err)
		}
		byLabel[best] = pid
	}
	for label, pid := range byLabel {
		ids, err := s.Stats().PositionIDsBySelection(ctx, "", storage.StatsFilter{DecisionType: -1}, storage.SelectionSpec{Kind: "cube_action", CubeAction: label})
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(ids, []int64{pid}) {
			t.Errorf("selection by best cube action %q = %v, want [%d]", label, ids, pid)
		}
	}
}

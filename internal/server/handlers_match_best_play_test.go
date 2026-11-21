package server

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// TestMatchMovesBestPlayHTTP: matches.moves and matches.movesByMatch carry
// best_move and best_cube_action from the stored analysis, and leave them out
// on a move whose Position was never analysed.
func TestMatchMovesBestPlayHTTP(t *testing.T) {
	ts, srv := newTestServerAndHandler(t)
	st := srv.opts.Storage
	ctx := context.Background()

	checkerPos := domain.InitializePosition()
	checkerPos.DecisionType = domain.CheckerAction
	checkerPos.Dice = [2]int{3, 1}
	checkerID, err := st.Positions().Save(ctx, testTenant, &checkerPos)
	if err != nil {
		t.Fatal(err)
	}
	e := 0.080
	if err := st.Analyses().Save(ctx, testTenant, checkerID, &domain.PositionAnalysis{AnalysisType: "CheckerMove",
		CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
			{Move: "8/5 6/5", Equity: 0.150},
			{Move: "24/21 6/5", Equity: 0.070, EquityError: &e},
		}}}); err != nil {
		t.Fatal(err)
	}
	cubePos := domain.InitializePosition()
	cubePos.DecisionType = domain.CubeAction
	cubeID, err := st.Positions().Save(ctx, testTenant, &cubePos)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Analyses().Save(ctx, testTenant, cubeID, &domain.PositionAnalysis{AnalysisType: "DoublingCube",
		DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{BestCubeAction: "No Double",
			CubefulNoDoubleEquity: 0.300, CubefulDoubleTakeEquity: 0.100, CubefulDoublePassEquity: 1.000,
			CubefulDoubleTakeError: -0.200, CubefulDoublePassError: 0.700}}); err != nil {
		t.Fatal(err)
	}
	barePos := domain.InitializePosition()
	barePos.DecisionType = domain.CheckerAction
	barePos.Dice = [2]int{6, 4}
	bareID, err := st.Positions().Save(ctx, testTenant, &barePos)
	if err != nil {
		t.Fatal(err)
	}

	matchID, err := st.Matches().Save(ctx, testTenant, &domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 5})
	if err != nil {
		t.Fatal(err)
	}
	gameID, err := st.Matches().CreateGame(ctx, testTenant, &domain.Game{MatchID: matchID, GameNumber: 1})
	if err != nil {
		t.Fatal(err)
	}
	for i, mv := range []domain.Move{
		{MoveType: "checker", PositionID: checkerID, Player: 1, CheckerMove: "24/21 6/5"},
		{MoveType: "cube", PositionID: cubeID, Player: -1, CubeAction: "Double"},
		{MoveType: "cube", PositionID: cubeID, Player: 1, CubeAction: "Pass"},
		{MoveType: "checker", PositionID: bareID, Player: -1, CheckerMove: "24/14"},
	} {
		mv.GameID, mv.MoveNumber = gameID, int32(i+1)
		if _, err := st.Matches().CreateMove(ctx, testTenant, &mv); err != nil {
			t.Fatal(err)
		}
	}

	want := []map[string]any{
		{"best_move": "8/5 6/5", "error_mp": float64(80)},
		{"best_cube_action": "No Double"},
		{"best_cube_action": "Take"},
		{},
	}
	for _, c := range []struct {
		path string
		body any
	}{
		{"/v1/matches.moves", gameIDReq{GameID: gameID}},
		{"/v1/matches.movesByMatch", matchIDReq{MatchID: matchID}},
	} {
		resp := post(t, ts, c.path, c.body)
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			t.Fatalf("%s: status %d", c.path, resp.StatusCode)
		}
		var rows []map[string]any
		sc := bufio.NewScanner(resp.Body)
		for sc.Scan() {
			if strings.TrimSpace(sc.Text()) == "" {
				continue
			}
			var row map[string]any
			if err := json.Unmarshal(sc.Bytes(), &row); err != nil {
				t.Fatalf("%s: decode %q: %v", c.path, sc.Text(), err)
			}
			rows = append(rows, row)
		}
		resp.Body.Close()
		if len(rows) != len(want) {
			t.Fatalf("%s: %d rows, want %d", c.path, len(rows), len(want))
		}
		for i, w := range want {
			for _, key := range []string{"best_move", "best_cube_action"} {
				got, has := rows[i][key]
				exp, wanted := w[key]
				if has != wanted || got != exp {
					t.Errorf("%s: row %d: %s = %v (present %v), want %v (present %v)", c.path, i, key, got, has, exp, wanted)
				}
			}
			if exp, ok := w["error_mp"]; ok && rows[i]["error_mp"] != exp {
				t.Errorf("%s: row %d: error_mp = %v, want %v", c.path, i, rows[i]["error_mp"], exp)
			}
		}
	}
}

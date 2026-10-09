package sqlshared_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// The cube a decision's MWC loss is converted at: a take or a pass on the
// turned cube held by no one steps down to the cube before the double; a
// take an .xg file left on the doubler's own row, an owned cube, already
// stands at that cube and keeps it; a double keeps its own cube.
func TestCubeMultiplierOfAnAnswer(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "answers.db")
	s, err := sqlite.Open(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	db, err := sql.Open("sqlite", sqlite.DSN(path))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	mid, err := s.Matches().Save(ctx, "", &domain.Match{Player1Name: "A", Player2Name: "B", MatchLength: 7})
	if err != nil {
		t.Fatal(err)
	}
	gid, err := s.Matches().CreateGame(ctx, "", &domain.Game{MatchID: mid, GameNumber: 1})
	if err != nil {
		t.Fatal(err)
	}
	for i, c := range []struct {
		action string
		cube   domain.Cube
		want   int
	}{
		{"Take", domain.Cube{Value: 2, Owner: domain.None}, 2},
		{"Pass", domain.Cube{Value: 1, Owner: domain.None}, 1},
		{"Take", domain.Cube{Value: 2, Owner: domain.Black}, 4},
		{"Double", domain.Cube{Value: 2, Owner: domain.Black}, 4},
	} {
		p := domain.InitializePosition()
		p.DecisionType = domain.CubeAction
		p.Board.Points[7+i] = domain.Point{Checkers: 1, Color: domain.Black}
		p.Cube = c.cube
		pid, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatal(err)
		}
		mvID, err := s.Matches().CreateMove(ctx, "", &domain.Move{GameID: gid, MoveNumber: int32(i + 1), MoveType: "cube", PositionID: pid, Player: -1, CubeAction: c.action})
		if err != nil {
			t.Fatal(err)
		}
		var got int
		if err := db.QueryRowContext(ctx, `SELECT `+sqlshared.CubeMultiplierExpr+` FROM move mv JOIN position p ON p.id = mv.position_id WHERE mv.id = ?`, mvID).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != c.want {
			t.Errorf("%s on cube %+v converted at %d, want %d", c.action, c.cube, got, c.want)
		}
	}
}

package sqlite

import (
	"context"
	"database/sql"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// countingExecer counts the statements a transaction sends, by text.
type countingExecer struct {
	execer
	sent map[string]int
}

func (c *countingExecer) ExecContext(ctx context.Context, q string, args ...any) (sql.Result, error) {
	c.sent[q]++
	return c.execer.ExecContext(ctx, q, args...)
}

func (c *countingExecer) QueryContext(ctx context.Context, q string, args ...any) (*sql.Rows, error) {
	c.sent[q]++
	return c.execer.QueryContext(ctx, q, args...)
}

func (c *countingExecer) QueryRowContext(ctx context.Context, q string, args ...any) *sql.Row {
	c.sent[q]++
	return c.execer.QueryRowContext(ctx, q, args...)
}

func (c *countingExecer) total() int {
	n := 0
	for _, v := range c.sent {
		n += v
	}
	return n
}

// TestImportPathStatementsArePreparedOnceAndSkipTheMoveTable walks the
// statements one imported decision costs: position insert, analysis read,
// analysis upsert, move insert. With the played actions known from the match
// graph, the move table is never read, and each hot statement is prepared once
// per transaction however many decisions it carries.
func TestImportPathStatementsArePreparedOnceAndSkipTheMoveTable(t *testing.T) {
	ctx := context.Background()
	st, err := Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	sqlTx, err := st.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer sqlTx.Rollback()
	stx := &stmtTx{Tx: sqlTx}
	count := &countingExecer{execer: stx, sent: map[string]int{}}
	tx := &txImpl{binder: binder{db: count}, tx: sqlTx}

	matchID, err := tx.Matches().Save(ctx, "", &domain.Match{Player1Name: "A", Player2Name: "B", MatchLength: 5})
	if err != nil {
		t.Fatal(err)
	}
	gameID, err := tx.Matches().CreateGame(ctx, "", &domain.Game{MatchID: matchID, GameNumber: 1})
	if err != nil {
		t.Fatal(err)
	}

	const decisions = 10
	before := count.total()
	for i := 0; i < decisions; i++ {
		p := domain.InitializePosition()
		p.DecisionType = domain.CheckerAction
		p.Score = [2]int{i, 0}
		posID, err := tx.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatal(err)
		}
		// A checker analysis names its played move, never a cube action: the
		// store needs played to complete it.
		played := &storage.PlayedActions{CheckerMove: "13/11 24/23", Position: &p}
		if _, err := tx.Analyses().Merge(ctx, "", posID, played, func(*domain.PositionAnalysis) *domain.PositionAnalysis {
			return &domain.PositionAnalysis{
				AnalysisType:    "CheckerMove",
				PlayedMoves:     []string{"13/11 24/23"},
				CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{{Move: "13/11 24/23", Equity: 0.1}}},
			}
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Matches().CreateMove(ctx, "", &domain.Move{GameID: gameID, MoveNumber: int32(i), PositionID: posID, CheckerMove: "13/11 24/23"}); err != nil {
			t.Fatal(err)
		}
	}

	if n := count.sent[playedActionsSQL]; n != 0 {
		t.Errorf("the move table was read %d times although the graph knew every played action", n)
	}
	if per := (count.total() - before) / decisions; per != 6 {
		t.Errorf("%d statements per decision, want 6 (position, analysis read, match stats invalidation, analysis upsert, move, position match date): %v", per, count.sent)
	}
	for _, q := range []string{positionInsertSQL, analysisMergeSelectSQL, invalidateMatchStatsOfPositionSQL, analysisUpsertSQL, moveInsertSQL, positionMatchDateOnMoveSQL} {
		if _, ok := stx.stmts[q]; !ok {
			t.Errorf("hot statement not prepared: %.40q", q)
		}
	}
	for q := range stx.stmts {
		if !hotStatements[q] {
			t.Errorf("a statement outside hotStatements was cached: %.40q", q)
		}
	}
}

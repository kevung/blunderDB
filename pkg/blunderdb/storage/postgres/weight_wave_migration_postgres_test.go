//go:build postgres

package postgres_test

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	pg "github.com/kevung/blunderdb/pkg/blunderdb/storage/postgres"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// TestMigrate_034_WeightWaveOnAPopulatedLibrary fills a library, rolls its
// position.state and action-label columns back to the text 033 left them in
// (the compact JSON board, the labels verbatim, no action_label table), and
// runs 034 again: every board and every label — a NULL, "" and labels the
// fixed list lacks among them — must read back as it was written.
func TestMigrate_034_WeightWaveOnAPopulatedLibrary(t *testing.T) {
	ctx := context.Background()
	dsn := startPostgres(t)
	s, err := pg.Open(ctx, dsn, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer conn.Close(ctx)

	ms := s.Matches()
	matchID, err := ms.Save(ctx, "", &domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 7})
	if err != nil {
		t.Fatal(err)
	}
	gameID, err := ms.CreateGame(ctx, "", &domain.Game{MatchID: matchID, GameNumber: 1})
	if err != nil {
		t.Fatal(err)
	}
	labels := append(slices.Clone(domain.ActionLabels()), "Unknown(-1)", "Doppel, Annahme")
	boards := map[int64]domain.Board{}
	for i, label := range labels {
		p := domain.InitializePosition()
		p.Score = [2]int{i % 7, i / 7}
		p.Board.Points[1+i%20].Checkers = i%3 + 1
		p.Board.Points[1+i%20].Color = i % 2
		pid, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("Save position %d: %v", i, err)
		}
		saved, err := s.Positions().Load(ctx, "", pid)
		if err != nil {
			t.Fatal(err)
		}
		boards[pid] = saved.Board
		mv := domain.Move{GameID: gameID, MoveNumber: int32(i + 1), MoveType: labels[len(labels)-1-i],
			PositionID: pid, Player: 1, CubeAction: label}
		if _, err := ms.CreateMove(ctx, "", &mv); err != nil {
			t.Fatalf("CreateMove(%q): %v", label, err)
		}
		a := domain.PositionAnalysis{AnalysisType: "DoublingCube",
			DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{AnalysisDepth: "4-ply", BestCubeAction: label}}
		if err := s.Analyses().Save(ctx, "", pid, &a); err != nil {
			t.Fatalf("Save analysis %d: %v", i, err)
		}
	}
	// A NULL label, which no write path produces but a 033 library may hold.
	if _, err := conn.Exec(ctx, `UPDATE move SET cube_action = NULL WHERE move_number = 1`); err != nil {
		t.Fatal(err)
	}

	columns := weightWaveColumns
	readLabels := func(read func(string) string) map[string]string {
		t.Helper()
		out := map[string]string{}
		for _, c := range columns {
			rows, err := conn.Query(ctx, `SELECT id, `+read(c.column)+` FROM `+c.table)
			if err != nil {
				t.Fatal(err)
			}
			for rows.Next() {
				var id int64
				var v *string
				if err := rows.Scan(&id, &v); err != nil {
					t.Fatal(err)
				}
				label := "<NULL>"
				if v != nil {
					label = "=" + *v
				}
				out[fmt.Sprintf("%s.%s:%d", c.table, c.column, id)] = label
			}
			if err := rows.Err(); err != nil {
				t.Fatal(err)
			}
			rows.Close()
		}
		return out
	}
	want := readLabels(sqlshared.ActionLabelSQL)
	if len(want) != 3*len(labels) {
		t.Fatalf("read %d labels, want %d", len(want), 3*len(labels))
	}

	rollback := weightWaveRollback()
	for _, stmt := range rollback {
		if _, err := conn.Exec(ctx, stmt); err != nil {
			t.Fatalf("roll back to 033 (%s): %v", stmt, err)
		}
	}
	if got := readLabels(func(col string) string { return col }); !maps.Equal(got, want) {
		t.Fatalf("the rolled-back text labels differ from the coded ones")
	}

	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	for _, c := range columns {
		var typ string
		if err := conn.QueryRow(ctx, `SELECT data_type FROM information_schema.columns
			WHERE table_schema = current_schema() AND table_name = $1 AND column_name = $2`, c.table, c.column).Scan(&typ); err != nil || typ != "integer" {
			t.Errorf("%s.%s is %q after 034, %v; want integer", c.table, c.column, typ, err)
		}
	}
	if got := readLabels(sqlshared.ActionLabelSQL); !maps.Equal(got, want) {
		t.Errorf("labels after 034 differ from the labels before:\n got %v\nwant %v", got, want)
	}
	for pid, b := range boards {
		p, err := s.Positions().Load(ctx, "", pid)
		if err != nil {
			t.Fatal(err)
		}
		if p.Board != b {
			t.Errorf("position %d: board after 034 differs from the board before", pid)
		}
	}
	got, err := ms.MovesByPositions(ctx, "", slices.Collect(maps.Keys(boards)))
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, mvs := range got {
		n += len(mvs)
	}
	if n != len(labels) {
		t.Errorf("read %d moves through the store after 034, want %d", n, len(labels))
	}
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
}

type labelColumn struct{ table, column string }

// weightWaveColumns are the action-label columns 034 turns into codes.
var weightWaveColumns = []labelColumn{{"analysis", "best_cube_action"}, {"move", "move_type"}, {"move", "cube_action"}}

// weightWaveRollback returns the statements that put a 2.31.0 library back in
// the shape 033 left: text labels, the compact JSON board, no action_label
// table and 034 unrecorded.
func weightWaveRollback() []string {
	var rollback []string
	for _, c := range weightWaveColumns {
		rollback = append(rollback,
			`ALTER TABLE `+c.table+` ADD COLUMN `+c.column+`_text TEXT`,
			`UPDATE `+c.table+` SET `+c.column+`_text = `+sqlshared.ActionLabelSQL(c.column),
			`ALTER TABLE `+c.table+` DROP COLUMN `+c.column,
			`ALTER TABLE `+c.table+` RENAME COLUMN `+c.column+`_text TO `+c.column)
	}
	rollback = append(rollback,
		`DROP TABLE action_label`,
		`ALTER TABLE position ADD COLUMN state_text TEXT`,
		`UPDATE position SET state_text = '[' || (SELECT string_agg((CASE WHEN get_byte(state, i) > 127
		     THEN get_byte(state, i) - 256 ELSE get_byte(state, i) END)::text, ',' ORDER BY i)
		   FROM generate_series(0, 27) AS i) || ']'`,
		`ALTER TABLE position DROP COLUMN state`,
		`ALTER TABLE position RENAME COLUMN state_text TO state`,
		`ALTER TABLE position ALTER COLUMN state SET NOT NULL`,
		`DELETE FROM schema_migrations WHERE version = '034_weight_wave'`)
	return rollback
}

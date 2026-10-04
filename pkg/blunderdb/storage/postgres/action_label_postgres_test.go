//go:build postgres

package postgres_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	pg "github.com/kevung/blunderdb/pkg/blunderdb/storage/postgres"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// TestActionLabelIsTenantScoped writes the same off-list label and one label
// of each tenant's own: every tenant gets its own row and code, reads back
// only what it wrote, and purging one tenant leaves the other's labels.
func TestActionLabelIsTenantScoped(t *testing.T) {
	ctx := context.Background()
	dsn := startPostgres(t)
	s, err := pg.Open(ctx, dsn, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer conn.Close(ctx)

	write := func(scope string, labels ...string) {
		t.Helper()
		ms := s.Matches()
		matchID, err := ms.Save(ctx, scope, &domain.Match{Player1Name: "A" + scope, Player2Name: "B", MatchLength: 5})
		if err != nil {
			t.Fatal(err)
		}
		gameID, err := ms.CreateGame(ctx, scope, &domain.Game{MatchID: matchID, GameNumber: 1})
		if err != nil {
			t.Fatal(err)
		}
		for i, l := range labels {
			mv := domain.Move{GameID: gameID, MoveNumber: int32(i + 1), Player: 1, MoveType: "cube", CubeAction: l}
			if _, err := ms.CreateMove(ctx, scope, &mv); err != nil {
				t.Fatalf("CreateMove(%s, %q): %v", scope, l, err)
			}
		}
	}
	write("11", "Doppel, Annahme", "Secret of tenant 11")
	write("22", "Doppel, Annahme")

	read := func(tenant int64) map[string]bool {
		t.Helper()
		rows, err := conn.Query(ctx, `SELECT `+sqlshared.ActionLabelSQL("cube_action")+` FROM move WHERE tenant_id = $1`, tenant)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		out := map[string]bool{}
		for rows.Next() {
			var l string
			if err := rows.Scan(&l); err != nil {
				t.Fatal(err)
			}
			out[l] = true
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		return out
	}
	labelRows := func(tenant int64) (n int) {
		t.Helper()
		if err := conn.QueryRow(ctx, `SELECT count(*) FROM action_label WHERE tenant_id = $1`, tenant).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	if got := read(22); len(got) != 1 || !got["Doppel, Annahme"] {
		t.Errorf("tenant 22 reads %v, want only its own label", got)
	}
	if got := read(11); len(got) != 2 || !got["Secret of tenant 11"] {
		t.Errorf("tenant 11 reads %v, want its two labels", got)
	}
	if n := labelRows(22); n != 1 {
		t.Errorf("tenant 22 holds %d action_label rows, want 1", n)
	}

	if err := s.PurgeTenant(ctx, "11"); err != nil {
		t.Fatalf("PurgeTenant: %v", err)
	}
	if n := labelRows(11); n != 0 {
		t.Errorf("purged tenant 11 still holds %d action_label rows", n)
	}
	if got := read(22); !got["Doppel, Annahme"] {
		t.Errorf("after purging tenant 11, tenant 22 reads %v", got)
	}
}

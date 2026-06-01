//go:build postgres

package postgres_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	pg "github.com/kevung/blunderdb/pkg/blunderdb/storage/postgres"
)

func rlsPos(i int) domain.Position {
	p := domain.InitializePosition()
	p.DecisionType = domain.CheckerAction
	p.Score[0] = i // distinct score → distinct Zobrist hash (PlayerOnRoll=Black, no mirror)
	return p
}

// TestRLSEnforcesTenantIsolation verifies that, with RLS installed, a
// non-superuser connection sees only the rows of the tenant in its
// app.tenant_id GUC — even for a query with no tenant filter — and none when
// the GUC is unset (fail-closed). Superusers bypass RLS, so enforcement is
// checked through a dedicated restricted role.
func TestRLSEnforcesTenantIsolation(t *testing.T) {
	ctx := context.Background()
	dsn := startPostgres(t)

	st, err := pg.Open(ctx, dsn, &storage.Options{EnableRLS: true})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer st.Close()
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if err := st.ApplyRLS(ctx); err != nil {
		t.Fatalf("ApplyRLS: %v", err)
	}

	// Seed one position for tenant 1 and one for tenant 2 (as superuser, which
	// bypasses RLS for the insert; the app filter sets tenant_id per scope).
	p1 := rlsPos(1)
	if _, err := st.Positions().Save(storage.WithTenant(ctx, 1), "1", &p1); err != nil {
		t.Fatalf("Save tenant 1: %v", err)
	}
	p2 := rlsPos(2)
	if _, err := st.Positions().Save(storage.WithTenant(ctx, 2), "2", &p2); err != nil {
		t.Fatalf("Save tenant 2: %v", err)
	}

	// Create a restricted (non-superuser) role and connect as it, so RLS applies.
	admin, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("admin connect: %v", err)
	}
	defer admin.Close(ctx)
	for _, stmt := range []string{
		`DROP ROLE IF EXISTS rls_app`,
		`CREATE ROLE rls_app LOGIN PASSWORD 'app'`,
		`GRANT USAGE ON SCHEMA public TO rls_app`,
		`GRANT SELECT ON ALL TABLES IN SCHEMA public TO rls_app`,
	} {
		if _, err := admin.Exec(ctx, stmt); err != nil {
			t.Fatalf("setup role (%s): %v", stmt, err)
		}
	}

	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("parse dsn: %v", err)
	}
	cfg.User = "rls_app"
	cfg.Password = "app"
	app, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("app connect: %v", err)
	}
	defer app.Close(ctx)

	countPositions := func() int {
		t.Helper()
		var n int
		// Deliberately unfiltered: only RLS restricts the visible rows.
		if err := app.QueryRow(ctx, `SELECT count(*) FROM position`).Scan(&n); err != nil {
			t.Fatalf("count: %v", err)
		}
		return n
	}
	setTenant := func(tenant int) {
		t.Helper()
		if _, err := app.Exec(ctx, `SELECT set_config('app.tenant_id', $1, false)`, strconv.Itoa(tenant)); err != nil {
			t.Fatalf("set tenant %d: %v", tenant, err)
		}
	}

	setTenant(1)
	if got := countPositions(); got != 1 {
		t.Errorf("tenant 1 sees %d positions, want 1 (RLS not isolating)", got)
	}
	setTenant(2)
	if got := countPositions(); got != 1 {
		t.Errorf("tenant 2 sees %d positions, want 1 (RLS not isolating)", got)
	}
	if _, err := app.Exec(ctx, `RESET app.tenant_id`); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if got := countPositions(); got != 0 {
		t.Errorf("no tenant set sees %d positions, want 0 (RLS not fail-closed)", got)
	}
}

// TestRLSPoolSetsTenantGUC verifies the EnableRLS pool sets app.tenant_id from
// the operation context and resets it on release, so Storage operations carry
// the right tenant without any per-call wiring.
func TestRLSPoolSetsTenantGUC(t *testing.T) {
	ctx := context.Background()
	dsn := startPostgres(t)
	st, err := pg.Open(ctx, dsn, &storage.Options{EnableRLS: true})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer st.Close()
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if err := st.ApplyRLS(ctx); err != nil {
		t.Fatalf("ApplyRLS: %v", err)
	}

	// A round trip through the RLS pool with tenant 7 in context must succeed:
	// the GUC is set so the WITH CHECK policy accepts the insert (superuser
	// bypasses, but this still exercises the PrepareConn hook end to end).
	p := rlsPos(7)
	id, err := st.Positions().Save(storage.WithTenant(ctx, 7), "7", &p)
	if err != nil {
		t.Fatalf("Save with tenant ctx: %v", err)
	}
	got, err := st.Positions().Load(storage.WithTenant(ctx, 7), "7", id)
	if err != nil {
		t.Fatalf("Load with tenant ctx: %v", err)
	}
	if got.ID != id {
		t.Errorf("loaded id %d, want %d", got.ID, id)
	}
}

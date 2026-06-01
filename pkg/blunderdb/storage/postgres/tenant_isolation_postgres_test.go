//go:build postgres

package postgres_test

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	pg "github.com/kevung/blunderdb/pkg/blunderdb/storage/postgres"
)

// TestTenantIsolation verifies the central multi-tenant promise: rows written
// under one tenant scope are invisible to another, and per-tenant uniqueness
// (position Zobrist, filter name) does not collide across tenants.
func TestTenantIsolation(t *testing.T) {
	ctx := context.Background()
	dsn := startPostgres(t)
	resetPublicSchema(t, dsn)
	s, err := pg.Open(ctx, dsn, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	countPositions := func(scope string) int {
		n := 0
		for _, err := range s.Positions().List(ctx, scope, storage.ListOpts{}) {
			if err != nil {
				t.Fatalf("List(%s): %v", scope, err)
			}
			n++
		}
		return n
	}

	// The same position saved under two tenants yields a distinct row per
	// tenant (the UNIQUE index is (tenant_id, zobrist_hash)).
	p1 := domain.InitializePosition()
	id1, err := s.Positions().Save(ctx, "1", &p1)
	if err != nil {
		t.Fatalf("save t1: %v", err)
	}
	p2 := domain.InitializePosition()
	id2, err := s.Positions().Save(ctx, "2", &p2)
	if err != nil {
		t.Fatalf("save t2: %v", err)
	}
	if id1 == id2 {
		t.Fatalf("tenants 1 and 2 share position id %d (not isolated)", id1)
	}

	// A second position only under tenant 1.
	q := domain.InitializePosition()
	q.Board.Points[1] = domain.Point{Checkers: 1, Color: domain.White}
	q.Board.Points[3] = domain.Point{Checkers: 1, Color: domain.White}
	if _, err := s.Positions().Save(ctx, "1", &q); err != nil {
		t.Fatalf("save t1 second: %v", err)
	}

	if got := countPositions("1"); got != 2 {
		t.Errorf("tenant 1 positions: got %d, want 2", got)
	}
	if got := countPositions("2"); got != 1 {
		t.Errorf("tenant 2 positions: got %d, want 1", got)
	}

	// Tenant 1's extra position is not loadable under tenant 2.
	if _, err := s.Positions().Load(ctx, "2", id1); err == nil {
		t.Error("tenant 2 could load tenant 1's position")
	}

	// Filter names are unique per tenant, not globally.
	if _, err := s.Filters().Save(ctx, "1", "fav", "cmd"); err != nil {
		t.Fatalf("filter t1: %v", err)
	}
	if _, err := s.Filters().Save(ctx, "2", "fav", "cmd"); err != nil {
		t.Fatalf("filter t2 (same name, different tenant should be allowed): %v", err)
	}

	// Counts are per tenant.
	c1, err := s.Metadata().Counts(ctx, "1")
	if err != nil {
		t.Fatal(err)
	}
	c2, _ := s.Metadata().Counts(ctx, "2")
	if c1.Positions != 2 || c2.Positions != 1 {
		t.Fatalf("per-tenant counts wrong: t1=%d t2=%d, want 2 and 1", c1.Positions, c2.Positions)
	}
}

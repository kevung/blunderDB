// Tenant isolation cases: one family at a time, tenant A writes a row and
// tenant B must see none of it — no List entry, no Get/Load by A's id.
//
// This table lives here (backend-agnostic, like the rest of storagetest) but
// is deliberately NOT part of RunContractTests: SQLite has no tenants (see
// storage.go's package doc — the desktop/CLI pass the single implicit tenant
// "", and `scope` is otherwise unused by that backend), so asserting cross-
// tenant invisibility against it would either be vacuous or fail by
// construction. RunTenantIsolationTests is for a real multi-tenant backend
// (PostgreSQL), called from its own build-tag-gated test
// (tenant_isolation_postgres_test.go) — see #235: before this, isolation was
// checked for 3 of storage's ~16 tenant-scoped families, each in its own
// hand-rolled test; this file gives every family the same loop.
package storagetest

import (
	"context"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// tenantIsolationCase is one family's isolation check.
type tenantIsolationCase struct {
	name string
	fn   func(t *testing.T, ctx context.Context, s storage.Storage, a, b string)
}

var tenantIsolationCases = []tenantIsolationCase{
	{"Position", checkPositionIsolation},
	{"Analysis", checkAnalysisIsolation},
	{"Collection", checkCollectionIsolation},
	{"Tournament", checkTournamentIsolation},
	{"Filter", checkFilterIsolation},
	{"Match", checkMatchIsolation},
	{"Comment", checkCommentIsolation},
	{"Anki/Deck", checkAnkiDeckIsolation},
}

// RunTenantIsolationTests runs every family's isolation check against a
// fresh Storage from factory (called once per case, like RunContractTests).
// a and b are two distinct tenant scopes.
func RunTenantIsolationTests(t *testing.T, factory func() storage.Storage, a, b string) {
	t.Helper()
	ctx := context.Background()
	for _, tc := range tenantIsolationCases {
		t.Run(tc.name, func(t *testing.T) {
			s := factory()
			defer s.Close()
			tc.fn(t, ctx, s, a, b)
		})
	}
}

func checkPositionIsolation(t *testing.T, ctx context.Context, s storage.Storage, a, b string) {
	p := checkerPos()
	id, err := s.Positions().Save(ctx, a, &p)
	if err != nil {
		t.Fatalf("Save(%s): %v", a, err)
	}

	n := 0
	for _, err := range s.Positions().List(ctx, b, storage.ListOpts{}) {
		if err != nil {
			t.Fatalf("List(%s): %v", b, err)
		}
		n++
	}
	if n != 0 {
		t.Errorf("tenant %s sees %d position(s) belonging to tenant %s, want 0", b, n, a)
	}
	if _, err := s.Positions().Load(ctx, b, id); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Load(%s, id from %s): got %v, want ErrNotFound", b, a, err)
	}
}

func checkAnalysisIsolation(t *testing.T, ctx context.Context, s storage.Storage, a, b string) {
	pa := checkerPos()
	idA, err := s.Positions().Save(ctx, a, &pa)
	if err != nil {
		t.Fatalf("Save position(%s): %v", a, err)
	}
	if err := s.Analyses().Save(ctx, a, idA, &domain.PositionAnalysis{}); err != nil {
		t.Fatalf("Save analysis(%s): %v", a, err)
	}

	// idA cannot collide with any id tenant b already holds (ids are
	// assigned from one global sequence, not per tenant), so this alone
	// proves b cannot read a's analysis by a's id.
	if _, err := s.Analyses().Load(ctx, b, idA); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Load(%s, id from %s): got %v, want ErrNotFound", b, a, err)
	}
}

func checkCollectionIsolation(t *testing.T, ctx context.Context, s storage.Storage, a, b string) {
	cid, err := s.Collections().Create(ctx, a, "priv-coll", "")
	if err != nil {
		t.Fatalf("Create(%s): %v", a, err)
	}

	n := 0
	for _, err := range s.Collections().List(ctx, b) {
		if err != nil {
			t.Fatalf("List(%s): %v", b, err)
		}
		n++
	}
	if n != 0 {
		t.Errorf("tenant %s sees %d collection(s) belonging to tenant %s, want 0", b, n, a)
	}
	if _, err := s.Collections().Get(ctx, b, cid); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Get(%s, id from %s): got %v, want ErrNotFound", b, a, err)
	}
}

func checkTournamentIsolation(t *testing.T, ctx context.Context, s storage.Storage, a, b string) {
	id, err := s.Tournaments().Create(ctx, a, "priv-tourney", "2026-01-01", "")
	if err != nil {
		t.Fatalf("Create(%s): %v", a, err)
	}

	n := 0
	for _, err := range s.Tournaments().List(ctx, b) {
		if err != nil {
			t.Fatalf("List(%s): %v", b, err)
		}
		n++
	}
	if n != 0 {
		t.Errorf("tenant %s sees %d tournament(s) belonging to tenant %s, want 0", b, n, a)
	}
	if _, err := s.Tournaments().Get(ctx, b, id); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Get(%s, id from %s): got %v, want ErrNotFound", b, a, err)
	}
}

func checkFilterIsolation(t *testing.T, ctx context.Context, s storage.Storage, a, b string) {
	if _, err := s.Filters().Save(ctx, a, "fav", "cmd"); err != nil {
		t.Fatalf("Save(%s): %v", a, err)
	}

	n := 0
	for _, err := range s.Filters().List(ctx, b) {
		if err != nil {
			t.Fatalf("List(%s): %v", b, err)
		}
		n++
	}
	if n != 0 {
		t.Errorf("tenant %s sees %d filter(s) belonging to tenant %s, want 0", b, n, a)
	}
	// The same name is free to reuse under a different tenant: it collides
	// only within one tenant's own filter library.
	if _, err := s.Filters().Save(ctx, b, "fav", "cmd"); err != nil {
		t.Errorf("Save(%s) with a's filter name: %v, want no collision across tenants", b, err)
	}
}

func checkMatchIsolation(t *testing.T, ctx context.Context, s storage.Storage, a, b string) {
	m := domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 7}
	id, err := s.Matches().Save(ctx, a, &m)
	if err != nil {
		t.Fatalf("Save(%s): %v", a, err)
	}
	if _, err := s.Matches().Get(ctx, b, id); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Get(%s, id from %s): got %v, want ErrNotFound", b, a, err)
	}
}

func checkCommentIsolation(t *testing.T, ctx context.Context, s storage.Storage, a, b string) {
	p := checkerPos()
	posID, err := s.Positions().Save(ctx, a, &p)
	if err != nil {
		t.Fatalf("Save position(%s): %v", a, err)
	}
	if _, err := s.Comments().Add(ctx, a, posID, "private note"); err != nil {
		t.Fatalf("Add comment(%s): %v", a, err)
	}

	// posID belongs to tenant a alone (global id sequence — see
	// checkAnalysisIsolation), so tenant b addressing it directly must read
	// no comment at all.
	got, err := s.Comments().Text(ctx, b, posID)
	if err != nil {
		t.Fatalf("Text(%s, position from %s): %v", b, a, err)
	}
	if got != "" {
		t.Errorf("tenant %s reads tenant %s's comment via its position id: %q", b, a, got)
	}
}

func checkAnkiDeckIsolation(t *testing.T, ctx context.Context, s storage.Storage, a, b string) {
	if _, err := s.Anki().CreateDeck(ctx, a, "priv-deck", "", "collection", 0, ""); err != nil {
		t.Fatalf("CreateDeck(%s): %v", a, err)
	}

	n := 0
	for _, err := range s.Anki().ListDecks(ctx, b) {
		if err != nil {
			t.Fatalf("ListDecks(%s): %v", b, err)
		}
		n++
	}
	if n != 0 {
		t.Errorf("tenant %s sees %d deck(s) belonging to tenant %s, want 0", b, n, a)
	}
}

//go:build postgres

package postgres_test

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	pg "github.com/kevung/blunderdb/pkg/blunderdb/storage/postgres"
)

func tctx(ctx context.Context, scope string) context.Context {
	n, _ := storage.ParseTenant(scope)
	return storage.WithTenant(ctx, n)
}

// openRLS applies RLS to the library openAsRLSOwner built and opens it again
// the way `serve --rls` does: every statement runs under its tenant's policy.
func openRLS(t *testing.T) (*pg.Storage, string) {
	t.Helper()
	ctx := context.Background()
	owner, _, dsn := openAsRLSOwner(t)
	if err := owner.ApplyRLS(ctx); err != nil {
		t.Fatalf("ApplyRLS: %v", err)
	}
	s, err := pg.Open(ctx, dsn, &storage.Options{EnableRLS: true})
	if err != nil {
		t.Fatalf("Open with RLS: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s, dsn
}

// TestMigrate_055_ChangeLogUnderRLS runs the journal migration on a role that
// is neither superuser nor BYPASSRLS, with FORCE on: the triggers write under
// the tenant policy, each tenant sees only its own changes, and a second
// Migrate adds nothing.
func TestMigrate_055_ChangeLogUnderRLS(t *testing.T) {
	ctx := context.Background()
	s, _ := openRLS(t)
	j := s.ChangeJournal()

	for scope, name := range map[string]string{"1": "alpha", "2": "beta"} {
		id, err := s.Collections().Create(tctx(ctx, scope), scope, name, "")
		if err != nil {
			t.Fatalf("tenant %s: %v", scope, err)
		}
		if err := s.Collections().Update(tctx(ctx, scope), scope, id, name+"!", ""); err != nil {
			t.Fatal(err)
		}
	}
	for _, scope := range []string{"1", "2"} {
		cs, err := j.Changes(tctx(ctx, scope), scope, 0, 100)
		if err != nil {
			t.Fatal(err)
		}
		if len(cs) != 1 || cs[0].Family != "collection" || cs[0].Op != storage.ChangeUpdate {
			t.Errorf("tenant %s journal = %+v, want one collection update", scope, cs)
		}
	}
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
	if cs, _ := j.Changes(tctx(ctx, "1"), "1", 0, 100); len(cs) != 1 {
		t.Errorf("after a second Migrate: %+v", cs)
	}
}

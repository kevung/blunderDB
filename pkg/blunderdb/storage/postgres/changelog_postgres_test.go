//go:build postgres

package postgres_test

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// TestMigrate_055_ChangeLogUnderRLS runs the journal migration on a role that
// is neither superuser nor BYPASSRLS, with FORCE on: the triggers write under
// the tenant policy, each tenant sees only its own changes, and a second
// Migrate adds nothing.
func TestMigrate_055_ChangeLogUnderRLS(t *testing.T) {
	ctx := context.Background()
	s, conn, _ := openAsRLSOwner(t)
	j := s.ChangeJournal()

	for scope, name := range map[string]string{"1": "alpha", "2": "beta"} {
		id, err := s.Collections().Create(ctx, scope, name, "")
		if err != nil {
			t.Fatalf("tenant %s: %v", scope, err)
		}
		if err := s.Collections().Update(ctx, scope, id, name+"!", ""); err != nil {
			t.Fatal(err)
		}
	}
	for _, scope := range []string{"1", "2"} {
		cs, err := j.Changes(ctx, scope, 0, 100)
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
	var forced bool
	if err := conn.QueryRow(ctx, `SELECT relforcerowsecurity FROM pg_class WHERE relname = 'change_log'`).Scan(&forced); err != nil || !forced {
		t.Errorf("change_log FORCE RLS = %v, %v", forced, err)
	}
	if cs, _ := j.Changes(ctx, "1", 0, 100); len(cs) != 1 {
		t.Errorf("after a second Migrate: %+v", cs)
	}
}

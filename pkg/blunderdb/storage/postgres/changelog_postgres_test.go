//go:build postgres

package postgres_test

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
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

// TestMigrate_055_FromAnRLSLibrary runs 055 the way an upgrade does: on a
// library whose RLS is already applied and FORCEd, by its owner (neither
// superuser nor BYPASSRLS). The 055 of a fresh library runs before RLS exists
// and a second Migrate replays nothing, so only this rewind proves the file
// holds under FORCE: the journal table comes out under the tenant policy and
// the triggers write through it.
func TestMigrate_055_FromAnRLSLibrary(t *testing.T) {
	ctx := context.Background()
	s, conn, ownerDSN := openAsRLSOwner(t)
	if err := s.ApplyRLS(ctx); err != nil {
		t.Fatalf("ApplyRLS: %v", err)
	}

	var drops []string
	for _, tbl := range []string{"position", "comment", "collection", "match", "transcription", "lesson", "anki_card"} {
		drops = append(drops, `DROP TRIGGER IF EXISTS change_log_`+tbl+` ON `+tbl)
	}
	execUnforced(t, conn, []string{"position", "comment", "collection", "match", "transcription", "lesson", "anki_card", "analysis"},
		append(drops,
			`DROP TRIGGER IF EXISTS change_log_analysis ON analysis`,
			`DROP FUNCTION IF EXISTS blunderdb_journal_analysis()`,
			`DROP FUNCTION IF EXISTS blunderdb_journal()`,
			`DROP TABLE IF EXISTS change_log`,
			`DROP SEQUENCE IF EXISTS change_log_seq`,
			`DELETE FROM schema_migrations WHERE version = '055_change_log'`))

	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("Migrate from 054 under RLS: %v", err)
	}
	assertForced(t, conn, "position", "change_log")

	rls, err := pg.Open(ctx, ownerDSN, &storage.Options{EnableRLS: true})
	if err != nil {
		t.Fatalf("Open with RLS: %v", err)
	}
	t.Cleanup(func() { _ = rls.Close() })
	for _, scope := range []string{"1", "2"} {
		p := domain.InitializePosition()
		pid, err := rls.Positions().Save(tctx(ctx, scope), scope, &p)
		if err != nil {
			t.Fatalf("tenant %s: %v", scope, err)
		}
		a := domain.PositionAnalysis{AnalysisType: "DoublingCube",
			DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{AnalysisDepth: "4-ply", BestCubeAction: "No Double"}}
		if err := rls.Analyses().Save(tctx(ctx, scope), scope, pid, &a); err != nil {
			t.Fatal(err)
		}
	}
	for _, scope := range []string{"1", "2"} {
		cs, err := rls.ChangeJournal().Changes(tctx(ctx, scope), scope, 0, 100)
		if err != nil {
			t.Fatal(err)
		}
		if len(cs) != 1 || cs[0].Family != "position" || cs[0].Op != storage.ChangeUpdate {
			t.Errorf("tenant %s journal = %+v, want its own position, moved by the analysis", scope, cs)
		}
	}
}

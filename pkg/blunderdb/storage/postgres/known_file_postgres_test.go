//go:build postgres

package postgres_test

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	pg "github.com/kevung/blunderdb/pkg/blunderdb/storage/postgres"
)

// TestKnownFileIsTenantScoped journals the same digest for two tenants: each
// knows its own match for it and only that one, and a digest the other tenant
// alone journaled is unknown.
func TestKnownFileIsTenantScoped(t *testing.T) {
	ctx := context.Background()
	s, err := pg.Open(ctx, startPostgres(t), nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	record := func(scope, sha string) int64 {
		t.Helper()
		match, err := s.Matches().Save(ctx, scope, &domain.Match{Player1Name: "A" + scope, Player2Name: "B", MatchLength: 5})
		if err != nil {
			t.Fatal(err)
		}
		batch, err := s.ImportBatches().Begin(ctx, scope, "corpus", "xg")
		if err != nil {
			t.Fatal(err)
		}
		if err := s.ImportBatches().RecordFiles(ctx, scope, batch, []domain.ImportFileEntry{
			{Path: "/c/a.xg", SHA256: sha, Outcome: domain.JournalNew, MatchID: match},
		}); err != nil {
			t.Fatal(err)
		}
		return match
	}
	m11 := record("11", "shared")
	m22 := record("22", "shared")
	record("22", "only22")

	for scope, want := range map[string]int64{"11": m11, "22": m22} {
		got, ok, err := s.ImportBatches().KnownFile(ctx, scope, "shared")
		if err != nil || !ok || got != want {
			t.Errorf("tenant %s: got (%d, %v, %v), want match %d", scope, got, ok, err, want)
		}
	}
	if _, ok, err := s.ImportBatches().KnownFile(ctx, "11", "only22"); err != nil || ok {
		t.Errorf("tenant 11 knows a digest only tenant 22 journaled: (%v, %v)", ok, err)
	}
}

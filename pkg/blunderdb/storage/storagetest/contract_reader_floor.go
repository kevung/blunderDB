package storagetest

import (
	"context"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testMetadataReaderFloor: a fresh database carries the floor this binary
// sets; a floor above the schema this binary speaks refuses Migrate before it
// writes anything; a lower floor is raised, never lowered below a newer one.
func testMetadataReaderFloor(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	md := s.Metadata()
	floor := func() string {
		t.Helper()
		m, err := md.Load(ctx, "")
		if err != nil {
			t.Fatal(err)
		}
		return m[storage.MinReaderVersionKey]
	}

	if got := floor(); got != domain.MinReaderVersion {
		t.Fatalf("fresh database floor = %q, want %q", got, domain.MinReaderVersion)
	}

	if err := md.Save(ctx, "", map[string]string{storage.MinReaderVersionKey: "99.0.0"}); err != nil {
		t.Fatal(err)
	}
	err := s.Migrate(ctx)
	if !errors.Is(err, storage.ErrDatabaseTooNew) {
		t.Fatalf("Migrate under floor 99.0.0: err = %v, want ErrDatabaseTooNew", err)
	}
	if got := floor(); got != "99.0.0" {
		t.Errorf("refused Migrate rewrote the floor to %q", got)
	}
	if v, err := md.Version(ctx, ""); err != nil || v != domain.DatabaseVersion {
		t.Errorf("refused Migrate left version %q (%v), want %q", v, err, domain.DatabaseVersion)
	}

	// Equal to the schema this binary speaks: opens, and is not lowered.
	if err := md.Save(ctx, "", map[string]string{storage.MinReaderVersionKey: domain.DatabaseVersion}); err != nil {
		t.Fatal(err)
	}
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("Migrate under floor %s: %v", domain.DatabaseVersion, err)
	}
	if got := floor(); got != domain.DatabaseVersion {
		t.Errorf("floor = %q after Migrate, want %q kept", got, domain.DatabaseVersion)
	}

	if err := md.Save(ctx, "", map[string]string{storage.MinReaderVersionKey: "2.0.0"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("Migrate under floor 2.0.0: %v", err)
	}
	if got := floor(); got != domain.MinReaderVersion {
		t.Errorf("floor = %q after Migrate, want raised to %q", got, domain.MinReaderVersion)
	}
}

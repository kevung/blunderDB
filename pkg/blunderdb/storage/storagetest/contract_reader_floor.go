package storagetest

import (
	"context"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// testMetadataReaderFloor: a fresh database carries the floor this binary
// sets, and MetadataStore.Save refuses the schema keys — the metadata table
// is global, so a tenant that set them would lock every other one out.
func testMetadataReaderFloor(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	md := s.Metadata()
	m, err := md.Load(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if got := m[storage.MinReaderVersionKey]; got != domain.MinReaderVersion {
		t.Fatalf("fresh database floor = %q, want %q", got, domain.MinReaderVersion)
	}
	for _, key := range []string{storage.MinReaderVersionKey, "database_version"} {
		err := md.Save(ctx, "", map[string]string{"description": "x", key: "99.0.0"})
		if !errors.Is(err, storage.ErrInvalid) {
			t.Errorf("Save(%s): err = %v, want ErrInvalid", key, err)
		}
	}
	after, err := md.Load(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if after[storage.MinReaderVersionKey] != domain.MinReaderVersion || after["database_version"] != domain.DatabaseVersion ||
		after["description"] != m["description"] {
		t.Errorf("a refused Save wrote: %v", after)
	}
}

// RunReaderFloor holds Migrate to the floor: above the schema this binary
// speaks it refuses before writing anything; at or below it the floor is
// raised to domain.MinReaderVersion, never lowered. exec writes the floor
// directly, as a newer binary would.
func RunReaderFloor(t *testing.T, s storage.Storage, exec sqlshared.Execer) {
	t.Helper()
	ctx := context.Background()
	setFloor := func(v string) {
		t.Helper()
		if _, err := exec.Exec(ctx, `UPDATE metadata SET value = ? WHERE key = ?`, v, storage.MinReaderVersionKey); err != nil {
			t.Fatal(err)
		}
	}
	floor := func() string {
		t.Helper()
		var v string
		if err := exec.QueryRow(ctx, `SELECT value FROM metadata WHERE key = ?`, storage.MinReaderVersionKey).Scan(&v); err != nil {
			t.Fatal(err)
		}
		return v
	}
	defer setFloor(domain.MinReaderVersion)

	setFloor("99.0.0")
	if err := s.Migrate(ctx); !errors.Is(err, storage.ErrDatabaseTooNew) {
		t.Fatalf("Migrate under floor 99.0.0: err = %v, want ErrDatabaseTooNew", err)
	}
	if got := floor(); got != "99.0.0" {
		t.Errorf("refused Migrate rewrote the floor to %q", got)
	}
	if v, err := s.Metadata().Version(ctx, ""); err != nil || v != domain.DatabaseVersion {
		t.Errorf("refused Migrate left version %q (%v), want %q", v, err, domain.DatabaseVersion)
	}

	setFloor(domain.DatabaseVersion)
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("Migrate under floor %s: %v", domain.DatabaseVersion, err)
	}
	if got := floor(); got != domain.DatabaseVersion {
		t.Errorf("floor = %q after Migrate, want %q kept", got, domain.DatabaseVersion)
	}

	setFloor("2.0.0")
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("Migrate under floor 2.0.0: %v", err)
	}
	if got := floor(); got != domain.MinReaderVersion {
		t.Errorf("floor = %q after Migrate, want raised to %q", got, domain.MinReaderVersion)
	}
}

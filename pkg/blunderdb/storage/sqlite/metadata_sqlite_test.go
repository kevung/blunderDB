package sqlite_test

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// TestMetadataVersion covers the bootstrapped version, SetVersion and the
// Storage.Version delegation (D6).
func TestMetadataVersion(t *testing.T) {
	ctx := context.Background()
	s := openMem(t)

	v, err := s.Metadata().Version(ctx, "")
	if err != nil {
		t.Fatalf("Version: %v", err)
	}
	if v != domain.DatabaseVersion {
		t.Errorf("bootstrapped version: got %q, want %q", v, domain.DatabaseVersion)
	}

	// Storage.Version delegates to MetadataStore.Version (D6).
	sv, err := s.Version(ctx)
	if err != nil {
		t.Fatalf("Storage.Version: %v", err)
	}
	if sv != v {
		t.Errorf("Storage.Version: got %q, want %q", sv, v)
	}

	if err := s.Metadata().SetVersion(ctx, "", "9.9.9"); err != nil {
		t.Fatalf("SetVersion: %v", err)
	}
	if sv, _ := s.Version(ctx); sv != "9.9.9" {
		t.Errorf("Version after SetVersion: got %q, want 9.9.9", sv)
	}
}

// TestMetadataSaveLoad covers Save and Load of arbitrary key/value pairs.
func TestMetadataSaveLoad(t *testing.T) {
	ctx := context.Background()
	s := openMem(t)

	if err := s.Metadata().Save(ctx, "", map[string]string{
		"user":        "Kévin",
		"description": "test database",
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	md, err := s.Metadata().Load(ctx, "")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if md["user"] != "Kévin" || md["description"] != "test database" {
		t.Errorf("Load: %+v", md)
	}
	// Load returns every pair, including the bootstrapped version row.
	if md["database_version"] != domain.DatabaseVersion {
		t.Errorf("Load missing database_version: %+v", md)
	}
}

// TestMetadataCounts covers the headline row counts.
func TestMetadataCounts(t *testing.T) {
	ctx := context.Background()
	s := openMem(t)

	c, err := s.Metadata().Counts(ctx, "")
	if err != nil {
		t.Fatalf("Counts: %v", err)
	}
	if (c != storage.Counts{}) {
		t.Errorf("fresh database counts: got %+v, want all zero", c)
	}

	savePos(t, s, domain.CheckerAction)
	savePos(t, s, domain.CubeAction)
	c, _ = s.Metadata().Counts(ctx, "")
	if c.Positions != 2 {
		t.Errorf("Counts.Positions: got %d, want 2", c.Positions)
	}
}

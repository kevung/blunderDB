package ingest

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// TestDBImporterRefusesMissingSource: importing a native .db whose path does
// not exist used to create that file — sqlite.Open bootstraps a fresh
// database — and then import nothing from it. The importer must refuse
// before anything touches the disk.
func TestDBImporterRefusesMissingSource(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	target, err := sqlite.Open(ctx, filepath.Join(dir, "target.db"), nil)
	if err != nil {
		t.Fatalf("open target: %v", err)
	}
	defer target.Close()

	missing := filepath.Join(dir, "does-not-exist.db")
	if _, err := (DBImporter{S: target}).Import(ctx, "", Source{Format: FormatNativeDB, Path: missing}, nil); err == nil {
		t.Fatal("import of a missing .db succeeded")
	}
	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Fatalf("the missing source was created on disk (stat: %v)", err)
	}

	if _, err := (DBImporter{S: target}).Import(ctx, "", Source{Format: FormatNativeDB, Path: dir}, nil); err == nil {
		t.Fatal("import of a directory succeeded")
	}
}

package ingest

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// exportTo runs ExportSQLite into a temporary file and copies its bytes to w,
// for tests that read an export as a stream.
func exportTo(ctx context.Context, src storage.Storage, w io.Writer, opts ExportOptions) error {
	dir, err := os.MkdirTemp("", "blunderdb-export-test-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	report, err := ExportSQLite(ctx, src, "", filepath.Join(dir, "export.sqlite"), opts)
	if err != nil {
		return err
	}
	f, err := os.Open(report.Path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(w, f)
	return err
}

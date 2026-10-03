package ingest

import (
	"archive/tar"
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func writeTar(t *testing.T, names ...string) string {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, n := range names {
		if err := tw.WriteHeader(&tar.Header{Name: n, Mode: 0o644, Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "a.tar")
	if err := os.WriteFile(p, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestExtractArchive_RefusesTraversal(t *testing.T) {
	if _, err := ExtractArchive(writeTar(t, "../evil.xg"), t.TempDir(), 1<<20); err == nil {
		t.Fatal("an entry leaving the archive was accepted")
	}
}

func TestExtractArchive_CapsEntries(t *testing.T) {
	names := make([]string, MaxArchiveEntries+1)
	for i := range names {
		names[i] = "x.pdf" // skipped entries count too
	}
	_, err := ExtractArchive(writeTar(t, names...), t.TempDir(), 1<<20)
	if !errors.Is(err, ErrArchiveTooManyEntries) {
		t.Fatalf("err = %v, want ErrArchiveTooManyEntries", err)
	}
}

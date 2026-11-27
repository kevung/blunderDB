//go:build linux && (dev || production) && sheetpdftest

package gui

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/arkqr"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

// TestSheetPDF renders print pages in WebKitGTK and checks the PDF carries
// the page's reimport payload. It needs a display (xvfb-run) and runs with
// -tags production,webkit2_41,sheetpdftest. BLUNDERDB_SHEETPDF_PAGES lists
// further pages (separated by ':'); each PDF is written beside its page.
func TestSheetPDF(t *testing.T) {
	if !startGTKLoop() {
		t.Skip("no display")
	}
	pages := []string{
		"../../frontend/tests/e2e/fixtures/printsheet/match.html",
		"../../frontend/tests/e2e/fixtures/printsheet/list-codes.html",
	}
	if extra := os.Getenv("BLUNDERDB_SHEETPDF_PAGES"); extra != "" {
		pages = append(pages, strings.Split(extra, ":")...)
	}
	for _, p := range pages {
		t.Run(filepath.Base(p), func(t *testing.T) {
			page, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			out := filepath.Join(t.TempDir(), "sheet.pdf")
			if filepath.IsAbs(p) {
				out = strings.TrimSuffix(p, filepath.Ext(p)) + ".pdf"
			}
			if err := writeSheetPDF(context.Background(), out, page); err != nil {
				t.Fatal(err)
			}
			pdf, err := os.ReadFile(out)
			if err != nil || !bytes.HasPrefix(pdf, []byte("%PDF-")) {
				t.Fatalf("not a PDF: %v", err)
			}
			if !arkqr.IsPage(page) {
				return
			}
			want, _ := arkqr.ReadPage(page)
			got, err := ingest.ReadArchive(pdf)
			if err != nil || len(got) != len(want) {
				t.Fatalf("payload: %d frames, want %d (%v)", len(got), len(want), err)
			}
		})
	}
}

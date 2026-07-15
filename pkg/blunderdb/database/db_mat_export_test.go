package database

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kevung/gnubgparser"
)

// TestExportMatchMAT imports a real .mat into a Database, exports it back through
// the desktop path (scope ""), and checks the written file re-parses and has the
// same game count as the original — the GUI/CLI export wiring end to end.
func TestExportMatchMAT(t *testing.T) {
	matFile := filepath.Join("testdata", "test.mat")
	content, err := os.ReadFile(matFile)
	if err != nil {
		t.Skipf("test.mat not found: %v", err)
	}

	db := newTestDB(t)
	matchID, err := db.ImportGnuBGMatchFromText(string(content))
	if err != nil {
		t.Fatalf("ImportGnuBGMatchFromText: %v", err)
	}

	// SuggestMatFilename should produce a .mat name for this match.
	name, err := db.SuggestMatFilename(matchID)
	if err != nil {
		t.Fatalf("SuggestMatFilename: %v", err)
	}
	if !strings.HasSuffix(name, ".mat") {
		t.Errorf("suggested name %q does not end in .mat", name)
	}

	out := filepath.Join(t.TempDir(), name)
	if err := db.ExportMatchMAT(matchID, out); err != nil {
		t.Fatalf("ExportMatchMAT: %v", err)
	}

	rendered, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read exported .mat: %v", err)
	}
	if !strings.Contains(string(rendered), "point match") {
		t.Fatalf("exported file missing match header:\n%s", rendered)
	}

	rt, err := gnubgparser.ParseMAT(strings.NewReader(string(rendered)))
	if err != nil {
		t.Fatalf("re-parse exported .mat: %v", err)
	}
	orig, err := gnubgparser.ParseMATFile(matFile)
	if err != nil {
		t.Fatalf("parse original: %v", err)
	}
	if len(rt.Games) != len(orig.Games) {
		t.Errorf("game count: exported %d vs original %d", len(rt.Games), len(orig.Games))
	}
}

// TestExportMatchMATReadErrorLeavesNoFile: a failing read (unknown match id)
// must not create a truncated output file.
func TestExportMatchMATReadErrorLeavesNoFile(t *testing.T) {
	db := newTestDB(t)
	out := filepath.Join(t.TempDir(), "should-not-exist.mat")
	if err := db.ExportMatchMAT(999999, out); err == nil {
		t.Fatal("expected error exporting a nonexistent match, got nil")
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Errorf("output file should not exist after a read failure, stat err = %v", err)
	}
}

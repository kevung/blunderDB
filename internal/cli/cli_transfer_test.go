package cli

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// seedTransferLibrary writes a library at path from an XG fixture.
func seedTransferLibrary(t *testing.T, path, fixture string) {
	t.Helper()
	ctx := context.Background()
	s, err := sqlite.Open(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err := (ingest.XGImporter{S: s}).Import(ctx, "", ingest.Source{Format: ingest.FormatXG, Path: filepath.Join("testdata", fixture)}, nil); err != nil {
		t.Fatal(err)
	}
}

// libraryRows counts a library's positions and matches.
func libraryRows(t *testing.T, path string) [2]int {
	t.Helper()
	ctx := context.Background()
	s, err := sqlite.Open(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var n [2]int
	for _, err := range s.Positions().List(ctx, "", storage.ListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		n[0]++
	}
	for _, err := range s.Matches().List(ctx, "", storage.MatchListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		n[1]++
	}
	return n
}

func runTransfer(t *testing.T, args ...string) ingest.TransferReport {
	t.Helper()
	cli := &CLI{db: NewDatabase()}
	defer cli.db.Close()
	var runErr error
	out := captureStdout(t, func() {
		runErr = cli.runTransfer(append(args, "--format", "json"))
	})
	if runErr != nil {
		t.Fatalf("transfer %v: %v", args, runErr)
	}
	var rep ingest.TransferReport
	if err := json.Unmarshal([]byte(out), &rep); err != nil {
		t.Fatalf("output %q: %v", out, err)
	}
	return rep
}

// transfer A → B then B → A leaves both libraries holding the union once; a
// second round adds nothing.
func TestCLI_TransferBothWays(t *testing.T) {
	dir := tempDir(t)
	a, b := filepath.Join(dir, "a.db"), filepath.Join(dir, "b.db")
	seedTransferLibrary(t, a, "match_with_comment.xg")
	seedTransferLibrary(t, b, "test.xg")
	before := libraryRows(t, a)

	rep := runTransfer(t, "--from", a, "--to", b)
	if rep.Exported.Positions != before[0] || rep.Exported.Matches != 1 {
		t.Errorf("A → B read %+v, want A's %d positions and its match", rep.Exported, before[0])
	}
	runTransfer(t, "--from", b, "--to", a)
	union := libraryRows(t, a)
	if got := libraryRows(t, b); got != union || union[1] != 2 {
		t.Fatalf("after A → B → A: A %v, B %v, want the same union with 2 matches", union, got)
	}
	again := runTransfer(t, "--from", a, "--to", b)
	if again.Imported.Matches != 0 || again.Imported.SkippedDuplicates != 2 {
		t.Errorf("a second round added matches: %+v", again.Imported)
	}
	if got := libraryRows(t, b); got != union {
		t.Errorf("a second round changed B: %v, was %v", got, union)
	}
}

// The source must exist and differ from the destination.
func TestCLI_TransferRefusals(t *testing.T) {
	dir := tempDir(t)
	a := filepath.Join(dir, "a.db")
	seedTransferLibrary(t, a, "match_with_comment.xg")
	for name, args := range map[string][]string{
		"missing source": {"--from", filepath.Join(dir, "none.db"), "--to", a},
		"same library":   {"--from", a, "--to", filepath.Join(dir, ".", "a.db")},
		"no destination": {"--from", a},
	} {
		cli := &CLI{db: NewDatabase()}
		var err error
		captureStdout(t, func() { err = cli.runTransfer(args) })
		cli.db.Close()
		if err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

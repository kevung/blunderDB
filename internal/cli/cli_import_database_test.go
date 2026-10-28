package cli

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// import --type database merges a .db as the desktop and the daemon do: the
// match travels with its tournament, and a second run adds nothing.
func TestCLI_ImportDatabase(t *testing.T) {
	ctx := context.Background()
	dir := tempDir(t)
	srcPath := filepath.Join(dir, "club.db")
	src, err := sqlite.Open(ctx, srcPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	p := domain.InitializePosition()
	if _, err := src.Positions().Save(ctx, "", &p); err != nil {
		t.Fatal(err)
	}
	tour, _ := src.Tournaments().Create(ctx, "", "Open de Lyon", "", "")
	mid, err := src.Matches().Save(ctx, "", &domain.Match{Player1Name: "Bob", Player2Name: "Zoé", MatchLength: 5, MatchHash: "h-bob"})
	if err != nil {
		t.Fatal(err)
	}
	_ = src.Tournaments().AddMatch(ctx, "", tour, mid)
	src.Close()

	target := filepath.Join(dir, "mine.db")
	for pass, want := range []float64{1, 0} {
		cli := &CLI{db: NewDatabase()}
		var runErr error
		out := captureStdout(t, func() {
			runErr = cli.runImport([]string{"--db", target, "--type", "database", "--file", srcPath, "--format", "json"})
		})
		cli.db.Close()
		if runErr != nil {
			t.Fatalf("pass %d: %v", pass+1, runErr)
		}
		var res map[string]any
		if err := json.Unmarshal([]byte(out), &res); err != nil {
			t.Fatalf("pass %d: output %q: %v", pass+1, out, err)
		}
		if res["matches"] != want || res["tournaments"] != want || res["matchesKnown"] != 1-want {
			t.Errorf("pass %d: %v, want %v match and tournament added", pass+1, res, want)
		}
	}
}

// import --type database --analyze counts the file and writes nothing;
// --max-positions carries the first positions and says what it left out.
func TestCLI_ImportDatabaseAnalyzeAndCap(t *testing.T) {
	ctx := context.Background()
	dir := tempDir(t)
	srcPath := filepath.Join(dir, "club.db")
	src, err := sqlite.Open(ctx, srcPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	for n := 1; n <= 3; n++ {
		p := domain.InitializePosition()
		p.Score = [2]int{n, 0}
		p.IndividuallyImported = true
		if _, err := src.Positions().Save(ctx, "", &p); err != nil {
			t.Fatal(err)
		}
	}
	src.Close()

	target := filepath.Join(dir, "mine.db")
	run := func(args ...string) map[string]any {
		t.Helper()
		cli := &CLI{db: NewDatabase()}
		var runErr error
		out := captureStdout(t, func() {
			runErr = cli.runImport(append([]string{"--db", target, "--type", "database", "--file", srcPath, "--format", "json"}, args...))
		})
		cli.db.Close()
		if runErr != nil {
			t.Fatalf("%v: %v", args, runErr)
		}
		var res map[string]any
		if err := json.Unmarshal([]byte(out), &res); err != nil {
			t.Fatalf("%v: output %q: %v", args, out, err)
		}
		return res
	}

	for range 2 {
		if res := run("--analyze"); res["positions"] != float64(3) || res["positionsHeld"] != float64(0) {
			t.Fatalf("--analyze = %v, want 3 positions, none held, every time: it writes nothing", res)
		}
	}
	if res := run("--max-positions", "2"); res["added"] != float64(2) || res["leftOut"] != float64(1) {
		t.Fatalf("--max-positions 2 = %v, want 2 added, 1 left out", res)
	}
	if res := run("--analyze"); res["positionsHeld"] != float64(2) {
		t.Fatalf("--analyze after the capped import = %v, want 2 held", res)
	}
}

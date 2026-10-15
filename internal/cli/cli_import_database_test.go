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

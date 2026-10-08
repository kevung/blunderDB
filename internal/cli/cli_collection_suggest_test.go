package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// TestCLI_CollectionSuggest runs the proposal on an analysed match and keeps
// it as a collection: the collection holds exactly the proposed positions,
// and a second run leaves them out, now handled (ADR-0078 §6).
func TestCLI_CollectionSuggest(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	if err := cli.Run([]string{"import", "--db", dbPath, "--type", "match", "--file",
		testdataPath("test.xg")}); err != nil {
		t.Fatalf("import: %v", err)
	}
	run := func(args ...string) storage.ReferenceSuggestions {
		t.Helper()
		var res struct {
			storage.ReferenceSuggestions
			CollectionID int64
		}
		out := captureStdout(t, func() {
			if err := cli.Run(append([]string{"collection", "suggest", "--db", dbPath, "--format", "json"}, args...)); err != nil {
				t.Fatalf("collection suggest: %v", err)
			}
		})
		if err := json.Unmarshal([]byte(out[strings.Index(out, "{"):]), &res); err != nil {
			t.Fatalf("suggest JSON: %v\n%s", err, out)
		}
		if res.CollectionID != 0 {
			ids, err := cli.db.ListCollectionPositionIDs(res.CollectionID, 0, 100)
			if err != nil {
				t.Fatalf("ListCollectionPositionIDs: %v", err)
			}
			if len(ids) != len(res.References) {
				t.Errorf("collection holds %d positions, proposal %d", len(ids), len(res.References))
			}
		}
		return res.ReferenceSuggestions
	}
	first := run("--size", "10", "--collection", "References")
	if len(first.References) == 0 || len(first.References) > 10 || first.Size != 10 {
		t.Fatalf("proposal = %+v, want 1 to 10 references", first)
	}
	second := run("--size", "10")
	if second.Handled < len(first.References) {
		t.Errorf("Handled %d after keeping %d references", second.Handled, len(first.References))
	}
	kept := map[int64]bool{}
	for _, r := range first.References {
		kept[r.PositionID] = true
	}
	for _, r := range second.References {
		if kept[r.PositionID] {
			t.Errorf("position %d proposed again once collected", r.PositionID)
		}
	}

	for _, args := range [][]string{
		{"collection", "suggest", "--db", dbPath, "--size", "0"},
		{"collection", "suggest", "--db", dbPath, "--size", "51"},
		{"collection", "suggest", "--db", dbPath, "--match", "x"},
		{"collection", "suggest", "--db", dbPath, "--match", "99999", "--deck", "Nothing"},
	} {
		var err error
		captureStdout(t, func() { err = cli.Run(args) })
		if err == nil {
			t.Errorf("%v: want an error", args)
		}
	}
}

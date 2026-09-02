package cli

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// seedCollection imports the XG fixture and gathers its first n positions
// into a new collection, returning the collection id and the position ids.
func seedCollection(t *testing.T, cli *CLI, name string, n int) (int64, []int64) {
	t.Helper()
	if _, err := cli.db.ImportXGMatch(testdataPath("test.xg")); err != nil {
		t.Fatalf("ImportXGMatch: %v", err)
	}
	positions, err := cli.db.LoadAllPositions()
	if err != nil {
		t.Fatalf("LoadAllPositions: %v", err)
	}
	if len(positions) < n {
		t.Fatalf("fixture has %d positions, need %d", len(positions), n)
	}
	id, err := cli.db.CreateCollection(name, "")
	if err != nil {
		t.Fatalf("CreateCollection: %v", err)
	}
	ids := make([]int64, 0, n)
	for _, p := range positions[:n] {
		ids = append(ids, p.ID)
	}
	if err := cli.db.AddPositionsToCollection(id, ids); err != nil {
		t.Fatalf("AddPositionsToCollection: %v", err)
	}
	return id, ids
}

func TestCLI_CollectionList(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)

	out := captureStdout(t, func() {
		if err := cli.Run([]string{"collection", "list", "--db", dbPath}); err != nil {
			t.Fatalf("collection list: %v", err)
		}
	})
	if !strings.Contains(out, "No collections found") {
		t.Errorf("expected empty-DB message, got:\n%s", out)
	}

	id, _ := seedCollection(t, cli, "Openings", 3)

	out = captureStdout(t, func() {
		if err := cli.Run([]string{"collection", "list", "--db", dbPath}); err != nil {
			t.Fatalf("collection list: %v", err)
		}
	})
	if !strings.Contains(out, "Openings") || !strings.Contains(out, "Found 1 collection(s)") {
		t.Errorf("text listing missing the collection:\n%s", out)
	}

	out = captureStdout(t, func() {
		if err := cli.Run([]string{"collection", "list", "--db", dbPath, "--format", "json"}); err != nil {
			t.Fatalf("collection list --format json: %v", err)
		}
	})
	var rows []Collection
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("json output does not parse: %v\n%s", err, out)
	}
	if len(rows) != 1 || rows[0].ID != id || rows[0].PositionCount != 3 {
		t.Errorf("json rows = %+v, want one collection %d with 3 positions", rows, id)
	}

	out = captureStdout(t, func() {
		if err := cli.Run([]string{"collection", "list", "--db", dbPath, "--format", "csv"}); err != nil {
			t.Fatalf("collection list --format csv: %v", err)
		}
	})
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 || lines[0] != "id,name,positions,description,created_at" {
		t.Errorf("csv output = %q, want a header and one row", out)
	}
}

func TestCLI_CollectionShow(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	id, ids := seedCollection(t, cli, "Openings", 2)

	out := captureStdout(t, func() {
		if err := cli.Run([]string{"collection", "show", "--db", dbPath, "--id", "1"}); err != nil {
			t.Fatalf("collection show: %v", err)
		}
	})
	if !strings.Contains(out, "Collection 1: Openings") || !strings.Contains(out, "Positions: 2") {
		t.Errorf("text output missing the collection header:\n%s", out)
	}

	out = captureStdout(t, func() {
		if err := cli.Run([]string{"collection", "show", "--db", dbPath, "--id", "1", "--format", "json"}); err != nil {
			t.Fatalf("collection show --format json: %v", err)
		}
	})
	var got struct {
		Collection Collection              `json:"collection"`
		Positions  []collectionPositionRow `json:"positions"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("json output does not parse: %v\n%s", err, out)
	}
	if got.Collection.ID != id || len(got.Positions) != 2 {
		t.Fatalf("json = %+v, want collection %d with 2 positions", got, id)
	}
	for i, row := range got.Positions {
		if row.ID != ids[i] {
			t.Errorf("position %d: id = %d, want %d (collection order)", i, row.ID, ids[i])
		}
		if row.Index <= 0 {
			t.Errorf("position %d: index = %d, want a 1-based database index", i, row.Index)
		}
		// The XG import stores no XGID, so this is the generated one: it
		// must decode back to the stored board and score.
		decoded, err := domain.DecodeXGID(row.XGID)
		if err != nil {
			t.Fatalf("position %d: generated XGID %q does not decode: %v", i, row.XGID, err)
		}
		stored, err := cli.db.LoadPosition(int(row.ID))
		if err != nil {
			t.Fatalf("LoadPosition(%d): %v", row.ID, err)
		}
		if decoded.Score != stored.Score || decoded.PlayerOnRoll != stored.PlayerOnRoll || decoded.Dice != stored.Dice {
			t.Errorf("position %d: XGID %q round-trips to %+v, stored %+v", i, row.XGID, decoded, *stored)
		}
		// An empty point's colour is a storage detail (0 stored, None
		// decoded); only occupied points carry information.
		for pt := range stored.Board.Points {
			want, got := stored.Board.Points[pt], decoded.Board.Points[pt]
			if want.Checkers != got.Checkers || (want.Checkers > 0 && want.Color != got.Color) {
				t.Errorf("position %d: XGID %q point %d = %+v, stored %+v", i, row.XGID, pt, got, want)
			}
		}
	}

	if err := cli.Run([]string{"collection", "show", "--db", dbPath, "--id", "99"}); err == nil {
		t.Error("expected an error for an unknown collection id")
	}
}

func TestCLI_CollectionCreateRenameDelete(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)

	out := captureStdout(t, func() {
		if err := cli.Run([]string{"collection", "create", "--db", dbPath, "--name", "Draft", "--description", "notes"}); err != nil {
			t.Fatalf("collection create: %v", err)
		}
	})
	if !strings.Contains(out, "Successfully created collection \"Draft\" (ID: 1)") {
		t.Errorf("create output = %q", out)
	}

	captureStdout(t, func() {
		if err := cli.Run([]string{"collection", "rename", "--db", dbPath, "--id", "1", "--name", "Final"}); err != nil {
			t.Fatalf("collection rename: %v", err)
		}
	})
	c, err := cli.db.GetCollectionByID(1)
	if err != nil {
		t.Fatalf("GetCollectionByID: %v", err)
	}
	if c.Name != "Final" || c.Description != "notes" {
		t.Errorf("after rename: name=%q description=%q, want Final/notes (description kept)", c.Name, c.Description)
	}

	captureStdout(t, func() {
		if err := cli.Run([]string{"collection", "delete", "--db", dbPath, "--id", "1", "--confirm"}); err != nil {
			t.Fatalf("collection delete: %v", err)
		}
	})
	collections, err := cli.db.GetAllCollections()
	if err != nil {
		t.Fatalf("GetAllCollections: %v", err)
	}
	if len(collections) != 0 {
		t.Errorf("collection still present after delete: %+v", collections)
	}

	if err := cli.Run([]string{"collection", "rename", "--db", dbPath, "--id", "1", "--name", "X"}); err == nil {
		t.Error("expected an error renaming a deleted collection")
	}
}

func TestCLI_CollectionExport(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	id, ids := seedCollection(t, cli, "Openings", 3)
	outPath := filepath.Join(t.TempDir(), "openings.db")

	captureStdout(t, func() {
		if err := cli.Run([]string{"collection", "export", "--db", dbPath, "--id", "1", "--out", outPath}); err != nil {
			t.Fatalf("collection export: %v", err)
		}
	})

	exported := NewDatabase()
	if err := exported.OpenDatabase(outPath); err != nil {
		t.Fatalf("open exported file: %v", err)
	}
	defer exported.Close()
	collections, err := exported.GetAllCollections()
	if err != nil {
		t.Fatalf("GetAllCollections on export: %v", err)
	}
	if len(collections) != 1 || collections[0].Name != "Openings" || collections[0].PositionCount != len(ids) {
		t.Fatalf("exported collections = %+v, want %q with %d positions (source id %d)", collections, "Openings", len(ids), id)
	}
	positions, err := exported.LoadAllPositions()
	if err != nil {
		t.Fatalf("LoadAllPositions on export: %v", err)
	}
	if len(positions) != len(ids) {
		t.Errorf("exported file holds %d positions, want only the collection's %d", len(positions), len(ids))
	}

	if err := cli.Run([]string{"collection", "export", "--db", dbPath, "--id", "42", "--out", outPath}); err == nil {
		t.Error("expected an error exporting an unknown collection id")
	}
}

func TestCLI_CollectionUsageErrors(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	cases := [][]string{
		{"collection"},
		{"collection", "bogus", "--db", dbPath},
		{"collection", "list"},
		{"collection", "show", "--db", dbPath},
		{"collection", "create", "--db", dbPath},
		{"collection", "rename", "--db", dbPath, "--id", "1"},
		{"collection", "delete", "--db", dbPath},
		{"collection", "export", "--db", dbPath, "--id", "1"},
		{"collection", "list", "--db", dbPath, "--format", "xml"},
	}
	for _, args := range cases {
		var err error
		captureStdout(t, func() { err = cli.Run(args) })
		if err == nil {
			t.Errorf("%v: expected an error", args)
		}
	}
}

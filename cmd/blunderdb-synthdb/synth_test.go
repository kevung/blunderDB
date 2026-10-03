package main

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// Two variants of one fixture must land on distinct position rows: if the
// score shift stopped reaching the Zobrist hash, the second match would
// deduplicate entirely onto the first and a "million-position" base would
// hold a few thousand.
func TestGenerate_VariantsDoNotDeduplicate(t *testing.T) {
	ctx := context.Background()
	fixture := filepath.Join("..", "..", "testdata", "charlot1-charlot2_7p_2025-11-08-2305.xg")
	open := func() (*sqlite.Storage, string) {
		path := filepath.Join(t.TempDir(), "synth.db")
		store, err := sqlite.Open(ctx, path, nil)
		if err != nil {
			t.Fatal(err)
		}
		return store, path
	}

	one, _ := open()
	res, err := Generate(ctx, one, Options{Fixtures: []string{fixture}, Positions: 1, Seed: 1})
	if err != nil {
		t.Fatal(err)
	}
	one.Close()
	perMatch := res.Positions

	two, path := open()
	res, err = Generate(ctx, two, Options{Fixtures: []string{fixture}, Positions: perMatch + 1, Seed: 1})
	if err != nil {
		t.Fatal(err)
	}
	two.Close()
	if res.Matches != 2 {
		t.Fatalf("want 2 matches for %d positions, got %d", perMatch+1, res.Matches)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var positions, matches int
	if err := db.QueryRow(`SELECT COUNT(*) FROM position`).Scan(&positions); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM match`).Scan(&matches); err != nil {
		t.Fatal(err)
	}
	if matches != 2 {
		t.Fatalf("want 2 matches, got %d", matches)
	}
	if positions < perMatch*3/2 {
		t.Fatalf("variant 1 deduplicated onto variant 0: %d positions for %d per match", positions, perMatch)
	}
}

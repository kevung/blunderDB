package ingest

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// DBAnalysis is what a native .db would bring to a target, counted without
// writing anything: the figures a client shows before sending the import.
type DBAnalysis struct {
	// SourceVersion is the schema version the file declares.
	SourceVersion string `json:"sourceVersion"`
	// Positions counts the file's positions; PositionsHeld those the target
	// already stores (by Zobrist hash), which the import merges rather than
	// adds.
	Positions     int `json:"positions"`
	PositionsHeld int `json:"positionsHeld"`
	// Matches counts the matches the import would create, MatchesKnown those
	// the target already holds by fingerprint (PlanImportMatches).
	Matches      int `json:"matches"`
	MatchesKnown int `json:"matchesKnown"`
	Collections  int `json:"collections"`
	Lessons      int `json:"lessons"`
	Decks        int `json:"decks"`
	// Unreadable names the families the file is too old to hold.
	Unreadable []string `json:"unreadable,omitempty"`
}

// AnalyzeDB opens the native .db at path as DBImporter would — so a file the
// import would refuse is refused here with the same error — and counts what
// importing it into target would bring. It writes nothing to target.
func AnalyzeDB(ctx context.Context, target storage.Stores, scope, path string) (DBAnalysis, error) {
	var a DBAnalysis
	source, err := openSourceDB(ctx, path)
	if err != nil {
		return a, err
	}
	defer source.Close()

	if a.SourceVersion, err = source.Version(ctx); err != nil {
		return a, fmt.Errorf("ingest: read source version: %w", err)
	}

	var hashes []uint64
	for p, err := range source.Positions().List(ctx, scope, storage.ListOpts{}) {
		if err != nil {
			return a, fmt.Errorf("ingest: list source positions: %w", err)
		}
		hashes = append(hashes, engine.ZobristHash(p))
	}
	a.Positions = len(hashes)
	for _, h := range hashes {
		if err := ctx.Err(); err != nil {
			return a, err
		}
		_, held, err := target.Positions().Exists(ctx, scope, h)
		if err != nil {
			return a, err
		}
		if held {
			a.PositionsHeld++
		}
	}

	plan, err := PlanImportMatches(ctx, target, scope, source)
	if err != nil {
		return a, err
	}
	a.Matches, a.MatchesKnown = plan.ToImport, plan.Known
	if plan.Unreadable {
		a.Unreadable = append(a.Unreadable, "matches")
	}

	colls, err := ReadSourceCollections(ctx, source, scope)
	if err != nil {
		return a, err
	}
	a.Collections = len(colls)
	if source.HasTable(ctx, "lesson") {
		lessons, err := ReadSourceLessons(ctx, source, scope)
		if err != nil {
			return a, err
		}
		a.Lessons = len(lessons)
	}
	if source.HasTable(ctx, "anki_deck") {
		decks, err := ReadSourceDecks(ctx, source, scope)
		if err != nil {
			return a, err
		}
		a.Decks = len(decks)
	}
	return a, nil
}

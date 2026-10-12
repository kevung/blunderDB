package storagetest

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testExportPlanMatchesReport: the plan of an export counts, family by
// family, what the export then writes, for a whole tenant and for narrowed
// selections; and a living collection whose query names a match left behind
// is announced frozen.
func testExportPlanMatchesReport(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	m1, pos1 := statsFixtureMatch(t, s, 0, "Alice", "Bob")
	m2, pos2 := statsFixtureMatch(t, s, 2, "Carol", "Dan")
	lone := saveDecisions(t, s, "", 10, 11)

	for _, c := range []struct {
		pos  int64
		text string
	}{{pos1[0], "a"}, {pos1[0], "b"}, {lone[0], "c"}} {
		if _, err := s.Comments().Add(ctx, "", c.pos, c.text); err != nil {
			t.Fatal(err)
		}
	}
	tid, err := s.Tournaments().Create(ctx, "", "Open", "2026-01-01", "Paris")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Tournaments().AddMatch(ctx, "", tid, m1); err != nil {
		t.Fatal(err)
	}
	plain, err := s.Collections().Create(ctx, "", "Plain", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Collections().AddPositions(ctx, "", plain, []int64{lone[0], pos1[0]}); err != nil {
		t.Fatal(err)
	}
	living, err := s.Collections().Create(ctx, "", "Of Carol", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.SetCollectionFilter(ctx, s, "", living, "ma"+strconv.FormatInt(m2, 10)); err != nil {
		t.Fatal(err)
	}
	lesson, err := s.Lessons().Create(ctx, "", "Leçon", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Lessons().AddStep(ctx, "", lesson, domain.LessonStep{Title: "Série", CollectionID: plain}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Anki().CreateDeck(ctx, "", "deck", "", domain.AnkiSourceSearch, 0, ""); err != nil {
		t.Fatal(err)
	}
	// A search deck of m2's positions whose command names m2: chosen alone, its
	// members travel but m2 does not, so its command is dropped on the way.
	m2Deck, err := s.Anki().CreateDeck(ctx, "", "Of m2", "", domain.AnkiSourceSearch, 0,
		fmt.Sprintf(`{"ids":[%d,%d],"command":"s ma%d"}`, pos2[0], pos2[1], m2))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Anki().SyncWithPositions(ctx, "", m2Deck, pos2[:]); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Transcriptions().Save(ctx, "", &storage.Transcription{FormatVersion: "1", MatchID: m1, Label: "m1", Document: "{}"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Transcriptions().Save(ctx, "", &storage.Transcription{FormatVersion: "1", Label: "loose", Document: "{}"}); err != nil {
		t.Fatal(err)
	}
	for _, a := range [][2]string{{"Ali", "Alice"}, {"Caro", "Carol"}} {
		if err := s.Aliases().Set(ctx, "", storage.AliasPlayer, a[0], a[1]); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.Filters().Save(ctx, "", "of m2", "s ma"+strconv.FormatInt(m2, 10)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Filters().Save(ctx, "", "all", "s"); err != nil {
		t.Fatal(err)
	}

	narrow := func(sel ingest.Selection) ingest.ExportOptions {
		o := ingest.WholeTenant(ingest.FormatSQLite)
		o.Selection, o.AnkiDecks, o.AnkiHistory = sel, false, false
		return o
	}
	allOff := ingest.WholeTenant(ingest.FormatSQLite)
	allOff.Analysis, allOff.Comments, allOff.PlayedMoves, allOff.FilterLibrary = false, false, false, false
	allOff.AnkiDecks, allOff.Transcriptions, allOff.Aliases, allOff.AnkiHistory = false, false, false, false
	stored, err := s.Positions().LoadByIDs(ctx, "", []int64{lone[0], lone[1]})
	if err != nil || len(stored) != 2 {
		t.Fatalf("LoadByIDs: %v (%d)", err, len(stored))
	}
	// Positions handed over whole: one without an id repeats a stored board, and
	// the file's Zobrist dedup writes it once.
	unsaved := stored[0]
	unsaved.ID = 0
	cases := []struct {
		name             string
		opts             ingest.ExportOptions
		frozen, degraded []int64
	}{
		{"whole tenant", ingest.WholeTenant(ingest.FormatSQLite), nil, nil},
		{"every option off", allOff, nil, nil},
		{"one deck chosen", narrow(ingest.Selection{DeckIDs: []int64{m2Deck}, DeckPositions: true}), nil, []int64{m2Deck}},
		{"living collection alone", narrow(ingest.Selection{CollectionIDs: []int64{living}, CollectionPositions: true}), []int64{living}, nil},
		{"positions handed over whole", narrow(ingest.Selection{Positions: []*domain.Position{&stored[0], &unsaved, &stored[1]}}), nil, nil},
		{"one tournament", narrow(ingest.Selection{
			TournamentIDs: []int64{tid}, TournamentMatches: true, MatchPositions: true,
			CollectionIDs: []int64{living}, CollectionPositions: true,
		}), []int64{living}, nil},
		{"one lesson and a stale id", narrow(ingest.Selection{
			LessonIDs: []int64{lesson, 9999}, LessonContents: true, PositionIDs: []int64{lone[1], 9998},
		}), nil, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			plan, err := ingest.PlanExport(ctx, s, "", c.opts)
			if err != nil {
				t.Fatalf("PlanExport: %v", err)
			}
			report, err := ingest.ExportSQLite(ctx, s, "", filepath.Join(t.TempDir(), "out.db"), c.opts)
			if err != nil {
				t.Fatalf("ExportSQLite: %v", err)
			}
			if m := plan.Mismatch(report); m != "" {
				t.Errorf("plan and report disagree on %s (plan %+v)", m, plan.Counts())
			}
			page, err := ingest.PlanExportIDs(ctx, s, "", c.opts, ingest.PlanPositions, 1, 2)
			if err != nil {
				t.Fatalf("PlanExportIDs: %v", err)
			}
			if page.Total > plan.Positions || len(page.IDs) != min(2, max(0, page.Total-1)) {
				t.Errorf("positions page %+v for a count of %d", page, plan.Positions)
			}
			if c.name == "positions handed over whole" && plan.Positions != 2 {
				t.Errorf("positions handed over whole: planned %d, want 2", plan.Positions)
			}
			if c.name == "whole tenant" {
				if _, err := ingest.PlanExportIDs(ctx, s, "", c.opts, "aliases", 0, 0); err == nil {
					t.Error("aliases have no ids, yet PlanExportIDs answered")
				}
				want := map[string]int{"positions": 6, "comments": 3, "matches": 2, "tournaments": 1,
					"collections": 2, "lessons": 1, "ankiDecks": 2, "transcriptions": 2, "aliases": 2, "filters": 2}
				for k, n := range want {
					if got := plan.Counts()[k]; got != n {
						t.Errorf("%s: planned %d, want %d", k, got, n)
					}
				}
			}
			if got := rowIDs(plan.FrozenCollections); !slices.Equal(got, c.frozen) {
				t.Errorf("frozen collections %v, want %v", got, c.frozen)
			}
			if got := rowIDs(plan.DegradedAnkiDecks); !slices.Equal(got, c.degraded) {
				t.Errorf("degraded decks %v, want %v", got, c.degraded)
			}
		})
	}
}

func rowIDs(rows []ingest.PlannedRow) []int64 {
	var ids []int64
	for _, r := range rows {
		ids = append(ids, r.ID)
	}
	return ids
}

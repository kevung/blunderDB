package storagetest

import (
	"context"
	"path/filepath"
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
	m2, _ := statsFixtureMatch(t, s, 2, "Carol", "Dan")
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
	cases := []struct {
		name   string
		opts   ingest.ExportOptions
		frozen []int64
	}{
		{"whole tenant", ingest.WholeTenant(ingest.FormatSQLite), nil},
		{"one tournament", narrow(ingest.Selection{
			TournamentIDs: []int64{tid}, TournamentMatches: true, MatchPositions: true,
			CollectionIDs: []int64{living}, CollectionPositions: true,
		}), []int64{living}},
		{"one lesson and a stale id", narrow(ingest.Selection{
			LessonIDs: []int64{lesson, 9999}, LessonContents: true, PositionIDs: []int64{lone[1], 9998},
		}), nil},
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
			if len(plan.Positions.IDs) != plan.Positions.Count {
				t.Errorf("positions: %d ids for a count of %d", len(plan.Positions.IDs), plan.Positions.Count)
			}
			if c.name == "whole tenant" {
				want := map[string]int{"positions": 6, "comments": 3, "matches": 2, "tournaments": 1,
					"collections": 2, "lessons": 1, "ankiDecks": 1, "transcriptions": 2, "aliases": 2, "filters": 2}
				for k, n := range want {
					if got := plan.Counts()[k]; got != n {
						t.Errorf("%s: planned %d, want %d", k, got, n)
					}
				}
			}
			var frozen []int64
			for _, f := range plan.FrozenCollections {
				frozen = append(frozen, f.ID)
			}
			if len(frozen) != len(c.frozen) || (len(frozen) > 0 && frozen[0] != c.frozen[0]) {
				t.Errorf("frozen collections %v, want %v", frozen, c.frozen)
			}
		})
	}
}

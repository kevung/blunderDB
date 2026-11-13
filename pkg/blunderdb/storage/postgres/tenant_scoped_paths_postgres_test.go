//go:build postgres

package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/apkg"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/maintenance"
	"github.com/kevung/blunderdb/pkg/blunderdb/searchquery"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// The paths TestTenantReadsNameTheirTenantWithoutRLS records beyond the
// import: each sends the statements a user of one tenant sends from the
// search bar, a training session, an Anki deck, the library's headline counts
// and the maintenance panel. A failing call is fatal, so a path that stops
// reaching its statements does not pass by planning nothing.

// exerciseSearch runs every command of the shared search corpus through each
// entry point of the search, `like` ranked against a position of the tenant.
func exerciseSearch(ctx context.Context, t *testing.T, s *Storage, scope string, ids []int64) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "testdata", "search_query_corpus.json"))
	if err != nil {
		t.Fatal(err)
	}
	var corpus struct {
		Cases []struct {
			Command string `json:"command"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &corpus); err != nil {
		t.Fatal(err)
	}
	target := ids[len(ids)/2]
	for _, c := range corpus.Cases {
		f, _ := searchquery.Parse(c.Command)
		if f.LikeFilter {
			f.LikeTargetID = target
			if _, err := s.Search().Rank(ctx, scope, f, storage.ListOpts{Limit: 5}); err != nil {
				t.Fatalf("rank %q: %v", c.Command, err)
			}
			// `like` is served only ranked: every other entry point refuses it
			// before sending a statement.
			if _, err := s.Search().Count(ctx, scope, f); !errors.Is(err, searchquery.ErrRefused) {
				t.Fatalf("count %q: want a refusal, got %v", c.Command, err)
			}
			continue
		}
		if _, err := s.Search().Count(ctx, scope, f); err != nil {
			t.Fatalf("count %q: %v", c.Command, err)
		}
		if _, _, err := s.Search().CountUpTo(ctx, scope, f, 3); err != nil {
			t.Fatalf("count up to %q: %v", c.Command, err)
		}
		if _, err := s.Search().FindIDs(ctx, scope, f, storage.ListOpts{Limit: 5}); err != nil {
			t.Fatalf("find ids %q: %v", c.Command, err)
		}
		if _, _, err := s.Search().IndexOf(ctx, scope, f, target); err != nil {
			t.Fatalf("index of %q: %v", c.Command, err)
		}
		for _, err := range s.Search().Find(ctx, scope, f, storage.ListOpts{Limit: 3}) {
			if err != nil {
				t.Fatalf("find %q: %v", c.Command, err)
			}
		}
	}
}

// exerciseTraining saves a session whose wrong answers point at positions,
// then reads it back the ways the training panel does.
func exerciseTraining(ctx context.Context, t *testing.T, s *Storage, scope string, ids []int64) {
	t.Helper()
	errMp := 120
	session := storage.TrainingSession{Exercise: "decisions", NumbersAsked: 3, Faults: 2}
	for _, id := range ids[:3] {
		session.Items = append(session.Items, storage.TrainingItem{NumberType: "decision", Wrong: true, PositionID: &id, ErrorMp: &errMp})
	}
	if _, err := s.Training().Save(ctx, scope, session); err != nil {
		t.Fatal(err)
	}
	must(s.Training().Sessions(ctx, scope, "decisions", 10))(t)
	must(s.Training().NumberStats(ctx, scope, "decisions"))(t)
	must(s.Training().Missed(ctx, scope, storage.TrainingMissedFilter{Exercise: "decisions", Limit: 10}))(t)
	must(s.Training().DecisionErrors(ctx, scope))(t)
	items := must(s.Training().ItemsOfPosition(ctx, scope, ids[0]))(t)
	if err := s.Training().RelinkItems(ctx, scope, items, ids[1]); err != nil {
		t.Fatal(err)
	}
}

// exerciseAnki builds a deck of positions, reviews it and exports it.
func exerciseAnki(ctx context.Context, t *testing.T, s *Storage, scope string, ids []int64) {
	t.Helper()
	a := s.Anki()
	deck := must(a.CreateDeck(ctx, scope, "plans", "", "search", 0, ""))(t)
	if err := a.SyncWithPositions(ctx, scope, deck, ids[:20]); err != nil {
		t.Fatal(err)
	}
	for _, err := range a.ListDecks(ctx, scope) {
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, err := range a.DeckPositions(ctx, scope, deck) {
		if err != nil {
			t.Fatal(err)
		}
	}
	must(a.DeckPositionIDs(ctx, scope, deck, storage.ListOpts{Limit: 5}))(t)
	must(a.DeckPositionCount(ctx, scope, deck))(t)
	if _, _, err := a.IndexOfDeckPosition(ctx, scope, deck, ids[3]); err != nil {
		t.Fatal(err)
	}
	for _, f := range []domain.AnkiCardFilter{domain.AnkiFilterAll, domain.AnkiFilterNew, domain.AnkiFilterLearning, domain.AnkiFilterReview} {
		must(a.FilteredDeckPositionIDs(ctx, scope, deck, f, storage.ListOpts{Limit: 5}))(t)
		must(a.FilteredDeckPositionCount(ctx, scope, deck, f))(t)
		if _, _, err := a.IndexOfFilteredDeckPosition(ctx, scope, deck, f, ids[3]); err != nil {
			t.Fatal(err)
		}
	}
	must(a.DeckStats(ctx, scope, deck))(t)
	card := orNotFound(a.NextCard(ctx, scope, deck))(t)
	if card == nil {
		t.Fatal("no card to review")
	}
	must(a.ReviewCard(ctx, scope, card.Card.ID, 3))(t)
	orNotFound(a.LinkedCard(ctx, scope, deck, card.Card.ID))(t)
	must(a.UndoReview(ctx, scope, card.Card.ID))(t)
	orNotFound(a.RandomCard(ctx, scope, deck, ids[0]))(t)
	for _, err := range a.DeckCards(ctx, scope, deck) {
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, err := range a.ReviewLog(ctx, scope, deck, 10) {
		if err != nil {
			t.Fatal(err)
		}
	}
	must(a.Retention(ctx, scope, deck))(t)
	must(a.Forecast(ctx, scope, deck, 7))(t)
	must(a.ReviewsByGameType(ctx, scope, "2000-01-01"))(t)
	if err := a.Sync(ctx, scope, deck); err != nil {
		t.Fatal(err)
	}
	must(apkg.Export(ctx, s, scope, apkg.Source{DeckID: deck}, "fr", time.Now(), io.Discard))(t)
}

// exerciseMetadata reads the library's headline counts.
func exerciseMetadata(ctx context.Context, t *testing.T, s *Storage, scope string) {
	t.Helper()
	must(s.Metadata().Counts(ctx, scope))(t)
	must(s.Metadata().EstimatedCounts(ctx, scope, 1<<30))(t)
	must(s.Metadata().StoredBytes(ctx, scope))(t)
	if b, ok := s.Metadata().(interface {
		BlunderCount(context.Context, string) (int, error)
	}); ok {
		must(b.BlunderCount(ctx, scope))(t)
	}
}

// exerciseMaintenance reads and runs every maintenance line, tags analyses
// with a match equity table, and runs the once-per-library repairs of
// answered doubles over a tenant that holds a gammonNet verdict on a
// take/pass position.
func exerciseMaintenance(ctx context.Context, t *testing.T, s *Storage, scope string, ids []int64) {
	t.Helper()
	must(maintenance.Status(ctx, s, scope))(t)
	for _, kind := range maintenance.Kinds() {
		// A line answers either a list or a count; the other is refused.
		_, _ = maintenance.IDs(ctx, s, scope, kind)
		_, _ = maintenance.Count(ctx, s, scope, kind)
		if maintenance.Runnable(s, kind) == nil {
			if _, err := maintenance.RunKind(ctx, s, scope, kind, maintenance.Opts{}); err != nil {
				t.Fatalf("maintenance %s: %v", kind, err)
			}
		}
	}
	if err := s.MatchEquityTables().TagAnalyses(ctx, scope, 0, ids[:10]); err != nil {
		t.Fatal(err)
	}
	if _, err := s.pool.Exec(ctx, `UPDATE analysis a SET analysis_engine = 'gammonNet test'
		FROM position p WHERE p.tenant_id = a.tenant_id AND p.id = a.position_id
		  AND a.tenant_id = $1 AND p.decision_type = 1 AND p.cube_value > 0 AND p.cube_owner = -1`, tenantID(scope)); err != nil {
		t.Fatal(err)
	}
	n := must(sqlshared.DropGammonNetResponseAnalyses(ctx, s.binder.shared(), scope))(t)
	if n == 0 {
		t.Fatal("no gammonNet verdict on a take/pass position: the drop planned nothing")
	}
	if err := sqlshared.DropOwnedCubeAnswerMatchStats(ctx, s.binder.shared(), scope); err != nil {
		t.Fatal(err)
	}
	if _, err := reanchorTenantAnsweredDoubles(ctx, s.pool, tenantID(scope)); err != nil {
		t.Fatal(err)
	}
}

// orNotFound is must accepting storage.ErrNotFound, the answer of a card
// that has no linked half or of a deck with nothing left to draw.
func orNotFound[T any](v T, err error) func(*testing.T) T {
	if errors.Is(err, storage.ErrNotFound) {
		err = nil
	}
	return must(v, err)
}

// must fails the test on an error and returns the value: must(f())(t).
func must[T any](v T, err error) func(*testing.T) T {
	return func(t *testing.T) T {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
}

// exerciseRows reads positions and analyses the ways the stores' own methods
// do: by id, by hash, by batch and by listing, and saves a position the tenant
// already holds, which looks its row up again.
func exerciseRows(ctx context.Context, t *testing.T, s *Storage, scope string, ids []int64) {
	t.Helper()
	pos, an := s.Positions(), s.Analyses()
	p := must(pos.Load(ctx, scope, ids[0]))(t)
	must(pos.LoadByIDs(ctx, scope, ids[:5]))(t)
	must(pos.ListIDs(ctx, scope, storage.ListOpts{Limit: 5}))(t)
	must(pos.Count(ctx, scope))(t)
	if _, _, err := pos.IndexOf(ctx, scope, ids[1]); err != nil {
		t.Fatal(err)
	}
	h := uint64(engine.PopulatePositionColumns(p).ZobristHash)
	if _, ok, err := pos.Exists(ctx, scope, h); err != nil || !ok {
		t.Fatalf("exists: %v, %v", ok, err)
	}
	must(pos.ExistsMany(ctx, scope, []uint64{h}))(t)
	dup := *p
	dup.ID = 0
	if _, created, err := pos.SaveCreated(ctx, scope, &dup); err != nil || created {
		t.Fatalf("save of a held position: created=%v, %v", created, err)
	}
	if _, _, err := pos.Reinstate(ctx, scope, p); err != nil {
		t.Fatal(err)
	}
	for _, id := range ids {
		if a, err := an.Load(ctx, scope, id); err == nil && a != nil {
			break
		}
	}
	must(an.LoadMany(ctx, scope, ids[:20]))(t)
	must(an.CountPositionsWithAnalysisRow(ctx, scope))(t)
	must(an.CountWithoutAnalysis(ctx, scope))(t)
	must(an.WithoutAnalysisIDs(ctx, scope))(t)
	for _, err := range an.WithoutAnalysis(ctx, scope, storage.ListOpts{Limit: 5}) {
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range ids {
		a, err := an.Load(ctx, scope, id)
		if err != nil || a == nil {
			continue
		}
		// Rewriting a stored analysis reads its position and the move played.
		if err := an.Save(ctx, scope, id, a); err != nil {
			t.Fatal(err)
		}
		if _, err := an.Merge(ctx, scope, id, nil, func(a *domain.PositionAnalysis) *domain.PositionAnalysis { return a }); err != nil {
			t.Fatal(err)
		}
		break
	}
	for m, err := range s.Matches().List(ctx, scope, storage.MatchListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		must(an.WithoutAnalysisIDsInMatch(ctx, scope, m.ID))(t)
		break
	}
}

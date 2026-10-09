package storagetest

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// unplayedQuery selects every position no match ever met: all of them, in a
// store that holds positions saved on their own.
const unplayedQuery = "s n<1"

func saveDecisions(t *testing.T, s storage.Storage, scope string, slots ...int) []int64 {
	t.Helper()
	ids := make([]int64, 0, len(slots))
	for _, slot := range slots {
		p := statsDecisionPos(t, slot)
		id, err := s.Positions().Save(context.Background(), scope, &p)
		if err != nil {
			t.Fatalf("Save position: %v", err)
		}
		ids = append(ids, id)
	}
	return ids
}

// testCollectionEvaluateUnderCeiling pins the declared ceiling: an evaluation
// never answers more than its limit, and always says the true total and
// whether it stopped short — a truncation is never silent.
func testCollectionEvaluateUnderCeiling(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ids := saveDecisions(t, s, "", 0, 1, 2)

	living, err := storage.CreateCollection(ctx, s, "", "unplayed", "", unplayedQuery)
	if err != nil {
		t.Fatalf("CreateCollection(living): %v", err)
	}
	whole, err := storage.EvaluateCollection(ctx, s, "", living, 0)
	if err != nil {
		t.Fatalf("EvaluateCollection: %v", err)
	}
	if !whole.Living || whole.FilterQuery != unplayedQuery {
		t.Errorf("living=%v query=%q; want living with %q", whole.Living, whole.FilterQuery, unplayedQuery)
	}
	if whole.Total != 3 || whole.Truncated || whole.Cap != storage.LivingCollectionCap {
		t.Errorf("whole: total=%d truncated=%v cap=%d; want 3, false, %d", whole.Total, whole.Truncated, whole.Cap, storage.LivingCollectionCap)
	}
	got := slices.Clone(whole.PositionIDs)
	slices.Sort(got)
	if !slices.Equal(got, ids) {
		t.Errorf("ids = %v; want %v", got, ids)
	}

	cut, err := storage.EvaluateCollection(ctx, s, "", living, 2)
	if err != nil {
		t.Fatalf("EvaluateCollection(2): %v", err)
	}
	if len(cut.PositionIDs) != 2 || cut.Total != 3 || !cut.Truncated || cut.Cap != 2 {
		t.Errorf("cut: %d ids, total=%d truncated=%v cap=%d; want 2, 3, true, 2", len(cut.PositionIDs), cut.Total, cut.Truncated, cut.Cap)
	}
	if !slices.Equal(cut.PositionIDs, whole.PositionIDs[:2]) {
		t.Errorf("cut %v is not the head of %v", cut.PositionIDs, whole.PositionIDs)
	}

	// A hand-made list answers under the same contract.
	hand, err := storage.CreateCollection(ctx, s, "", "hand", "", "")
	if err != nil {
		t.Fatalf("CreateCollection(hand): %v", err)
	}
	if err := s.Collections().AddPositions(ctx, "", hand, []int64{ids[2], ids[0]}); err != nil {
		t.Fatalf("AddPositions: %v", err)
	}
	h, err := storage.EvaluateCollection(ctx, s, "", hand, 1)
	if err != nil {
		t.Fatalf("EvaluateCollection(hand): %v", err)
	}
	if h.Living || h.Total != 2 || !h.Truncated || !slices.Equal(h.PositionIDs, []int64{ids[2]}) {
		t.Errorf("hand: %+v; want not living, total 2, truncated, [%d]", h, ids[2])
	}

	if _, err := storage.EvaluateCollection(ctx, s, "", 99999, 0); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("an unknown collection must be ErrNotFound: got %v", err)
	}
}

// testCollectionUnreadableQueryRefused: a query a token of which no rule
// claims would select the whole library. It is refused when set, and a
// refused creation leaves no collection behind.
func testCollectionUnreadableQueryRefused(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	count := func() int {
		n := 0
		for _, err := range s.Collections().List(ctx, "") {
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			n++
		}
		return n
	}
	before := count()
	if _, err := storage.CreateCollection(ctx, s, "", "bad", "", "s quux"); !errors.Is(err, storage.ErrUnreadableFilter) {
		t.Fatalf("CreateCollection(unreadable): got %v, want ErrUnreadableFilter", err)
	}
	if after := count(); after != before {
		t.Errorf("a refused creation left %d collection(s) behind", after-before)
	}
	id, err := storage.CreateCollection(ctx, s, "", "good", "", "")
	if err != nil {
		t.Fatalf("CreateCollection: %v", err)
	}
	if err := storage.SetCollectionFilter(ctx, s, "", id, "s quux"); !errors.Is(err, storage.ErrUnreadableFilter) {
		t.Errorf("SetCollectionFilter(unreadable): got %v, want ErrUnreadableFilter", err)
	}
	if c, _ := s.Collections().Get(ctx, "", id); c == nil || c.FilterQuery != "" {
		t.Errorf("a refused query must not be stored: %+v", c)
	}
}

// testAnkiDeckFedByLivingCollection: a deck whose source is a living
// collection is fed by the query's result, re-evaluated at every sync, and
// the report carries the evaluation.
func testAnkiDeckFedByLivingCollection(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	saveDecisions(t, s, "", 0, 1)
	col, err := storage.CreateCollection(ctx, s, "", "unplayed", "", unplayedQuery)
	if err != nil {
		t.Fatalf("CreateCollection: %v", err)
	}
	deck, err := s.Anki().CreateDeck(ctx, "", "living deck", "", domain.AnkiSourceCollection, col, "")
	if err != nil {
		t.Fatalf("CreateDeck: %v", err)
	}
	report, err := storage.SyncDeck(ctx, s, "", deck)
	if err != nil {
		t.Fatalf("SyncDeck: %v", err)
	}
	if report.Source == nil || report.Source.Total != 2 || report.Source.Truncated {
		t.Fatalf("report source = %+v; want total 2, not truncated", report.Source)
	}
	if n, err := s.Anki().DeckPositionCount(ctx, "", deck); err != nil || n != 2 {
		t.Fatalf("DeckPositionCount = %d, %v; want 2", n, err)
	}

	saveDecisions(t, s, "", 2)
	if _, err := storage.SyncDeck(ctx, s, "", deck); err != nil {
		t.Fatalf("SyncDeck (again): %v", err)
	}
	if n, err := s.Anki().DeckPositionCount(ctx, "", deck); err != nil || n != 3 {
		t.Errorf("after a new matching position, DeckPositionCount = %d, %v; want 3", n, err)
	}

	// A hand-made collection's deck reports no evaluation: it is fed whole.
	hand, err := storage.CreateCollection(ctx, s, "", "hand", "", "")
	if err != nil {
		t.Fatalf("CreateCollection(hand): %v", err)
	}
	handDeck, err := s.Anki().CreateDeck(ctx, "", "hand deck", "", domain.AnkiSourceCollection, hand, "")
	if err != nil {
		t.Fatalf("CreateDeck(hand): %v", err)
	}
	if r, err := storage.SyncDeck(ctx, s, "", handDeck); err != nil || r.Source != nil {
		t.Errorf("SyncDeck(hand) = %+v, %v; want no source evaluation", r, err)
	}
	if _, err := storage.SyncDeck(ctx, s, "", 99999); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("an unknown deck must be ErrNotFound: got %v", err)
	}
}

// checkLivingCollectionIsolation: a living collection is evaluated by the
// search engine inside its tenant — the same query in b sees nothing of a,
// and b cannot evaluate a's collection by id.
func checkLivingCollectionIsolation(t *testing.T, ctx context.Context, s storage.Storage, a, b string) {
	saveDecisions(t, s, a, 0, 1)
	colA, err := storage.CreateCollection(ctx, s, a, "unplayed", "", unplayedQuery)
	if err != nil {
		t.Fatalf("CreateCollection(%s): %v", a, err)
	}
	if ev, err := storage.EvaluateCollection(ctx, s, a, colA, 0); err != nil || ev.Total != 2 {
		t.Fatalf("EvaluateCollection(%s) = %+v, %v; want total 2", a, ev, err)
	}
	if _, err := storage.EvaluateCollection(ctx, s, b, colA, 0); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("EvaluateCollection(%s, a's id): got %v, want ErrNotFound", b, err)
	}
	colB, err := storage.CreateCollection(ctx, s, b, "unplayed", "", unplayedQuery)
	if err != nil {
		t.Fatalf("CreateCollection(%s): %v", b, err)
	}
	if ev, err := storage.EvaluateCollection(ctx, s, b, colB, 0); err != nil || ev.Total != 0 || len(ev.PositionIDs) != 0 {
		t.Errorf("EvaluateCollection(%s) = %+v, %v; want nothing of %s", b, ev, err, a)
	}
	deckB, err := s.Anki().CreateDeck(ctx, b, "living", "", domain.AnkiSourceCollection, colB, "")
	if err != nil {
		t.Fatalf("CreateDeck(%s): %v", b, err)
	}
	if _, err := storage.SyncDeck(ctx, s, b, deckB); err != nil {
		t.Fatalf("SyncDeck(%s): %v", b, err)
	}
	if n, err := s.Anki().DeckPositionCount(ctx, b, deckB); err != nil || n != 0 {
		t.Errorf("DeckPositionCount(%s) = %d, %v; want 0", b, n, err)
	}
}

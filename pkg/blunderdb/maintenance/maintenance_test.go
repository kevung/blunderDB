package maintenance

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

func openStore(t *testing.T) *sqlite.Storage {
	t.Helper()
	st, err := sqlite.Open(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func lineOf(t *testing.T, lines []Line, kind Kind) Line {
	t.Helper()
	for _, l := range lines {
		if l.Kind == kind {
			return l
		}
	}
	t.Fatalf("no line %s", kind)
	return Line{}
}

func TestStatusCountsOnlyTheFreeLines(t *testing.T) {
	st := openStore(t)
	lines, err := Status(context.Background(), st, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != len(Kinds()) {
		t.Fatalf("%d lines, want %d", len(lines), len(Kinds()))
	}
	for _, l := range lines {
		if free := l.Tier == TierFree && l.Applicable; free != (l.Count != nil) {
			t.Errorf("%s (tier %s): count set = %v", l.Kind, l.Tier, l.Count != nil)
		}
		if l.Last != nil {
			t.Errorf("%s: a run recorded on a library where nothing ran", l.Kind)
		}
	}
	if v := lineOf(t, lines, Vacuum); !v.Applicable {
		t.Error("vacuum does not apply to SQLite")
	}
	if m := lineOf(t, lines, AnalysisMissing); m.Count != nil || !m.Listable || m.Runnable {
		t.Errorf("analysis.missing: %+v", m)
	}
}

func TestCountIsTheLengthOfIDs(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	p := domain.InitializePosition()
	p.DecisionType = domain.CheckerAction
	id, err := st.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatal(err)
	}
	n, err := Count(ctx, st, "", AnalysisMissing)
	if err != nil {
		t.Fatal(err)
	}
	ids, err := IDs(ctx, st, "", AnalysisMissing)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 || len(ids) != 1 || ids[0] != id {
		t.Errorf("count %d, ids %v, want 1 and [%d]", n, ids, id)
	}
}

func TestRefusals(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	if _, err := Count(ctx, st, "", AnalysisColumns); !errors.Is(err, storage.ErrInvalid) {
		t.Errorf("Count(analysis.columns): %v, want ErrInvalid", err)
	}
	if _, err := IDs(ctx, st, "", Trash); !errors.Is(err, storage.ErrInvalid) {
		t.Errorf("IDs(trash): %v, want ErrInvalid", err)
	}
	if _, err := RunKind(ctx, st, "", Vacuum, Opts{}); !errors.Is(err, storage.ErrInvalid) {
		t.Errorf("RunKind(vacuum): %v, want ErrInvalid", err)
	}
	if _, err := RunKind(ctx, st, "", Kind("nope"), Opts{}); !errors.Is(err, ErrUnknownKind) {
		t.Errorf("RunKind(nope): %v, want ErrUnknownKind", err)
	}
}

func TestRunIsJournaledWithItsOutcome(t *testing.T) {
	st := openStore(t)
	at := time.Unix(1_700_000_000, 0)
	now = func() time.Time { return at }
	t.Cleanup(func() { now = time.Now })

	locks := 0
	r, err := RunKind(context.Background(), st, "", StatsMatches, Opts{Before: func() func() {
		locks++
		return func() {}
	}})
	if err != nil {
		t.Fatal(err)
	}
	if r.Outcome != OutcomeDone || r.StartedAt != at.Unix() {
		t.Errorf("run %+v", r)
	}
	if locks == 0 {
		t.Error("Before never called")
	}
	lines, err := Status(context.Background(), st, "")
	if err != nil {
		t.Fatal(err)
	}
	if last := lineOf(t, lines, StatsMatches).Last; last == nil || *last != r {
		t.Errorf("journal holds %+v, want %+v", last, r)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r, err = RunKind(ctx, st, "", StatsMatches, Opts{})
	if !errors.Is(err, context.Canceled) || r.Outcome != OutcomeCancelled {
		t.Fatalf("cancelled run: %+v, %v", r, err)
	}
	lines, _ = Status(context.Background(), st, "")
	if last := lineOf(t, lines, StatsMatches).Last; last == nil || last.Outcome != OutcomeCancelled {
		t.Errorf("journal after a cancelled run: %+v", last)
	}
}

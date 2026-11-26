package gammonnet

import (
	"context"
	"fmt"
	"math"
	"slices"
	"sync/atomic"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// countingExec runs cells as LocalExec does and counts the phases it ran.
func countingExec(ply, pruneK int, phases *atomic.Int32) CellExec {
	exec := LocalExec(0, ply, pruneK)
	return func(n int, cell func(int, *Searcher)) error {
		phases.Add(1)
		return exec(n, cell)
	}
}

// refusingExec fails the test if a phase is searched.
func refusingExec(t *testing.T) CellExec {
	return func(int, func(int, *Searcher)) error {
		t.Helper()
		t.Fatal("a phase the cache holds was searched")
		return nil
	}
}

func sameRollMap(a, b RollMap) bool {
	if a.Ply != b.Ply || a.CubeAccess != b.CubeAccess || math.Float64bits(a.Average) != math.Float64bits(b.Average) || len(a.Rows) != len(b.Rows) {
		return false
	}
	for i := range a.Rows {
		x, y := a.Rows[i], b.Rows[i]
		if x.Die1 != y.Die1 || x.Die2 != y.Die2 || x.Weight != y.Weight || x.Move != y.Move || x.Dance != y.Dance ||
			math.Float64bits(x.Equity) != math.Float64bits(y.Equity) || x.Ply != y.Ply || !equalBoolPtr(x.MarketLost, y.MarketLost) {
			return false
		}
	}
	return true
}

func sameScoreMap(a, b ScoreMap) bool {
	if a.MatchLength != b.MatchLength || a.Ply != b.Ply || a.Decision != b.Decision || len(a.Cells) != len(b.Cells) {
		return false
	}
	for i := range a.Cells {
		if !sameScoreMapCell(a.Cells[i], b.Cells[i]) {
			return false
		}
	}
	return true
}

// A map asked again, and every shallower map its phases passed through, is
// served without a search and is, bit for bit, the map computed fresh.
func TestMapCache_RollMapServesEveryTierBitForBit(t *testing.T) {
	for _, c := range oracleCases {
		pos := decodeXGIDForTest(t, c.xgid)
		cache := NewMapCache(DefaultMapCacheBytes)
		first, err := cache.RollMap(context.Background(), "", pos, nil, 2, c.pruneK, LocalExec(0, 2, c.pruneK), nil)
		if err != nil {
			t.Fatal(err)
		}
		for _, ply := range []int{2, 1, 0} {
			got, err := cache.RollMap(context.Background(), "", pos, nil, ply, c.pruneK, refusingExec(t), nil)
			if err != nil {
				t.Fatal(err)
			}
			fresh, err := ComputeRollMap(context.Background(), pos, nil, ply, c.pruneK, LocalExec(0, ply, c.pruneK), nil)
			if err != nil {
				t.Fatal(err)
			}
			if !sameRollMap(got, fresh) {
				t.Fatalf("%s pruneK %d: cached %d-ply map %+v, fresh %+v", c.xgid, c.pruneK, ply, got, fresh)
			}
			if ply == 2 && !sameRollMap(got, first) {
				t.Fatalf("%s: the map served again differs from the one computed", c.xgid)
			}
		}
	}
}

// A deeper request starts from the deepest phase held: the rows arrive at
// once, then only the missing phase is searched.
func TestMapCache_RollMapResumesFromTheDeepestPhaseHeld(t *testing.T) {
	pos := decodeXGIDForTest(t, openingXGID)
	cache := NewMapCache(DefaultMapCacheBytes)
	if _, err := cache.RollMap(context.Background(), "", pos, nil, 1, 0, LocalExec(0, 1, 0), nil); err != nil {
		t.Fatal(err)
	}
	var phases atomic.Int32
	var plies []int
	got, err := cache.RollMap(context.Background(), "", pos, nil, 2, 0, countingExec(2, 0, &phases), func(_ int, r RollMapRow) {
		plies = append(plies, r.Ply)
	})
	if err != nil {
		t.Fatal(err)
	}
	if phases.Load() != 1 {
		t.Fatalf("%d phases searched, want only the 2-ply one", phases.Load())
	}
	if len(plies) != 2*NumRolls || plies[0] != 1 || plies[len(plies)-1] != 2 {
		t.Fatalf("rows arrived at plies %v, want 21 at 1 then 21 at 2", plies)
	}
	fresh, err := ComputeRollMap(context.Background(), pos, nil, 2, 0, LocalExec(0, 2, 0), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !sameRollMap(got, fresh) {
		t.Fatal("a map resumed from a held phase differs from the map computed fresh")
	}
}

// Every part of the key keeps its own entry: scope, table, PruneK, the
// position's game. Its provenance does not.
func TestMapCache_KeySeparatesWhatChangesAValue(t *testing.T) {
	pos := decodeXGIDForTest(t, busyMatchXGID)
	cache := NewMapCache(DefaultMapCacheBytes)
	ctx := context.Background()
	if _, err := cache.RollMap(ctx, "tenant-a", pos, nil, 0, 0, LocalExec(0, 0, 0), nil); err != nil {
		t.Fatal(err)
	}

	met := loadTestMET(t, "Rockwell-Kazaross.xml")
	other := pos
	other.Board.Bearoff[0]++
	other.Board.Points[6].Checkers--

	misses := map[string]func(CellExec) error{
		"another tenant": func(e CellExec) error {
			_, err := cache.RollMap(ctx, "tenant-b", pos, nil, 0, 0, e, nil)
			return err
		},
		"another table": func(e CellExec) error {
			_, err := cache.RollMap(ctx, "tenant-a", pos, met, 0, 0, e, nil)
			return err
		},
		"another PruneK": func(e CellExec) error {
			_, err := cache.RollMap(ctx, "tenant-a", pos, nil, 0, 3, e, nil)
			return err
		},
		"another position": func(e CellExec) error {
			_, err := cache.RollMap(ctx, "tenant-a", other, nil, 0, 0, e, nil)
			return err
		},
		"another Jacoby rule": func(e CellExec) error {
			p := pos
			p.HasJacoby = 1 - p.HasJacoby
			_, err := cache.RollMap(ctx, "tenant-a", p, nil, 0, 0, e, nil)
			return err
		},
	}
	for name, run := range misses {
		var phases atomic.Int32
		if err := run(countingExec(0, 0, &phases)); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if phases.Load() != 1 {
			t.Fatalf("%s was served from the cache", name)
		}
	}

	withID := pos
	withID.ID, withID.Flagged, withID.IndividuallyImported = 42, true, true
	withID.Dice = [2]int{3, 1} // a roll map ignores the dice
	if _, err := cache.RollMap(ctx, "tenant-a", withID, nil, 0, 0, refusingExec(t), nil); err != nil {
		t.Fatal(err)
	}
}

// The score map, checker and cube: every tier held is the map computed fresh.
func TestMapCache_ScoreMapServesEveryTierBitForBit(t *testing.T) {
	for _, c := range []struct {
		xgid   string
		pruneK int
	}{{scoreMapCheckerXGID, 0}, {ownedCubeMatchXGID, DefaultPruneK}} {
		pos := decodeXGIDForTest(t, c.xgid)
		const length = 3
		cache := NewMapCache(DefaultMapCacheBytes)
		if _, err := cache.ScoreMap(context.Background(), "", pos, nil, length, 2, c.pruneK, LocalExec(0, 2, c.pruneK), nil); err != nil {
			t.Fatal(err)
		}
		for _, ply := range []int{2, 1, 0} {
			got, err := cache.ScoreMap(context.Background(), "", pos, nil, length, ply, c.pruneK, refusingExec(t), nil)
			if err != nil {
				t.Fatal(err)
			}
			fresh, err := ComputeScoreMap(context.Background(), pos, nil, length, ply, c.pruneK, LocalExec(0, ply, c.pruneK), nil)
			if err != nil {
				t.Fatal(err)
			}
			if !sameScoreMap(got, fresh) {
				t.Fatalf("%s: cached %d-ply score map differs from the fresh one", c.xgid, ply)
			}
		}
		var phases atomic.Int32
		if _, err := cache.ScoreMap(context.Background(), "", pos, nil, length+1, 0, c.pruneK, countingExec(0, c.pruneK, &phases), nil); err != nil {
			t.Fatal(err)
		}
		if phases.Load() != 1 {
			t.Fatal("another match length was served from the cache")
		}
	}
}

// A phase cut short is not kept: the next call searches it.
func TestMapCache_CancelledPhaseIsNotKept(t *testing.T) {
	pos := decodeXGIDForTest(t, openingXGID)
	cache := NewMapCache(DefaultMapCacheBytes)
	ctx, cancel := context.WithCancel(context.Background())
	_, err := cache.RollMap(ctx, "", pos, nil, 1, 0, LocalExec(1, 1, 0), func(_ int, r RollMapRow) {
		if r.Ply == 1 {
			cancel()
		}
	})
	if err == nil {
		t.Fatal("the cancelled map returned no error")
	}
	if cache.Len() != 1 {
		t.Fatalf("%d phases held, want the 0-ply one only", cache.Len())
	}
}

// The budget bounds the cache: old entries go first.
func TestMapCache_BudgetEvictsTheLeastRecentlyUsed(t *testing.T) {
	pos := decodeXGIDForTest(t, openingXGID)
	one := NewMapCache(DefaultMapCacheBytes)
	if _, err := one.RollMap(context.Background(), "a", pos, nil, 0, 0, LocalExec(0, 0, 0), nil); err != nil {
		t.Fatal(err)
	}
	cache := NewMapCache(3 * one.used)
	ctx := context.Background()
	scopes := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	for _, s := range scopes {
		if _, err := cache.RollMap(ctx, s, pos, nil, 0, 0, LocalExec(0, 0, 0), nil); err != nil {
			t.Fatal(err)
		}
	}
	if cache.used > cache.budget {
		t.Fatalf("%d bytes held over a budget of %d", cache.used, cache.budget)
	}
	if n := cache.Len(); n == 0 || n == len(scopes) {
		t.Fatalf("%d entries held, want some evicted", n)
	}
	if _, ok := cache.rows(rollMapKey("a", pos, (*engine.MET)(nil).Digest(), keyRollConfig(t, pos))); ok {
		t.Fatal("the oldest entry survived")
	}
}

func keyRollConfig(t *testing.T, pos domain.Position) SearchConfig {
	t.Helper()
	job, err := newRollMapJob(pos, nil, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	return job.cfg
}

// skippingExec runs every cell but the first: a phase short of an answer.
func skippingExec(n int, cell func(int, *Searcher)) error {
	for i := 1; i < n; i++ {
		cell(i, nil)
	}
	return nil
}

// A phase short of a row or a cell returns what it has, but is not kept.
func TestMapCache_IncompletePhaseIsNotKept(t *testing.T) {
	ctx := context.Background()
	cache := NewMapCache(DefaultMapCacheBytes)
	if _, err := cache.RollMap(ctx, "", decodeXGIDForTest(t, openingXGID), nil, 0, 0, skippingExec, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := cache.ScoreMap(ctx, "", decodeXGIDForTest(t, scoreMapCheckerXGID), nil, 3, 0, 0, skippingExec, nil); err != nil {
		t.Fatal(err)
	}
	if n := cache.Len(); n != 0 {
		t.Fatalf("%d incomplete phases kept, want none", n)
	}

	stopped := ScoreMapCell{}
	stopped.Refused, stopped.Reason = true, fmt.Errorf("rank: %w", ErrCanceled).Error()
	declined := ScoreMapCell{}
	declined.Refused, declined.Reason = true, ErrNotEvaluable.Error()
	if !refusedByCancel(stopped) || refusedByCancel(declined) {
		t.Fatal("refusedByCancel does not tell a stopped search from a refusal")
	}
}

// A map handed out shares nothing with the cache: changing it changes no
// later answer.
func TestMapCache_ServedMapsAreCopies(t *testing.T) {
	ctx := context.Background()
	cache := NewMapCache(DefaultMapCacheBytes)

	rollPos := decodeXGIDForTest(t, ownedCubeMatchXGID)
	computed, err := cache.RollMap(ctx, "", rollPos, nil, 0, 0, LocalExec(0, 0, 0), nil)
	if err != nil {
		t.Fatal(err)
	}
	served, err := cache.RollMap(ctx, "", rollPos, nil, 0, 0, refusingExec(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	if served.Rows[0].MarketLost == nil {
		t.Fatal("the fixture has no market to lose")
	}
	want := *served.Rows[0].MarketLost
	*computed.Rows[0].MarketLost = !want
	*served.Rows[0].MarketLost = !want
	again, err := cache.RollMap(ctx, "", rollPos, nil, 0, 0, refusingExec(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	if *again.Rows[0].MarketLost != want {
		t.Fatal("changing a roll map handed out changed the cache")
	}

	scorePos := decodeXGIDForTest(t, scoreMapCheckerXGID)
	first, err := cache.ScoreMap(ctx, "", scorePos, nil, 3, 0, 0, LocalExec(0, 0, 0), nil)
	if err != nil {
		t.Fatal(err)
	}
	k := slices.IndexFunc(first.Cells, func(c ScoreMapCell) bool { return c.Gap != nil })
	if k < 0 {
		t.Fatal("the fixture has no cell with two plays")
	}
	gap := *first.Cells[k].Gap
	*first.Cells[k].Gap = gap + 1
	second, err := cache.ScoreMap(ctx, "", scorePos, nil, 3, 0, 0, refusingExec(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	*second.Cells[k].Gap = gap + 2
	third, err := cache.ScoreMap(ctx, "", scorePos, nil, 3, 0, 0, refusingExec(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	if math.Float64bits(*third.Cells[k].Gap) != math.Float64bits(gap) {
		t.Fatal("changing a score map handed out changed the cache")
	}
}

package storagetest

import (
	"context"
	"fmt"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testRepairCrawfordSentinelFromPastedXGID pins the half of the Crawford repair
// that has no match to ask (#360).
//
// A position no game points at carries no match to contradict its away score,
// so the match-driven pass leaves it alone. But a position that came in through
// an XGID someone else wrote — pasted before domain.DecodeXGID read field 7
// through /v1/positions.fromXGID, or a BGBlitz position file — still holds that
// XGID in its analysis, and the XGID states the rule: field 7 is 0 after the
// Crawford game. That is a fact of the source, which the repair may act on.
//
// What it may NOT act on is an XGID blunderDB wrote itself: generateXGID (the
// board saved, the position edited) and the CLI's encoder write field 7 from
// the stored sentinel, so at a stored away 1 they always write 1, and such an
// XGID proves nothing. That is also why only field 7 = 0 next to a stored 1
// counts: it is the one combination no blunderDB encoder ever produced. And the
// XGID must describe the row — same board, cube, turn, dice and distance —
// before its word is taken for it.
func testRepairCrawfordSentinelFromPastedXGID(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ps, as := s.Positions(), s.Analyses()

	// xgidOf writes the XGID a source would for p, one point short in a
	// 7-point match: points scored from the away score, field 7 as given.
	xgidOf := func(p domain.Position, flag int) string {
		owner := 0
		switch p.Cube.Owner {
		case domain.Black:
			owner = 1
		case domain.White:
			owner = -1
		}
		turn := -1
		if p.PlayerOnRoll == domain.Black {
			turn = 1
		}
		return fmt.Sprintf("XGID=%s:%d:%d:%d:%d%d:%d:%d:%d:7:10",
			domain.EncodeXGIDBoard(&p), p.Cube.Value, owner, turn, p.Dice[0], p.Dice[1],
			7-domain.PointsAway(p.Score[0]), 7-domain.PointsAway(p.Score[1]), flag)
	}
	save := func(what string, p domain.Position, xgid string) int64 {
		t.Helper()
		p.IndividuallyImported = true
		id, err := ps.Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("Save %s: %v", what, err)
		}
		if xgid != "" {
			if err := as.Save(ctx, "", id, &domain.PositionAnalysis{XGID: xgid}); err != nil {
				t.Fatalf("Save the analysis of %s: %v", what, err)
			}
		}
		return id
	}

	// Pasted after the Crawford game, stored at the ambiguous 1: repaired.
	pasted := statsDecisionPos(t, 0)
	pasted.Score = [2]int{domain.Crawford, 4}
	pastedID := save("the pasted post-Crawford position", pasted, xgidOf(pasted, 0))

	// Pasted after the Crawford game too, and its correct twin is already
	// stored: the two become one row, as in the match-driven pass.
	twin := statsDecisionPos(t, 1)
	twin.Score = [2]int{4, domain.PostCrawford}
	twinID := save("the correct twin", twin, "")
	staleTwin := statsDecisionPos(t, 1)
	staleTwin.Score = [2]int{4, domain.Crawford}
	staleTwinID := save("the stale twin", staleTwin, xgidOf(staleTwin, 0))
	if staleTwinID == twinID {
		t.Fatal("the two away scores hashed to one row; the case proves nothing")
	}

	// What must not move.
	generated := statsDecisionPos(t, 2)
	generated.Score = [2]int{domain.Crawford, 3}
	generatedID := save("a position whose XGID blunderDB wrote", generated, xgidOf(generated, 1))

	elsewhere := statsDecisionPos(t, 3)
	elsewhere.Score = [2]int{domain.Crawford, 2}
	other := elsewhere
	other.Dice = [2]int{6, 5} // an XGID of another position
	elsewhereID := save("a position whose XGID describes another one", elsewhere, xgidOf(other, 0))

	farther := statsDecisionPos(t, 4)
	farther.Score = [2]int{domain.Crawford, 5}
	fartherXGID := xgidOf(farther, 0)
	farther.Score = [2]int{domain.Crawford, 6} // the XGID says 5 away, the row 6
	fartherID := save("a position whose XGID gives another distance", farther, fartherXGID)

	bare := statsDecisionPos(t, 5)
	bare.Score = [2]int{domain.Crawford, 3}
	bareID := save("a position with no analysis", bare, "")

	n, err := ps.RepairCrawfordSentinel(ctx, "")
	if err != nil {
		t.Fatalf("RepairCrawfordSentinel: %v", err)
	}
	if n != 2 {
		t.Errorf("repaired %d positions, want 2 (the two pasted post-Crawford ones)", n)
	}

	got, err := ps.Load(ctx, "", pastedID)
	if err != nil {
		t.Fatalf("Load the repaired position: %v", err)
	}
	if got.Score != [2]int{domain.PostCrawford, 4} {
		t.Errorf("away score after repair = %v, want [0 4]", got.Score)
	}
	if !got.IndividuallyImported {
		t.Error("the repaired position lost its individually-imported mark")
	}
	if _, err := ps.Load(ctx, "", staleTwinID); err == nil {
		t.Errorf("the stale twin %d is still stored; it should have merged into %d", staleTwinID, twinID)
	}
	if a, err := as.Load(ctx, "", twinID); err != nil || a.XGID != xgidOf(staleTwin, 0) {
		t.Errorf("the survivor's analysis = %+v (err=%v), want the stale twin's pasted XGID", a, err)
	}

	for id, want := range map[int64][2]int{
		generatedID: {domain.Crawford, 3},
		elsewhereID: {domain.Crawford, 2},
		fartherID:   {domain.Crawford, 6},
		bareID:      {domain.Crawford, 3},
	} {
		p, err := ps.Load(ctx, "", id)
		if err != nil {
			t.Fatalf("Load %d: %v", id, err)
		}
		if p.Score != want {
			t.Errorf("position %d was rewritten to %v, want %v", id, p.Score, want)
		}
	}

	if again, err := ps.RepairCrawfordSentinel(ctx, ""); err != nil || again != 0 {
		t.Errorf("second pass: repaired=%d err=%v, want 0 and no error", again, err)
	}
}

package database

import (
	"math"
	"path/filepath"
	"testing"
)

// TestMatchWinTraceAgreesWithDecisionLosses locks, on real analysed matches,
// that the trace moves by exactly the loss MatchDecisionLosses gives each
// decision, toward the player who made it, and that a double's chances after
// it are the response's chances before it — the doubler's and the
// responder's sides read the same cube analysis. That last identity holds to
// the stored error's rounding (whole millipoints), the range to the
// analysis's own rounding: the trace is not clamped.
func TestMatchWinTraceAgreesWithDecisionLosses(t *testing.T) {
	t.Parallel()
	for _, file := range []string{
		"test.xg",
		"charlot1-charlot2_7p_2025-11-08-2305.xg",
		"HsbtMarseille_main_ronde4_LamourDeCaslouGildas_UngerKevin_7p.xg",
	} {
		t.Run(file, func(t *testing.T) {
			db := newTestDB(t)
			id, err := db.ImportXGMatch(filepath.Join("testdata", file))
			if err != nil {
				t.Fatalf("ImportXGMatch: %v", err)
			}
			losses, err := db.GetMatchDecisionLosses(id)
			if err != nil {
				t.Fatalf("GetMatchDecisionLosses: %v", err)
			}
			trace, err := db.GetMatchWinTrace(id)
			if err != nil {
				t.Fatalf("GetMatchWinTrace: %v", err)
			}
			if trace.Money {
				t.Fatalf("a 7-point match read as money")
			}
			if len(trace.Decisions) != len(losses) {
				t.Fatalf("%d trace points for %d decisions", len(trace.Decisions), len(losses))
			}
			var priced, pairs int
			for i, pt := range trace.Decisions {
				d := losses[i]
				if pt.MoveID != d.MoveID || pt.Player != d.Player {
					t.Fatalf("point %d is move %d (player %d), loss is move %d (player %d)", i, pt.MoveID, pt.Player, d.MoveID, d.Player)
				}
				if !pt.Analysed && (pt.MWCBefore != nil || pt.MWCAfter != nil) {
					t.Errorf("move %d: values without analysis", pt.MoveID)
				}
				for _, v := range []*float64{pt.MWCBefore, pt.MWCAfter} {
					if v != nil && (*v < -1e-3 || *v > 1+1e-3) {
						t.Errorf("move %d: chances %v outside [0,1]", pt.MoveID, *v)
					}
				}
				if d.MWCLoss == nil {
					continue
				}
				if pt.MWCBefore == nil || pt.MWCAfter == nil {
					t.Errorf("move %d: scored loss %v but no trace value", pt.MoveID, *d.MWCLoss)
					continue
				}
				priced++
				drop := *pt.MWCBefore - *pt.MWCAfter
				if d.Player == 1 {
					drop = -drop
				}
				if math.Abs(drop-*d.MWCLoss) > 1e-12 {
					t.Errorf("move %d: trace moves by %v, loss is %v", pt.MoveID, drop, *d.MWCLoss)
				}
				if i+1 < len(trace.Decisions) {
					next := trace.Decisions[i+1]
					if pt.DecisionType == "cube" && next.DecisionType == "cube" && next.Player != pt.Player &&
						next.GameNumber == pt.GameNumber && next.MoveNumber == pt.MoveNumber && next.MWCBefore != nil {
						pairs++
						if math.Abs(*next.MWCBefore-*pt.MWCAfter) > 1e-3 {
							t.Errorf("double %d leaves %v, its answer %d starts at %v", pt.MoveID, *pt.MWCAfter, next.MoveID, *next.MWCBefore)
						}
					}
				}
			}
			first := trace.Decisions[0]
			t.Logf("%s: %d points, %d priced, %d double/answer pairs, first %v", file, len(trace.Decisions), priced, pairs, deref(first.MWCBefore))
			if priced == 0 {
				t.Fatalf("no decision priced")
			}
		})
	}
}

func deref(v *float64) any {
	if v == nil {
		return nil
	}
	return *v
}

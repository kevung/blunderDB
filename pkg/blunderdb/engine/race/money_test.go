package race

import "testing"

// TestVerdictFromEquities_TooGoodIsReachable: a synthetic no-double equity
// above the cash value (DoublePass, 1.0 in this scale) must come back as
// VerdictTooGood, never folded into VerdictNoDouble (ADR-0020). The exact
// table never exercises this branch — its domain is gammonless.
func TestVerdictFromEquities_TooGoodIsReachable(t *testing.T) {
	// nd beats both the cash value (dp) and the double branch (min(dt,dp)).
	got := VerdictFromEquities(1.5, 1.8, 1.0, 0)
	if got != VerdictTooGood {
		t.Fatalf("VerdictFromEquities(1.5, 1.8, 1.0, 0) = %q, want %q", got, VerdictTooGood)
	}
}

func TestVerdictFromEquities_OtherThreeVerdicts(t *testing.T) {
	tests := []struct {
		name       string
		nd, dt, dp float64
		want       Verdict
	}{
		// Doubling loses nothing and the take is wrong: DoublePass.
		{"double_pass", 0.5, 0.6, 0.55, VerdictDoublePass},
		// Doubling loses nothing and the opponent should take: DoubleTake.
		{"double_take", 0.3, 0.5, 0.6, VerdictDoubleTake},
		// Doubling would be a mistake (ND beats both DT and DP without
		// exceeding the cash value): NoDouble.
		{"no_double", 0.5, 0.3, 0.6, VerdictNoDouble},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := VerdictFromEquities(tc.nd, tc.dt, tc.dp, 0)
			if got != tc.want {
				t.Errorf("VerdictFromEquities(%v, %v, %v, 0) = %q, want %q", tc.nd, tc.dt, tc.dp, got, tc.want)
			}
		})
	}
}

// TestMoneyFromEntry_TooGoodSynthetic exercises MoneyFromEntry itself (not
// just the shared comparison) with an Entry engineered so the centered-cube
// continuation equity exceeds the cash value — the case the exact table's
// own domain never produces (see the test below), but that any future
// gammon-aware caller of this same function must still be able to name.
func TestMoneyFromEntry_TooGoodSynthetic(t *testing.T) {
	e := Entry{WinProb: 0.95, Cubeless: 0.9, OwnedND: 1.5, CenteredND: 1.5, Against: 0.9}
	m := MoneyFromEntry(e, CubeCentered)
	if m.Verdict != VerdictTooGood {
		t.Fatalf("centered: Verdict = %q, want %q (nd=%.3f dt=%.3f dp=%.3f)", m.Verdict, VerdictTooGood, m.NoDouble, m.DoubleTake, m.DoublePass)
	}
	m = MoneyFromEntry(e, CubeOwned)
	if m.Verdict != VerdictTooGood {
		t.Fatalf("owned: Verdict = %q, want %q (nd=%.3f dt=%.3f dp=%.3f)", m.Verdict, VerdictTooGood, m.NoDouble, m.DoubleTake, m.DoublePass)
	}
}

// TestMoneyFromEntry_RealBearoffDomainNeverProducesTooGood sweeps every
// (us, them) pair of the TS-06-06 table and pins two facts:
//
//  1. MoneyFromEntry's verdict equals the plain 3-way ND/DT/DP rule on every
//     entry of the domain.
//  2. VerdictTooGood never fires: the no-double equity never exceeds the cash
//     value (1.0) in this gammonless domain.

func TestMoneyFromEntry_RealBearoffDomainNeverProducesTooGood(t *testing.T) {
	ts := testTwoSided(t)
	n := ts.Checkers()
	var boards [][6]int
	var gen func(remaining, idx int, cur [6]int)
	gen = func(remaining, idx int, cur [6]int) {
		if idx == 6 {
			boards = append(boards, cur)
			return
		}
		for v := 0; v <= remaining; v++ {
			next := cur
			next[idx] = v
			gen(remaining-v, idx+1, next)
		}
	}
	gen(n, 0, [6]int{})

	// The old rule, verbatim, kept only here as the regression's oracle.
	oldRule := func(nd, dt, dp float64) Verdict {
		double := dt >= nd-moneyEps && dp >= nd-moneyEps
		if !double {
			return VerdictNoDouble
		} else if dt < dp-moneyEps {
			return VerdictDoubleTake
		}
		return VerdictDoublePass
	}

	total, tooGood, changed := 0, 0, 0
	for _, us := range boards {
		for _, them := range boards {
			e, err := ts.Lookup(us, them)
			if err != nil {
				t.Fatal(err)
			}
			for _, st := range []CubeState{CubeCentered, CubeOwned} {
				m := MoneyFromEntry(e, st)
				total++
				if m.Verdict == VerdictTooGood {
					tooGood++
				}
				if m.Verdict != oldRule(m.NoDouble, m.DoubleTake, m.DoublePass) {
					changed++
				}
			}
		}
	}
	if total < 1_000_000 {
		t.Fatalf("suspiciously few entries swept: %d", total)
	}
	if tooGood != 0 {
		t.Errorf("VerdictTooGood fired %d/%d times on the gammonless exact table — domain assumption above is stale, update the doc comment", tooGood, total)
	}
	if changed != 0 {
		t.Errorf("%d/%d verdicts differ from the old inline 3-way rule — #193's routing through VerdictFromEquities is not the no-op this test pins", changed, total)
	}
}

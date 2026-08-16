package engine

import (
	"math"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// dca builds a DoublingCubeAnalysis with the equity/error fields used by the
// cube-error attribution. Errors are equity-vs-best (≤ 0), equities are the
// cubeful equities of each line from the doubler's perspective.
func dca() *domain.DoublingCubeAnalysis {
	return &domain.DoublingCubeAnalysis{
		CubefulNoDoubleError:    -0.030,
		CubefulDoubleTakeError:  -0.120,
		CubefulDoublePassError:  -0.250,
		CubefulDoubleTakeEquity: 0.640,
		CubefulDoublePassEquity: 0.510,
	}
}

func approx(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// TestCubeActionError_DoublingDecisions is the regression guard for the bug
// where a doubler's action was either dropped ("Double") or scored with the
// opponent's response error ("Double/Pass" → pass error). All doubling
// decisions must use min(DoubleTakeError, DoublePassError).
func TestCubeActionError_DoublingDecisions(t *testing.T) {
	d := dca()
	want := math.Min(d.CubefulDoubleTakeError, d.CubefulDoublePassError) // -0.250
	for _, action := range []string{"Double", "Double/Take", "Double/Pass", "DoubleTake", "Redouble", "double"} {
		got, ok := CubeActionError(d, action)
		if !ok {
			t.Fatalf("CubeActionError(%q) ok=false, want true (doubling decisions must be retained)", action)
		}
		if !approx(got, want) {
			t.Errorf("CubeActionError(%q) = %v, want %v", action, got, want)
		}
	}
}

// TestCubeActionError_NoDouble covers every spelling of "the cube stayed put".
//
// "Double No" is not a typo: it is what the XG importer writes into
// PlayedCubeActions for a no-double, alongside "No Double" — 194 occurrences in
// a 151-match tournament corpus. Normalised, it reads "doubleno", which does NOT
// contain "nodouble"; it therefore used to fall through to the doubling branch
// and be scored with min(DoubleTakeError, DoublePassError) — the error of the
// double that never happened. See kevung/blunderDB#115.
func TestCubeActionError_NoDouble(t *testing.T) {
	d := dca()
	for _, action := range []string{"No Double", "NoDouble", "nd", "ND", "Double No", "double no"} {
		got, ok := CubeActionError(d, action)
		if !ok || !approx(got, d.CubefulNoDoubleError) {
			t.Errorf("CubeActionError(%q) = (%v,%v), want (%v,true)", action, got, ok, d.CubefulNoDoubleError)
		}
	}
}

func TestCubeActionError_Responses(t *testing.T) {
	d := dca()
	minEq := math.Min(d.CubefulDoubleTakeEquity, d.CubefulDoublePassEquity) // 0.510
	wantTake := minEq - d.CubefulDoubleTakeEquity                           // 0.510 - 0.640
	wantPass := minEq - d.CubefulDoublePassEquity                           // 0.510 - 0.510 = 0

	for _, action := range []string{"Take", "take", "dt"} {
		got, ok := CubeActionError(d, action)
		if !ok || !approx(got, wantTake) {
			t.Errorf("CubeActionError(%q) = (%v,%v), want (%v,true)", action, got, ok, wantTake)
		}
	}
	for _, action := range []string{"Pass", "Drop", "dp"} {
		got, ok := CubeActionError(d, action)
		if !ok || !approx(got, wantPass) {
			t.Errorf("CubeActionError(%q) = (%v,%v), want (%v,true)", action, got, ok, wantPass)
		}
	}
}

// TestCanonicalCubeAction_ImporterLabels pins the exact labels the XG importer
// writes, as counted over a 151-match tournament corpus (57 820 positions):
//
//	No Double 19286 · Double/Take 508 · Take 508 · Double/Pass 411 · Pass 411 · Double No 194
//
// Six spellings for four actions, two of them for the same one. This test is
// the record of what the data actually contains: if a new label appears, it
// belongs here with its count, not in a new strings.Contains somewhere.
func TestCanonicalCubeAction_ImporterLabels(t *testing.T) {
	for _, tc := range []struct{ label, want string }{
		{"No Double", CubeNoDouble},
		{"Double No", CubeNoDouble}, // 194 occurrences — the one that was misread
		{"Double/Take", CubeDouble}, // the DOUBLER's action, not the response
		{"Double/Pass", CubeDouble},
		{"Take", CubeTake},
		{"Pass", CubePass},
		// Abbreviations and spacing variants met in move.cube_action and filters.
		{"nd", CubeNoDouble}, {"dt", CubeTake}, {"dp", CubePass},
		{"  no  double ", CubeNoDouble}, {"NO-DOUBLE", CubeNoDouble}, {"Redouble", CubeDouble},
		{"Drop", CubePass}, {"", CubeUnknown}, {"   ", CubeUnknown}, {"garbage", CubeUnknown},
	} {
		if got := CanonicalCubeAction(tc.label); got != tc.want {
			t.Errorf("CanonicalCubeAction(%q) = %q, want %q", tc.label, got, tc.want)
		}
	}
}

// TestBestCubeVerdict covers the OTHER side of the label problem: a single
// bestCubeAction states TWO rulings at once — whether the cube should have been
// offered, and how it should have been answered. Reading it as one action loses
// half of it: "Double, Take" and "Double, Pass" agree on the offer and disagree
// on the answer, while "No Double" rules on the offer and still implies that
// taking is right if the cube comes anyway.
func TestBestCubeVerdict(t *testing.T) {
	for _, tc := range []struct {
		best       string
		wantDouble bool
		wantPass   bool
		wantOK     bool
	}{
		{"No Double", false, false, true},
		{"Double, Take", true, false, true},
		{"Double, Pass", true, true, true},
		// "Too good" contains "double" but rules AGAINST doubling: the player is
		// too strong to cash and should play on for the gammon.
		{"Too good to double, take", false, false, true},
		{"Too good to double, pass", false, true, true},
		{"", false, false, false},
		{"garbage", false, false, false},
	} {
		got, ok := BestCubeVerdict(tc.best)
		if ok != tc.wantOK {
			t.Errorf("BestCubeVerdict(%q) ok = %v, want %v", tc.best, ok, tc.wantOK)
			continue
		}
		if ok && (got.ShouldDouble != tc.wantDouble || got.ShouldPass != tc.wantPass) {
			t.Errorf("BestCubeVerdict(%q) = {double:%v pass:%v}, want {double:%v pass:%v}",
				tc.best, got.ShouldDouble, got.ShouldPass, tc.wantDouble, tc.wantPass)
		}
	}
}

func TestCubeActionError_Unrecognized(t *testing.T) {
	d := dca()
	for _, action := range []string{"", "   ", "garbage"} {
		if _, ok := CubeActionError(d, action); ok {
			t.Errorf("CubeActionError(%q) ok=true, want false", action)
		}
	}
	if _, ok := CubeActionError(nil, "Double"); ok {
		t.Error("CubeActionError(nil, ...) ok=true, want false")
	}
}

func TestIsResponseCubeAction(t *testing.T) {
	responses := []string{"Take", "Pass", "take", "Drop", "dt", "dp"}
	for _, a := range responses {
		if !IsResponseCubeAction(a) {
			t.Errorf("IsResponseCubeAction(%q) = false, want true", a)
		}
	}
	// Doubling decisions (incl. combined) and no-double are NOT responses.
	nonResponses := []string{"Double", "Double/Take", "Double/Pass", "No Double", "NoDouble", "Redouble", ""}
	for _, a := range nonResponses {
		if IsResponseCubeAction(a) {
			t.Errorf("IsResponseCubeAction(%q) = true, want false", a)
		}
	}
}

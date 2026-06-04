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

func TestCubeActionError_NoDouble(t *testing.T) {
	d := dca()
	for _, action := range []string{"No Double", "NoDouble", "nd", "ND"} {
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

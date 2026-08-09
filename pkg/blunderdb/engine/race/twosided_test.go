package race

import (
	"encoding/json"
	"math"
	"os"
	"strings"
	"testing"
)

// moneyFixture mirrors testdata/money_fixtures.json, generated with gnubg
// 1.08 (cfevaluate, 0-ply cubeful, money, jacoby off) over the embedded
// TS-06-06 domain. It pins both the plane semantics and the verdict rule.
type moneyFixture struct {
	Us     []int          `json:"us"`
	Them   []int          `json:"them"`
	Planes []float64      `json:"planes"`
	States map[string]fix `json:"states"`
}

type fix struct {
	Optimal  float64 `json:"optimal"`
	NoDouble float64 `json:"nodouble"`
	Take     float64 `json:"take"`
	Drop     float64 `json:"drop"`
	Decision string  `json:"decision"`
	P        float64 `json:"p"`
}

func loadFixtures(t *testing.T) []moneyFixture {
	t.Helper()
	raw, err := os.ReadFile("testdata/money_fixtures.json")
	if err != nil {
		t.Fatal(err)
	}
	var fs []moneyFixture
	if err := json.Unmarshal(raw, &fs); err != nil {
		t.Fatal(err)
	}
	if len(fs) < 50 {
		t.Fatalf("suspiciously few fixtures: %d", len(fs))
	}
	return fs
}

func board6(xs []int) (b [6]int) {
	copy(b[:], xs)
	return
}

// gnubg decision strings → our verdict space.
func parseDecision(dec string) (double, take bool) {
	d := strings.ToLower(dec)
	double = strings.Contains(d, "double") &&
		!strings.Contains(d, "no double") && !strings.Contains(d, "no redouble") &&
		!strings.Contains(d, "never")
	take = strings.Contains(d, "take")
	return
}

func TestTwoSided_PlanesMatchGnubg(t *testing.T) {
	ts := EmbeddedTwoSided()
	if ts.Checkers() != 6 {
		t.Fatalf("embedded database is TS-06-%02d, want TS-06-06", ts.Checkers())
	}
	for _, f := range loadFixtures(t) {
		e, err := ts.Lookup(board6(f.Us), board6(f.Them))
		if err != nil {
			t.Fatal(err)
		}
		for i, got := range []float64{e.Cubeless, e.OwnedND, e.CenteredND, e.Against} {
			if math.Abs(got-f.Planes[i]) > 1e-9 {
				t.Fatalf("us=%v them=%v plane %d: got %v want %v", f.Us, f.Them, i, got, f.Planes[i])
			}
		}
		// gnubg's cubeless win probability equals plane 0.
		if st, ok := f.States["centered"]; ok {
			if math.Abs(e.WinProb-st.P) > 1e-6 {
				t.Fatalf("us=%v them=%v p: got %v want %v", f.Us, f.Them, e.WinProb, st.P)
			}
		}
		// Continuation equities: plane 1/2/3 == gnubg NODOUBLE per cube state.
		if math.Abs(e.OwnedND-f.States["owned"].NoDouble) > 1e-6 ||
			math.Abs(e.CenteredND-f.States["centered"].NoDouble) > 1e-6 ||
			math.Abs(e.Against-f.States["against"].NoDouble) > 1e-6 {
			t.Fatalf("us=%v them=%v: continuation equities disagree with gnubg", f.Us, f.Them)
		}
	}
}

func TestMoney_VerdictsMatchGnubg(t *testing.T) {
	ts := EmbeddedTwoSided()
	checked := 0
	for _, f := range loadFixtures(t) {
		e, err := ts.Lookup(board6(f.Us), board6(f.Them))
		if err != nil {
			t.Fatal(err)
		}
		for _, tc := range []struct {
			label string
			state CubeState
		}{{"centered", CubeCentered}, {"owned", CubeOwned}} {
			st := f.States[tc.label]
			m := MoneyFromEntry(e, tc.state)
			if math.Abs(m.DoubleTake-st.Take) > 1e-6 || math.Abs(m.DoublePass-st.Drop) > 1e-6 {
				t.Fatalf("us=%v them=%v %s: DT/DP got %v/%v want %v/%v",
					f.Us, f.Them, tc.label, m.DoubleTake, m.DoublePass, st.Take, st.Drop)
			}
			wantDouble, wantTake := parseDecision(st.Decision)
			gotDouble := m.Verdict == VerdictDoubleTake || m.Verdict == VerdictDoublePass
			gotTake := m.Verdict == VerdictDoubleTake
			if gotDouble != wantDouble || (wantDouble && gotTake != wantTake) {
				t.Fatalf("us=%v them=%v %s: verdict %q vs gnubg %q (ND %.4f DT %.4f)",
					f.Us, f.Them, tc.label, m.Verdict, st.Decision, m.NoDouble, m.DoubleTake)
			}
			checked++
		}
		// Cube against: no verdict, continuation equity = plane 3.
		m := MoneyFromEntry(e, CubeAgainst)
		if m.Verdict != "" || math.Abs(m.NoDouble-e.Against) > 1e-12 {
			t.Fatalf("us=%v them=%v against: verdict %q nd %v", f.Us, f.Them, m.Verdict, m.NoDouble)
		}
	}
	if checked < 100 {
		t.Fatalf("only %d verdicts checked", checked)
	}
}

func TestTwoSided_RejectsGarbage(t *testing.T) {
	dir := t.TempDir()
	bad := dir + "/bad.bd"
	if err := os.WriteFile(bad, []byte("this is not a bearoff database, not even close!!"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenTwoSided(bad); err == nil {
		t.Fatal("garbage file must be rejected")
	}
	// One-sided header must be rejected too.
	os1 := dir + "/os.bd"
	hdr := make([]byte, 60)
	copy(hdr, []byte("gnubg-OS-06-15-1-1-0"))
	if err := os.WriteFile(os1, hdr, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenTwoSided(os1); err == nil {
		t.Fatal("one-sided database must be rejected by the two-sided reader")
	}
}

package domain

import (
	"math"
	"testing"
)

func near(a, b, tol float64) bool { return math.Abs(a-b) <= tol }

func TestMatchMWC7_Scaling(t *testing.T) {
	cases := []struct {
		loss float64
		n    int
		want float64
	}{
		{0, 7, 0},
		{0.3003, 7, 0.3003}, // a 7-point match reads its own loss
		{0.1, 1, 0.1 * math.Sqrt(7)},
		{0.2, 25, 0.2 * math.Sqrt(7.0/25)},
	}
	for _, c := range cases {
		e := MatchMWC7(c.loss, c.n, nil)
		if !e.Available || !near(e.Loss, c.want, 1e-12) || e.HasInterval {
			t.Errorf("MatchMWC7(%v, %d) = %+v, want loss %v", c.loss, c.n, e, c.want)
		}
	}
	for _, n := range []int{0, -1, 65, 99999} {
		if e := MatchMWC7(0.1, n, []float64{0.05, 0.05}); e.Available {
			t.Errorf("length %d (money): available", n)
		}
	}
}

func TestMWC7Elo_KnownValues(t *testing.T) {
	// q = 0.4 over seven points: (2000/√7)·log10(0.4/0.6) ≈ −133.1.
	if d, _ := MWC7Elo(0.1); !near(d, 2000/math.Sqrt(7)*math.Log10(0.4/0.6), 1e-9) || !near(d, -133.1, 0.1) {
		t.Errorf("MWC7Elo(0.1) = %v", d)
	}
	if d, _ := MWC7Elo(0); d != 0 {
		t.Errorf("MWC7Elo(0) = %v", d)
	}
	if d, _ := MWC7Elo(-0.01); d != 0 {
		t.Errorf("negative loss gives %v", d)
	}
}

// For a small loss the Elo reading is the linearisation D ≈ −3474·L/√N.
func TestMWC7Elo_Linearisation(t *testing.T) {
	for _, n := range []int{1, 3, 7, 11, 25} {
		for _, l := range []float64{0.001, 0.005, 0.01} {
			d := MatchMWC7(l, n, nil).Elo
			lin := -3474 * l / math.Sqrt(float64(n))
			if !near(d, lin, 0.005*math.Abs(lin)+0.05) {
				t.Errorf("L=%v N=%d: %v, linear %v", l, n, d, lin)
			}
		}
	}
}

func TestMWC7_Monotone(t *testing.T) {
	prevL, prevD := -1.0, 1.0
	for l := 0.0; l < 0.49; l += 0.01 {
		e := MatchMWC7(l, 7, nil)
		if e.Loss <= prevL || (l > 0 && e.Elo >= prevD) {
			t.Fatalf("not monotone in L at %v: %+v", l, e)
		}
		prevL, prevD = e.Loss, e.Elo
	}
	// The same loss over a longer match is a smaller 7-point loss.
	prevL = math.Inf(1)
	for n := 1; n <= 25; n++ {
		e := MatchMWC7(0.2, n, nil)
		if e.Loss >= prevL {
			t.Fatalf("not decreasing in N at %d", n)
		}
		prevL = e.Loss
	}
}

func TestMWC7Elo_Floor(t *testing.T) {
	want := 2000 / math.Sqrt(7) * math.Log10(0.01/0.99)
	for _, l := range []float64{0.495, 0.5, 0.512, 0.9, 3} {
		e := MatchMWC7(l, 7, []float64{l / 2, l / 2, 0})
		if math.IsNaN(e.Elo) || math.IsInf(e.Elo, 0) || math.IsNaN(e.EloLow) || math.IsInf(e.EloLow, 0) {
			t.Fatalf("L=%v gives %+v", l, e)
		}
		if !e.EloFloored || !near(e.Elo, want, 1e-9) || e.Loss != l {
			t.Errorf("L=%v: %+v, want the loss kept and the Elo floored at %v", l, e, want)
		}
	}
}

func TestMatchMWC7_Interval(t *testing.T) {
	if e := MatchMWC7(0.1, 7, []float64{0.1}); e.HasInterval {
		t.Errorf("one game: %+v, want no interval", e)
	}
	e := MatchMWC7(0.2, 7, []float64{0.02, 0.1, 0.03, 0.05})
	if !e.HasInterval || !(e.Low < e.Loss && e.Loss < e.High) || !(e.EloLow < e.Elo && e.Elo < e.EloHigh && e.EloHigh <= 0) {
		t.Fatalf("interval %+v does not bracket the value", e)
	}
	// Variance of a sum of 4 games: 4/3·Σ(l − l̄)².
	half := 1.96 * math.Sqrt(4.0/3*(0.03*0.03+0.05*0.05+0.02*0.02+0))
	if !near(e.High-e.Loss, half, 1e-12) {
		t.Errorf("half width %v, want %v", e.High-e.Loss, half)
	}
	if e.Low < 0 {
		t.Errorf("low bound under zero: %v", e.Low)
	}
	// Equal games have nothing to resample: the interval is the point.
	e = MatchMWC7(0.2, 7, []float64{0.05, 0.05, 0.05, 0.05})
	if e.Low != e.Loss || e.High != e.Loss {
		t.Errorf("equal games: %+v", e)
	}
}

func TestMWC7Pool(t *testing.T) {
	var empty MWC7Pool
	if empty.Result().Available {
		t.Error("empty pool available")
	}
	// A pool of one unit is that unit, interval included.
	one := MatchMWC7(0.12, 5, []float64{0.02, 0.1})
	var p MWC7Pool
	p.Add(one, 5)
	p.AddLoss(0.1, 0) // money: ignored
	if got := p.Result(); got != one {
		t.Errorf("pool of one = %+v, want %+v", got, one)
	}

	// The pool is √7·ΣL/Σ√N, not a mean of the units' ratios.
	var q MWC7Pool
	q.AddLoss(0.30, 7)
	q.AddLoss(0.05, 1)
	q.AddLoss(0.40, 11)
	r := q.Result()
	want := math.Sqrt(7) * (0.30 + 0.05 + 0.40) / (math.Sqrt(7) + 1 + math.Sqrt(11))
	if !near(r.Loss, want, 1e-12) {
		t.Errorf("pooled %v, want %v", r.Loss, want)
	}
	if d, _ := MWC7Elo(want); r.Elo != d {
		t.Errorf("pooled Elo %v, want %v", r.Elo, d)
	}
	if !r.HasInterval || r.Low > r.Loss || r.High < r.Loss || r.Matches != 3 {
		t.Errorf("pooled %+v", r)
	}

	// A unit past half a match keeps every figure finite.
	q.AddLoss(0.9, 7)
	r = q.Result()
	for _, v := range []float64{r.Loss, r.Low, r.High, r.Elo, r.EloLow, r.EloHigh} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			t.Fatalf("pool with a huge loss: %+v", r)
		}
	}
}

func TestMWC7PoolAddMatchIsOneUnit(t *testing.T) {
	var p MWC7Pool
	p.AddMatch(0.3, 2, 7)
	if r := p.Result(); !r.Available || r.HasInterval || r.Matches != 2 || !near(r.Loss, 0.15, 1e-12) {
		t.Errorf("one match, two seats: %+v", r)
	}
	// A seat alone is AddLoss.
	var a, b MWC7Pool
	a.AddMatch(0.2, 1, 5)
	b.AddLoss(0.2, 5)
	if a.Result() != b.Result() {
		t.Errorf("AddMatch of one seat %+v, AddLoss %+v", a.Result(), b.Result())
	}
}

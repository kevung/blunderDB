package domain

import "testing"

func TestSuggestRetention(t *testing.T) {
	const small = AnkiOptimizeMinSample - 1
	const big = AnkiOptimizeMinSample

	cases := []struct {
		name     string
		current  float64
		observed float64
		sample   int
		want     float64
	}{
		{"below sample threshold keeps current", 0.90, 0.50, small, 0.90},
		{"under-retention raises (clamped)", 0.90, 0.80, big, 0.97}, // 0.90 + 0.10 = 1.00 → clamp 0.97
		{"over-retention lowers", 0.90, 0.95, big, 0.85},            // 0.90 + (0.90-0.95) = 0.85
		{"on target keeps current", 0.90, 0.90, big, 0.90},
		{"floor clamp", 0.82, 0.99, big, 0.80}, // 0.82 - 0.17 = 0.65 → clamp 0.80
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := SuggestRetention(c.current, c.observed, c.sample)
			if diff := got - c.want; diff > 1e-9 || diff < -1e-9 {
				t.Errorf("SuggestRetention(%v,%v,%d) = %v, want %v", c.current, c.observed, c.sample, got, c.want)
			}
		})
	}
}

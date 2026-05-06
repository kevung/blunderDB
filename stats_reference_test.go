package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestStatsReferenceJSON verifies that all JSON files in testdata/stats_reference/
// are valid and parseable, and that each file's sanity checks pass.
func TestStatsReferenceJSON(t *testing.T) {
	files, err := filepath.Glob("testdata/stats_reference/*.json")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no JSON files found in testdata/stats_reference/")
	}

	for _, path := range files {
		t.Run(filepath.Base(path), func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			var doc map[string]any
			if err := json.Unmarshal(data, &doc); err != nil {
				t.Fatalf("json.Unmarshal: %v", err)
			}

			// Sanity-check XG section when present
			if xg, ok := doc["xg"].(map[string]any); ok {
				for _, pKey := range []string{"player1", "player2"} {
					p, ok := xg[pKey].(map[string]any)
					if !ok {
						continue
					}
					checkXGPlayerSanity(t, pKey, p)
				}
			}
		})
	}
}

// checkXGPlayerSanity verifies that the XG PR is consistent with the equity error
// and decision count: PR ≈ 500 × |equity_error_emg| / total_decisions (±0.05).
func checkXGPlayerSanity(t *testing.T, player string, p map[string]any) {
	t.Helper()
	pr, hasPR := p["pr"].(float64)
	eq, hasEQ := p["total_equity_error_emg"].(float64)
	dec, hasDec := p["total_decisions"].(float64)
	if !hasPR || !hasEQ || !hasDec || dec == 0 {
		return
	}
	computed := 500 * eq / dec
	diff := pr - computed
	if diff < 0 {
		diff = -diff
	}
	if diff > 0.05 {
		t.Errorf("%s PR sanity: got pr=%.3f but 500×%.3f/%.0f=%.3f (diff=%.3f, want ≤0.05)",
			player, pr, eq, dec, computed, diff)
	}
}

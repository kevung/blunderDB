package engine

import (
	"encoding/json"
	"os"
	"testing"
)

// The frontend reads the same file (cubeAction.test.js): one list of the labels the importers
// write, two tables that must agree on it.
func TestCanonicalCubeActionOnSharedLabels(t *testing.T) {
	raw, err := os.ReadFile("../../../testdata/cube_action_labels.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct{ Label, Canonical string }
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		if got := CanonicalCubeAction(c.Label); got != c.Canonical {
			t.Errorf("CanonicalCubeAction(%q) = %q, want %q", c.Label, got, c.Canonical)
		}
	}
}

package openapigen

import (
	"os"
	"regexp"
	"sort"
	"testing"
)

// TestHeadlessPageNamesOnlyRealRoutes locks the hand-written server page,
// doc/source/mode_headless.rst, to the route table: every `/v1/<family>.<op>`
// or `/ops/<family>.<op>` the prose cites must be a route the daemon serves.
// The generated annex (api_reference.rst) cannot drift; the prose can, and
// did — two routes stayed written `/v1/` for a release after moving under
// `/ops/` (tasks/critique-doc-2026-09, persona 4 #1 and persona 7 #3).
func TestHeadlessPageNamesOnlyRealRoutes(t *testing.T) {
	model, err := Parse("internal/server")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	served := map[string]bool{}
	for _, r := range model.Routes {
		served[r.Pattern] = true
	}

	src, err := os.ReadFile("doc/source/mode_headless.rst")
	if err != nil {
		t.Fatalf("reading doc/source/mode_headless.rst: %v", err)
	}
	cited := regexp.MustCompile(`/(?:v1|ops)/[A-Za-z]+\.[A-Za-z]+`).FindAllString(string(src), -1)

	seen := map[string]bool{}
	var missing []string
	for _, route := range cited {
		if seen[route] {
			continue
		}
		seen[route] = true
		if !served[route] {
			missing = append(missing, route)
		}
	}
	sort.Strings(missing)
	for _, route := range missing {
		t.Errorf("doc/source/mode_headless.rst cites %s, which the daemon does not serve (moved, renamed, or removed? api_reference.rst has the real list)", route)
	}
}

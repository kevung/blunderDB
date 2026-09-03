package openapigen

import (
	"context"
	"sort"
	"testing"

	internalserver "github.com/kevung/blunderdb/internal/server"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// realServerPaths builds a real *internal/server.Server exactly as
// production code would and returns its sorted /v1 route patterns — the
// runtime ground truth this package's static parser must agree with.
func realServerPaths(t *testing.T) []string {
	t.Helper()
	st, err := sqlite.Open(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatalf("sqlite.Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	srv, err := internalserver.New(internalserver.Options{Storage: st})
	if err != nil {
		t.Fatalf("internalserver.New: %v", err)
	}
	return srv.Paths()
}

// TestParse_MatchesRealServerRoutes is openapigen's own non-drift guard,
// independent of the committed openapi.yaml/api_reference.rst: it compares
// the AST-derived route set directly against the real, running Server's
// Paths() — the same table the CLI's `call --list` walks. A route this
// static parser cannot recognise (a new hand-written shape, a route table
// built a fourth way) fails HERE, not silently producing a stale-but-still-
// "successful" generation.
func TestParse_MatchesRealServerRoutes(t *testing.T) {
	want := realServerPaths(t)

	model, err := Parse("internal/server")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	var got []string
	for _, r := range model.Routes {
		got = append(got, r.Pattern)
	}
	sort.Strings(got)

	if len(got) != len(want) {
		t.Fatalf("parsed %d routes, server has %d — see the diff below", len(got), len(want))
	}
	wantSet := make(map[string]bool, len(want))
	for _, p := range want {
		wantSet[p] = true
	}
	gotSet := make(map[string]bool, len(got))
	for _, p := range got {
		gotSet[p] = true
	}
	for _, p := range got {
		if !wantSet[p] {
			t.Errorf("parsed a route the real server does not have: %s", p)
		}
	}
	for _, p := range want {
		if !gotSet[p] {
			t.Errorf("real server has a route the parser missed: %s", p)
		}
	}
}

// TestParse_EveryRouteHasAKnownKind guards against a silent classification
// gap: every parsed route must be one of the three recognised kinds.
func TestParse_EveryRouteHasAKnownKind(t *testing.T) {
	model, err := Parse("internal/server")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	for _, r := range model.Routes {
		switch r.Kind {
		case kindJSON, kindStream, kindCustom:
		default:
			t.Errorf("route %s has an unrecognised kind %q", r.Pattern, r.Kind)
		}
	}
}

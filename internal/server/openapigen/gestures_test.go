package openapigen

import (
	"context"
	"strings"
	"testing"

	internalserver "github.com/kevung/blunderdb/internal/server"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// directionOnlyPaths are the routes --direction adds: the server's own answer, so a gesture the
// parser mistakes for something else cannot also drop out of the check.
func directionOnlyPaths(t *testing.T) []string {
	t.Helper()
	paths := func(on bool) []string {
		st, err := sqlite.Open(context.Background(), ":memory:", nil)
		if err != nil {
			t.Fatalf("sqlite.Open: %v", err)
		}
		t.Cleanup(func() { st.Close() })
		srv, err := internalserver.New(internalserver.Options{Storage: st, EnableDirection: on})
		if err != nil {
			t.Fatalf("internalserver.New: %v", err)
		}
		return srv.Paths()
	}
	off := map[string]bool{}
	for _, p := range paths(false) {
		off[p] = true
	}
	var only []string
	for _, p := range paths(true) {
		if !off[p] {
			only = append(only, p)
		}
	}
	return only
}

// pathItem cuts the openapi.yaml item of one pattern.
func pathItem(doc, pattern string) string {
	i := strings.Index(doc, "\n  "+pattern+":\n")
	if i < 0 {
		return ""
	}
	rest := doc[i+1:]
	if j := strings.Index(rest[1:], "\n  /"); j >= 0 {
		return rest[:j+1]
	}
	return rest
}

// TestDirectionGestures_DeclareTheirContract: every versioned gesture of a Direction or a
// Rencontre states, in the contract, the If-Match it requires, the 409 and 428 it answers, the
// Direction-Version it leaves and the Idempotency-Key it accepts — and the Python client sends
// the version. The two creates name nothing that has a version yet; previewConfig writes nothing.
func TestDirectionGestures_DeclareTheirContract(t *testing.T) {
	exempt := map[string]bool{
		"/v1/directions.create": true,
		"/v1/rencontres.create": true,
	}
	model, err := Parse("internal/server")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	doc := GenerateOpenAPI(model)
	py := GeneratePythonClient(model)
	gestures := directionOnlyPaths(t)
	if len(gestures) < 20 {
		t.Fatalf("only %d routes behind --direction: the probe is broken", len(gestures))
	}
	for _, p := range gestures {
		if p == "/v1/directions.previewConfig" {
			continue
		}
		item := pathItem(doc, p)
		if item == "" {
			t.Errorf("%s: no path item", p)
			continue
		}
		if strings.Contains(item, "x-kind: custom") {
			t.Errorf("%s: classified custom, so without its schema", p)
		}
		if !strings.Contains(item, "name: Idempotency-Key") {
			t.Errorf("%s: no Idempotency-Key header declared", p)
		}
		if exempt[p] {
			continue
		}
		for _, want := range []string{"name: If-Match", `"409":`, `"428":`, "Direction-Version:"} {
			if !strings.Contains(item, want) {
				t.Errorf("%s: missing %s", p, want)
			}
		}
		if !strings.Contains(py, `"`+p+`", payload, if_match=if_match`) {
			t.Errorf("%s: the Python client does not send If-Match", p)
		}
	}
}

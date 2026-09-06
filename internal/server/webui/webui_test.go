package webui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestHandlerServesThePage pins what a stale or broken build would break
// first: the page is there, and its assets are reachable from it (#295).
func TestHandlerServesThePage(t *testing.T) {
	h := Handler()

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, Prefix, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s: got %d, want 200", Prefix, rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "<div id=\"app\">") {
		t.Errorf("the page does not look like the built front: %.120s", body)
	}
	// The page must not be cached while its hashed assets may be: getting this
	// backwards is how a user runs last week's page against this week's
	// daemon.
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control: got %q, want no-store", got)
	}

	// Every asset the page names must actually be embedded — the one failure
	// a half-committed dist produces.
	for _, ref := range assetRefs(body) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, Prefix+ref, nil))
		if rec.Code != http.StatusOK {
			t.Errorf("asset %q: got %d, want 200", ref, rec.Code)
		}
		if rec.Body.Len() == 0 {
			t.Errorf("asset %q is empty", ref)
		}
	}
}

// TestUnknownPathFallsBackToThePage: the front is ONE page, so a deep link a
// user bookmarked must land on it rather than on a 404.
func TestUnknownPathFallsBackToThePage(t *testing.T) {
	rec := httptest.NewRecorder()
	Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, Prefix+"whatever", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("deep link: got %d, want 200", rec.Code)
	}
}

// assetRefs pulls the src/href values the page carries, which the build
// rewrites to hashed names on every change.
func assetRefs(page string) []string {
	var out []string
	for _, attr := range []string{`src="`, `href="`} {
		rest := page
		for {
			i := strings.Index(rest, attr)
			if i < 0 {
				break
			}
			rest = rest[i+len(attr):]
			j := strings.Index(rest, `"`)
			if j < 0 {
				break
			}
			ref := strings.TrimPrefix(rest[:j], "./")
			if strings.HasPrefix(ref, "assets/") {
				out = append(out, ref)
			}
			rest = rest[j:]
		}
	}
	return out
}

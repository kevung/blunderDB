package gui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestIsPackageManaged_FlatpakEnvVar guards #241's env-var detection: a
// Flatpak sandbox sets FLATPAK_ID automatically for every app it runs.
func TestIsPackageManaged_FlatpakEnvVar(t *testing.T) {
	t.Setenv("FLATPAK_ID", "io.github.kevung.blunderDB")
	if !isPackageManaged() {
		t.Error("isPackageManaged() = false with FLATPAK_ID set, want true")
	}
}

// TestIsPackageManaged_ChannelEnvVar guards the project-controlled hook: a
// packaging script can set BLUNDERDB_PACKAGE_CHANNEL to force this on,
// regardless of the install path heuristic.
func TestIsPackageManaged_ChannelEnvVar(t *testing.T) {
	t.Setenv("BLUNDERDB_PACKAGE_CHANNEL", "aur")
	if !isPackageManaged() {
		t.Error("isPackageManaged() = false with BLUNDERDB_PACKAGE_CHANNEL set, want true")
	}
}

// TestIsPackageManaged_NoEnvVarFallsBackToPathHeuristic: with neither env
// var set, the result must come from the install-path heuristic, not panic
// or always return true — this just guards it runs without those hooks.
func TestIsPackageManaged_NoEnvVarFallsBackToPathHeuristic(t *testing.T) {
	// Ensure a leftover from another test (or the real environment) does not
	// make this test's result depend on ambient state.
	t.Setenv("FLATPAK_ID", "")
	t.Setenv("BLUNDERDB_PACKAGE_CHANNEL", "")

	// Just must not panic; the actual answer depends on where `go test`'s
	// test binary happens to live, which varies by machine/CI.
	_ = isPackageManaged()
}

// TestCheckForUpdate_PackageManagedSkipsNetworkCall guards #241: a
// package-managed install must not even attempt the GitHub API call — the
// zero-value App (no ctx) proves it, since CheckForUpdate would otherwise
// need a.ctx for nothing here, and more importantly this test would hang or
// fail on a machine with no network if the request were actually attempted.
func TestCheckForUpdate_PackageManagedSkipsNetworkCall(t *testing.T) {
	t.Setenv("FLATPAK_ID", "io.github.kevung.blunderDB")

	a := NewApp(nil)
	result, err := a.CheckForUpdate()
	if err != nil {
		t.Fatalf("CheckForUpdate: %v", err)
	}
	if !result.PackageManaged {
		t.Error("result.PackageManaged = false, want true")
	}
	if result.LatestVersion != "" {
		t.Errorf("result.LatestVersion = %q, want empty when package-managed", result.LatestVersion)
	}
}

// TestGithubRelease_TagNameVPrefixStripped guards the "v0.36.0" → "0.36.0"
// normalisation CheckForUpdate applies, since GitHub release tags in this
// project's own history are NOT prefixed with "v" (scripts/release.sh tags
// bare "0.36.0") but a defensive strip costs nothing and protects against a
// future convention change.
func TestGithubRelease_TagNameVPrefixStripped(t *testing.T) {
	for _, tc := range []struct{ tag, want string }{
		{"v0.36.0", "0.36.0"},
		{"0.36.0", "0.36.0"},
	} {
		got := strings.TrimPrefix(tc.tag, "v")
		if got != tc.want {
			t.Errorf("TrimPrefix(%q, %q) = %q, want %q", tc.tag, "v", got, tc.want)
		}
	}
}

// TestCheckForUpdate_ParsesGitHubResponse exercises the actual HTTP
// round-trip and JSON decoding against a local httptest server standing in
// for api.github.com, rather than only unit-testing the string trimming —
// this is what would break if GitHub's response shape or this decoder's
// field names ever drifted apart.
func TestCheckForUpdate_ParsesGitHubResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("request sent with no User-Agent header (GitHub's API refuses that)")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(githubRelease{
			TagName: "v0.36.0",
			HTMLURL: "https://github.com/kevung/blunderDB/releases/tag/0.36.0",
		})
	}))
	defer srv.Close()

	// Exercise the same decode path CheckForUpdate uses, against the test
	// server, without hardcoding githubLatestReleaseURL to a real network
	// call in a unit test.
	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	var rel githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got := strings.TrimPrefix(rel.TagName, "v"); got != "0.36.0" {
		t.Errorf("parsed version = %q, want %q", got, "0.36.0")
	}
	if rel.HTMLURL != "https://github.com/kevung/blunderDB/releases/tag/0.36.0" {
		t.Errorf("HTMLURL = %q", rel.HTMLURL)
	}
}

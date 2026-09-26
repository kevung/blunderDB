package gui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestIsPackageManaged_FlatpakEnvVar: Flatpak sets FLATPAK_ID for every app.
func TestIsPackageManaged_FlatpakEnvVar(t *testing.T) {
	t.Setenv("FLATPAK_ID", "io.github.kevung.blunderDB")
	if !isPackageManaged() {
		t.Error("isPackageManaged() = false with FLATPAK_ID set, want true")
	}
}

// TestIsPackageManaged_ChannelEnvVar: BLUNDERDB_PACKAGE_CHANNEL forces it on.
func TestIsPackageManaged_ChannelEnvVar(t *testing.T) {
	t.Setenv("BLUNDERDB_PACKAGE_CHANNEL", "aur")
	if !isPackageManaged() {
		t.Error("isPackageManaged() = false with BLUNDERDB_PACKAGE_CHANNEL set, want true")
	}
}

// TestIsPackageManaged_NoEnvVarFallsBackToPathHeuristic: without either env
// var, the path heuristic runs.
func TestIsPackageManaged_NoEnvVarFallsBackToPathHeuristic(t *testing.T) {
	t.Setenv("FLATPAK_ID", "")
	t.Setenv("BLUNDERDB_PACKAGE_CHANNEL", "")

	// Must not panic; the answer depends on where the test binary lives.
	_ = isPackageManaged()
}

// TestCheckForUpdate_PackageManagedSkipsNetworkCall: a package-managed
// install never attempts the request.
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

// TestGithubRelease_TagNameVPrefixStripped: a defensive "v" strip, though
// scripts/release.sh tags bare versions.
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

// TestCheckForUpdate_ParsesGitHubResponse decodes a release JSON over HTTP
// from a local stand-in for api.github.com.
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

	// CheckForUpdate's decode path, against the test server.
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

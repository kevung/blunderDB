package gui

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"time"
)

// updateCheckTimeout bounds the GitHub API call: this must never hang the
// frontend's startup path waiting on a slow or unreachable network.
const updateCheckTimeout = 5 * time.Second

// githubLatestReleaseURL is blunderDB's own repository — no configurability,
// there is only one upstream to check against.
const githubLatestReleaseURL = "https://api.github.com/repos/kevung/blunderDB/releases/latest"

// UpdateCheckResult is what CheckForUpdate returns. The frontend (which
// already knows the running version — frontend/src/stores/metaStore.js,
// one of the four places scripts/release.sh bumps) does the actual
// comparison and decides whether to show a notice; this side only fetches
// and reports whether the check ran at all.
type UpdateCheckResult struct {
	// PackageManaged is true when this install is detected as coming from a
	// package manager (Flatpak, a distro package, a Homebrew/winget/AUR
	// install — see isPackageManaged): the check is not performed at all,
	// LatestVersion/HTMLURL are empty, and the frontend must not show
	// anything — that channel is the update mechanism, not blunderDB
	// itself.
	PackageManaged bool `json:"packageManaged"`
	// LatestVersion is the latest GitHub release's tag, with any leading
	// "v" stripped (e.g. "0.36.0"). Empty if the check did not run or the
	// request failed.
	LatestVersion string `json:"latestVersion,omitempty"`
	// HTMLURL links to the release page, for the notice to point at.
	HTMLURL string `json:"htmlUrl,omitempty"`
}

// githubRelease is the subset of GitHub's release JSON this cares about.
type githubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

// CheckForUpdate queries the GitHub Releases API for blunderDB's latest
// published release — opt-in (Config.GetCheckForUpdates), and a no-op that
// reports PackageManaged rather than performing the request at all when this
// install looks package-managed (#241). It is the frontend's job to call
// this only when the opt-in is set, compare LatestVersion against its own
// known running version, and show a non-blocking notice — never a dialog
// that blocks the interface.
func (a *App) CheckForUpdate() (*UpdateCheckResult, error) {
	if isPackageManaged() {
		return &UpdateCheckResult{PackageManaged: true}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), updateCheckTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubLatestReleaseURL, nil)
	if err != nil {
		return nil, newGUIError(CodeInternal, err.Error())
	}
	// GitHub's API refuses requests with no User-Agent at all.
	req.Header.Set("User-Agent", "blunderDB-update-check")
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, newGUIError(CodeInternal, "checking for an update: %s", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, newGUIError(CodeInternal, "checking for an update: GitHub returned %s", resp.Status)
	}

	var rel githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, newGUIError(CodeInternal, "checking for an update: %s", err)
	}

	return &UpdateCheckResult{
		LatestVersion: strings.TrimPrefix(rel.TagName, "v"),
		HTMLURL:       rel.HTMLURL,
	}, nil
}

// isPackageManaged reports whether this process looks like it was installed
// through a package manager rather than a manual download of the official
// binary/installer — ADR-0004's "optional host capability" posture: detect,
// then behave differently, rather than assume. A distro package, a
// Homebrew/winget/AUR install and a Flatpak sandbox each already have their
// own update mechanism, and blunderDB nagging about a GitHub release the
// distro has not packaged yet (or never will, on its own schedule) would
// only confuse a user who is not meant to bypass their package manager.
//
// Detection is intentionally two-layered, per #241's spec: an environment
// variable a packaging channel sets (FLATPAK_ID is set automatically by
// Flatpak for every sandboxed app; BLUNDERDB_PACKAGE_CHANNEL is one this
// project's own AUR/Homebrew/winget packaging can set, though none of them
// do yet — this is the hook for whoever wires that up), and a fallback
// heuristic on the running binary's own install path for the common
// system-package locations this project actually ships to.
func isPackageManaged() bool {
	if os.Getenv("FLATPAK_ID") != "" {
		return true
	}
	if os.Getenv("BLUNDERDB_PACKAGE_CHANNEL") != "" {
		return true
	}

	exe, err := os.Executable()
	if err != nil {
		return false
	}
	exe = filepath.ToSlash(exe)

	switch goruntime.GOOS {
	case "linux":
		for _, prefix := range []string{"/usr/bin/", "/usr/local/bin/", "/snap/", "/var/lib/flatpak/", "/opt/"} {
			if strings.HasPrefix(exe, prefix) {
				return true
			}
		}
	case "darwin":
		// Homebrew's Cellar (both the Intel /usr/local and Apple Silicon
		// /opt/homebrew prefixes) and any Homebrew Caskroom install.
		for _, marker := range []string{"/Cellar/", "/Caskroom/", "/opt/homebrew/"} {
			if strings.Contains(exe, marker) {
				return true
			}
		}
	case "windows":
		// winget/the Microsoft Store install packages land under
		// WindowsApps or a per-user WinGet Links/Packages directory; a
		// manual download is run from wherever the user put it, never
		// there.
		if strings.Contains(strings.ToLower(exe), "windowsapps") || strings.Contains(strings.ToLower(exe), "winget") {
			return true
		}
	}
	return false
}

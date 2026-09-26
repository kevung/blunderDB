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

// updateCheckTimeout keeps a slow network from hanging startup.
const updateCheckTimeout = 5 * time.Second

const githubLatestReleaseURL = "https://api.github.com/repos/kevung/blunderDB/releases/latest"

// UpdateCheckResult is what CheckForUpdate returns; the frontend, which knows
// the running version, compares and decides on a notice.
type UpdateCheckResult struct {
	// PackageManaged: the check did not run (isPackageManaged) and the
	// frontend shows nothing.
	PackageManaged bool `json:"packageManaged"`
	// LatestVersion is the latest release tag without "v"; empty if the
	// check did not run or failed.
	LatestVersion string `json:"latestVersion,omitempty"`
	// HTMLURL links to the release page, for the notice to point at.
	HTMLURL string `json:"htmlUrl,omitempty"`
}

// githubRelease is the subset of GitHub's release JSON this cares about.
type githubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

// CheckForUpdate queries the GitHub Releases API for the latest release, or
// reports PackageManaged without a request. Opt-in (Config.GetCheckForUpdates),
// enforced by the frontend, which shows a non-blocking notice.
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

// isPackageManaged reports whether this install came from a package manager,
// which owns updates (ADR-0004: detect, don't assume). It checks FLATPAK_ID
// and BLUNDERDB_PACKAGE_CHANNEL (a hook for packagers), then the binary's
// install path.
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
		// winget and the Microsoft Store install under WindowsApps or WinGet.
		if strings.Contains(strings.ToLower(exe), "windowsapps") || strings.Contains(strings.ToLower(exe), "winget") {
			return true
		}
	}
	return false
}

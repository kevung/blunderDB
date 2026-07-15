package gui

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/adrg/xdg"
)

// The image clipboard is an Optional host capability (see docs/adr/0004): its
// presence is not guaranteed and the way to reach it varies per system. This
// file isolates that variability into a thin Capability probe (raw facts) and a
// pure Fallback policy (which rung to take), so the risky part — the choice — is
// unit-testable without a real host. The imperative shell (exec, file writes)
// stays dumb around it.

// clipboardFacts are the raw facts probeClipboardLinux gathers about the host's
// image-clipboard capability. Facts only; it decides nothing.
type clipboardFacts struct {
	HasXclip  bool // the `xclip` tool is on PATH (X11 clipboard)
	HasWlCopy bool // the `wl-copy` tool is on PATH (Wayland clipboard)
	IsWayland bool // the session looks like Wayland
}

// clipboardRung is one rung of the image-clipboard Fallback strategy on Linux.
type clipboardRung int

const (
	rungWlCopy clipboardRung = iota // pipe through wl-copy
	rungXclip                       // pipe through xclip
	rungFile                        // no clipboard tool: save the PNG to a file
)

// probeClipboardLinux inspects the host and reports the facts. Deliberately thin
// — two LookPaths and an env read, no branching logic worth more than a smoke
// test; all the deciding lives in chooseClipboardRung.
func probeClipboardLinux() clipboardFacts {
	_, xclipErr := exec.LookPath("xclip")
	_, wlErr := exec.LookPath("wl-copy")
	return clipboardFacts{
		HasXclip:  xclipErr == nil,
		HasWlCopy: wlErr == nil,
		IsWayland: isWaylandSession(),
	}
}

// isWaylandSession reports whether the current session looks like Wayland.
func isWaylandSession() bool {
	if strings.EqualFold(os.Getenv("XDG_SESSION_TYPE"), "wayland") {
		return true
	}
	return os.Getenv("WAYLAND_DISPLAY") != ""
}

// chooseClipboardRung is the pure Fallback policy: facts in, chosen rung out, no
// I/O. On Wayland prefer wl-copy — xclip on a pure-Wayland session only reaches
// an XWayland clipboard the compositor may not surface — then fall back to xclip
// if it is the only tool present. On X11 prefer xclip, then wl-copy. When
// neither tool exists, fall back to writing the image to a file.
func chooseClipboardRung(f clipboardFacts) clipboardRung {
	if f.IsWayland {
		switch {
		case f.HasWlCopy:
			return rungWlCopy
		case f.HasXclip:
			return rungXclip
		}
	} else {
		switch {
		case f.HasXclip:
			return rungXclip
		case f.HasWlCopy:
			return rungWlCopy
		}
	}
	return rungFile
}

// copyImageLinux walks the image-clipboard ladder on Linux and returns the path
// the image was saved to when it fell back to a file (empty string when the
// image reached the clipboard directly). A tool that is present but fails at
// runtime degrades to the file rung rather than losing the user's gesture.
func copyImageLinux(pngData []byte) (string, error) {
	switch chooseClipboardRung(probeClipboardLinux()) {
	case rungWlCopy:
		if err := pipeToClipboardTool(pngData, "wl-copy", "--type", "image/png"); err == nil {
			return "", nil
		}
		return saveImageFallback(pngData)
	case rungXclip:
		if err := pipeToClipboardTool(pngData, "xclip", "-selection", "clipboard", "-t", "image/png"); err == nil {
			return "", nil
		}
		return saveImageFallback(pngData)
	default: // rungFile
		return saveImageFallback(pngData)
	}
}

// pipeToClipboardTool runs an external clipboard tool, feeding the PNG on stdin.
func pipeToClipboardTool(pngData []byte, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = bytes.NewReader(pngData)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s failed: %w", name, err)
	}
	return nil
}

// saveImageFallback writes the PNG to a predictable, user-findable location and
// returns the full path. This is the last rung of the ladder on every platform:
// the user's goal (obtain the board image) still succeeds, just through a file
// instead of the clipboard.
func saveImageFallback(pngData []byte) (string, error) {
	return writeBoardImage(imageFallbackDir(), pngData)
}

// writeBoardImage writes the PNG into dir under a timestamped name and returns
// the full path. Split out from directory selection so it is testable against a
// temp dir without depending on the host's real Pictures/home location.
func writeBoardImage(dir string, pngData []byte) (string, error) {
	name := fmt.Sprintf("blunderDB-board-%s.png", time.Now().Format("20060102-150405"))
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, pngData, 0644); err != nil {
		return "", fmt.Errorf("failed to save board image to %s: %w", path, err)
	}
	return path, nil
}

// imageFallbackDir picks where to drop the fallback image: the user's Pictures
// directory when it exists, then the home directory, then the temp directory.
func imageFallbackDir() string {
	if p := xdg.UserDirs.Pictures; p != "" {
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			return p
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		return home
	}
	return os.TempDir()
}

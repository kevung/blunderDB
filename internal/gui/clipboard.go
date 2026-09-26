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

// The image clipboard is an optional host capability (ADR-0004): a thin probe
// gathers facts, a pure policy picks the rung, so the choice is unit-testable.

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

// probeClipboardLinux reports the host's facts; chooseClipboardRung decides.
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

// chooseClipboardRung is the pure policy. On Wayland prefer wl-copy (xclip
// only reaches XWayland's clipboard), on X11 xclip; otherwise a file.
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

// copyImageLinux walks the ladder and returns the fallback file's path ("" if
// the clipboard was reached). A failing tool degrades to the file rung.
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

// saveImageFallback, the last rung everywhere, writes the PNG to a findable
// location and returns its path.
func saveImageFallback(pngData []byte) (string, error) {
	return writeBoardImage(imageFallbackDir(), pngData)
}

// writeBoardImage writes the PNG into dir under a timestamped name and returns
// the full path.
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

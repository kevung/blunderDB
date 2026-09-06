package gui

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Saving a board image to a file the user chooses (issue #278, fiche I.22).
//
// Copying to the clipboard already worked, and it is the everyday gesture. It
// is not the one that produces an illustration for an article, a forum post or
// a lesson: those want a FILE, and often a vector one. SVG is what the board
// already is — two.js draws it — so the vector form costs nothing to offer and
// is the one that survives being enlarged.
//
// The clipboard's fallback ladder (ADR-0004) does not apply here: the user
// picked a path, so there is nothing to fall back to and nothing to guess. A
// failure is an error with the path in it.

// SaveBoardImageDialog asks where to save a board image and returns the chosen
// path, or "" when the user cancelled. format is "svg" or "png"; the extension
// is appended when the user did not type one, because a file named without one
// opens in nothing.
func (a *App) SaveBoardImageDialog(format, defaultName string) (string, error) {
	format = strings.ToLower(strings.TrimSpace(format))
	var filter runtime.FileFilter
	switch format {
	case "svg":
		filter = runtime.FileFilter{DisplayName: "SVG image (*.svg)", Pattern: "*.svg"}
	case "png":
		filter = runtime.FileFilter{DisplayName: "PNG image (*.png)", Pattern: "*.png"}
	default:
		return "", fmt.Errorf("unsupported image format %q (svg or png)", format)
	}

	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:                "Save the board image",
		DefaultFilename:      defaultName,
		Filters:              []runtime.FileFilter{filter},
		CanCreateDirectories: true,
	})
	if err != nil || path == "" {
		return path, err
	}
	if !strings.EqualFold(filepath.Ext(path), "."+format) {
		path += "." + format
	}
	return path, nil
}

// SaveBoardSVG writes an SVG document to path. The content arrives as text
// because that is what it is: a serialised document, not an encoded blob.
func (a *App) SaveBoardSVG(path, svg string) error {
	if path == "" {
		return fmt.Errorf("no path given")
	}
	if err := os.WriteFile(path, []byte(svg), 0o600); err != nil {
		return fmt.Errorf("saving %s: %w", path, err)
	}
	return nil
}

// SaveBoardPNG writes a PNG to path. base64Data is the payload without its
// `data:` prefix — the same shape CopyImageToClipboard takes, so the frontend
// produces one representation and chooses where it goes.
func (a *App) SaveBoardPNG(path, base64Data string) error {
	if path == "" {
		return fmt.Errorf("no path given")
	}
	data, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return fmt.Errorf("decoding the image: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("saving %s: %w", path, err)
	}
	return nil
}

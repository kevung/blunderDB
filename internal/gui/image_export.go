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

// SaveBoardImageDialog asks where to save a document blunderDB produces from
// the board and returns the chosen path, or "" when the user cancelled. format
// is "svg", "png" or "html"; the extension is appended when the user did not
// type one, because a file named without one opens in nothing.
//
// HTML is here rather than in a dialog of its own because it is the same
// question — where do I put this? — asked about the same subject: a document
// made of the board. The report (#279) is a single self-contained file, so it
// travels exactly like an image does.
func (a *App) SaveBoardImageDialog(format, defaultName string) (string, error) {
	format = strings.ToLower(strings.TrimSpace(format))
	var filter runtime.FileFilter
	title := "Save the board image"
	switch format {
	case "svg":
		filter = runtime.FileFilter{DisplayName: "SVG image (*.svg)", Pattern: "*.svg"}
	case "png":
		filter = runtime.FileFilter{DisplayName: "PNG image (*.png)", Pattern: "*.png"}
	case "html":
		filter = runtime.FileFilter{DisplayName: "HTML document (*.html)", Pattern: "*.html"}
		title = "Save the report"
	default:
		return "", fmt.Errorf("unsupported format %q (svg, png or html)", format)
	}

	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:                title,
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

// SaveBoardSVG writes a text document — an SVG board, or the HTML report
// (#279) — to path. The content arrives as text because that is what it is: a
// serialised document, not an encoded blob.
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

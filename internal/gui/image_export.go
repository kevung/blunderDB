package gui

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Saving a board image to a file the user chooses. No fallback ladder
// (ADR-0004) here: the user picked the path, so a failure is an error.

// SaveBoardImageDialog asks where to save a board document ("svg", "png" or
// the self-contained "html" report), appending a missing extension. Returns
// "" on cancel.
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

// SaveBoardSVG writes a text document (an SVG board or the HTML report).
func (a *App) SaveBoardSVG(path, svg string) error {
	if path == "" {
		return fmt.Errorf("no path given")
	}
	if err := os.WriteFile(path, []byte(svg), 0o600); err != nil {
		return fmt.Errorf("saving %s: %w", path, err)
	}
	return nil
}

// SaveBoardPNG writes a PNG to path; base64Data has no `data:` prefix, as
// for CopyImageToClipboard.
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

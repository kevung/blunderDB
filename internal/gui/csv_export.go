package gui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Saving a CSV of the Direction view to a file: the same text the clipboard
// receives, so the two cannot drift. No fallback ladder (ADR-0004): the user
// picked the path, so a failure is an error.

// SaveCSV asks where to save body, proposing defaultName, and writes it there.
// It returns the path written, or "" when the user cancelled.
func (a *App) SaveCSV(defaultName, body string) (string, error) {
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:                "Save as CSV",
		DefaultFilename:      defaultName,
		Filters:              []runtime.FileFilter{{DisplayName: "CSV (*.csv)", Pattern: "*.csv"}},
		CanCreateDirectories: true,
	})
	if err != nil || path == "" {
		return "", err
	}
	path = csvPath(path)
	if err := writeCSV(path, body); err != nil {
		return "", err
	}
	return path, nil
}

// csvPath appends a missing ".csv".
func csvPath(path string) string {
	if strings.EqualFold(filepath.Ext(path), ".csv") {
		return path
	}
	return path + ".csv"
}

// writeCSV writes body as it is — no BOM, no line-ending rewrite — so the file
// is the copied CSV byte for byte.
func writeCSV(path, body string) error {
	if path == "" {
		return fmt.Errorf("no path given")
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		return fmt.Errorf("saving %s: %w", path, err)
	}
	return nil
}

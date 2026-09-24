package gui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Saving a CSV of the Direction view to a file (issue #454, D8.1).
//
// The standings and the directory used to leave only through the clipboard. A
// director who shows "Tuesday's standings" wants a file, not something to paste
// somewhere first. The CSV is the one the clipboard receives: the frontend asks
// the database once and hands the same text to either destination, so the two
// cannot drift. The command line already writes the standings to a file with
// `blunderdb tournament standings … > classement.csv`, from the same
// Database.StandingsCSV.
//
// As for the board image, the clipboard's fallback ladder (ADR-0004) does not
// apply: the user picked a path, so there is nothing to fall back to. A failure
// is an error with the path in it.

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

// csvPath appends ".csv" when the user typed a name without it: a file named
// without one opens in nothing.
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

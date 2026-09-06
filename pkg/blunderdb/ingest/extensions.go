package ingest

import (
	"path/filepath"
	"sort"
	"strings"
)

// The file extensions blunderDB knows how to import, in ONE place.
//
// This list was written three times — the Wails app's drag-and-drop and
// folder collection, the CLI's batch walk, and that walk's own error message
// — and the three could drift apart without anything noticing. A watched
// folder (#258) would have been the fourth.

// importableExtensions maps a lower-cased extension (with its dot) to true.
var importableExtensions = map[string]bool{
	".txt": true, // BGBlitz position text, Snowie-style match text
	".xg":  true, // eXtreme Gammon match
	".xgp": true, // eXtreme Gammon single position
	".sgf": true, // GNU Backgammon match
	".mat": true, // Jellyfish match
	".bgf": true, // BGBlitz match
}

// IsImportable reports whether path's extension is one blunderDB can read.
// The comparison is on the extension only: what the file actually contains is
// the parser's business, and a mis-named file is a parse error, not a
// filtering one.
func IsImportable(path string) bool {
	return importableExtensions[strings.ToLower(filepath.Ext(path))]
}

// ImportableExtensions lists the extensions, sorted, for a message that has to
// name them ("no match file found in …"). Sorted so the message is stable:
// map iteration order would make the same error read differently twice.
func ImportableExtensions() []string {
	out := make([]string, 0, len(importableExtensions))
	for ext := range importableExtensions {
		out = append(out, ext)
	}
	sort.Strings(out)
	return out
}

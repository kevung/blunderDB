package database

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/printsheet"
)

// MatchSheetOptions are the choices of the match sheet's print dialog.
type MatchSheetOptions = printsheet.MatchOptions

// MatchSheetHTML renders the printable match sheet of each match, one A4
// sheet a match in one self-contained HTML document: transcription, header
// and the positions to review written in ArkID. It writes nothing to the
// library (ADR-0007).
func (d *Database) MatchSheetHTML(matchIDs []int64, opts MatchSheetOptions) (string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	page, err := printsheet.BuildMatches(context.Background(), d.store, "", matchIDs, opts)
	return string(page), err
}

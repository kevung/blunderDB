package database

import (
	"context"
	"os"

	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

// ExportMatchMAT writes match matchID as a Jellyfish/gnubg .mat transcript to
// outputPath. It reads and renders first, then writes the file, so a read
// failure never leaves a truncated .mat behind. The desktop store is
// single-tenant, hence the empty scope. The same path backs the GUI export
// button and the CLI `export --type mat` command (CLI/GUI parity).
func (d *Database) ExportMatchMAT(matchID int64, outputPath string) error {
	ctx := context.Background()
	m, games, moves, err := ingest.ReadMatchForMAT(ctx, d.store, "", matchID)
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, []byte(ingest.RenderMAT(m, games, moves)), 0o644)
}

// SuggestMatFilename returns the default .mat filename for a match — the name
// pre-filled in the GUI save dialog and used to auto-name files in a CLI batch
// export. Both come from ingest.SuggestMATFilename so they never diverge.
func (d *Database) SuggestMatFilename(matchID int64) (string, error) {
	m, err := d.GetMatchByID(matchID)
	if err != nil {
		return "", err
	}
	return ingest.SuggestMATFilename(m), nil
}

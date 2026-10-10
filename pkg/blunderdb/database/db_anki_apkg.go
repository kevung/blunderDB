package database

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/apkg"
)

// ExportAnkiPackage writes a study deck (deckID) or a collection
// (collectionID), exactly one of them, as an Anki package (.apkg) to
// outputPath, its cards written in lang. A living collection's query is
// evaluated now. The package is built whole before the file is written, so a
// failure never leaves a truncated .apkg behind; the database is only read
// (ADR-0007).
func (d *Database) ExportAnkiPackage(deckID, collectionID int64, lang, outputPath string) (*apkg.Result, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}
	var buf bytes.Buffer
	res, err := apkg.Export(context.Background(), d.store, "", apkg.Source{DeckID: deckID, CollectionID: collectionID}, lang, time.Now(), &buf)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(outputPath, buf.Bytes(), 0o644); err != nil {
		return nil, fmt.Errorf("write %s: %w", outputPath, err)
	}
	return res, nil
}

package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/apkg"
)

// ExportAnkiPackage writes a study deck (deckID) or a collection
// (collectionID), exactly one of them, as an Anki package (.apkg) to
// outputPath, its cards written in lang. A living collection's query is
// evaluated now. The package is written to a temporary file beside outputPath
// and renamed over it once complete, so a failure never leaves a truncated
// .apkg behind; the database is only read (ADR-0007).
func (d *Database) ExportAnkiPackage(deckID, collectionID int64, lang, outputPath string) (*apkg.Result, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}
	tmp, err := os.CreateTemp(filepath.Dir(outputPath), ".blunderdb-*.apkg")
	if err != nil {
		return nil, fmt.Errorf("write %s: %w", outputPath, err)
	}
	defer func() { _ = os.Remove(tmp.Name()) }() // a no-op once renamed
	res, err := apkg.Export(context.Background(), d.store, "", apkg.Source{DeckID: deckID, CollectionID: collectionID}, lang, time.Now(), tmp)
	if cerr := tmp.Close(); err == nil && cerr != nil {
		err = fmt.Errorf("write %s: %w", outputPath, cerr)
	}
	if err != nil {
		return nil, err
	}
	// CreateTemp makes the file private; a package is an ordinary document.
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		return nil, fmt.Errorf("write %s: %w", outputPath, err)
	}
	if err := os.Rename(tmp.Name(), outputPath); err != nil {
		return nil, fmt.Errorf("write %s: %w", outputPath, err)
	}
	return res, nil
}

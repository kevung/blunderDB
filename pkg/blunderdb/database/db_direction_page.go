package database

import (
	"context"
	"os"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
)

// The standalone display page, rewritten after every action without a gesture: cheap, and
// harmless when there is nothing to write. A write failure is reported and NEVER interrupts the
// tournament (ADR-0004: detect and degrade).

// SetDirectionStrings hands the frontend's `direction` catalogue to the backend, so the page it
// writes speaks the language the panel speaks.
//
// Nicomaque emits codes; rather than copy frontend/src/i18n/locales/<lang>.json into Go, the
// frontend passes its block so both sides render the same words.
func (d *Database) SetDirectionStrings(lang, catalogJSON string) error {
	cat, err := direction.NewCatalog([]byte(catalogJSON))
	if err != nil {
		return err
	}
	d.directionMu.Lock()
	defer d.directionMu.Unlock()
	d.directionCatalog, d.directionLang = cat, lang
	return nil
}

// directionStrings gives the catalogue in force, defaulting to French — the engine's own
// language, so a host that never published one still gets sentences rather than codes.
func (d *Database) directionStrings() (*direction.Catalog, string) {
	d.directionMu.RLock()
	defer d.directionMu.RUnlock()
	lang := d.directionLang
	if lang == "" {
		lang = "fr"
	}
	return d.directionCatalog, lang
}

// WriteDirectionPage rewrites the display page of a Direction and returns the file written.
//
// It returns an empty path, and no error, when no folder has been chosen.
func (d *Database) WriteDirectionPage(tournamentID int64) (string, error) {
	dir, err := direction.Open(context.Background(), d.DirectionStore(), tournamentID)
	if err != nil {
		return "", err
	}
	out := dir.Record().OutputDir
	if out == "" {
		return "", nil
	}
	cat, lang := d.directionStrings()
	page, err := dir.Page(cat, lang, time.Now())
	if err != nil {
		return "", err
	}
	return direction.WritePage(out, page)
}

// DirectionPageHTML renders the page without writing it, which is what a test — and the CLI's
// `tournament page` — needs.
func (d *Database) DirectionPageHTML(tournamentID int64) (string, error) {
	dir, err := direction.Open(context.Background(), d.DirectionStore(), tournamentID)
	if err != nil {
		return "", err
	}
	cat, lang := d.directionStrings()
	return dir.Page(cat, lang, time.Now())
}

// The printable pairing sheet: the display page's plumbing, plus a print instruction that opens
// the system dialog by itself.

// DirectionPairingSheetHTML renders the sheet of one batch without writing it.
func (d *Database) DirectionPairingSheetHTML(tournamentID int64, round int) (string, error) {
	dir, err := direction.Open(context.Background(), d.DirectionStore(), tournamentID)
	if err != nil {
		return "", err
	}
	cat, lang := d.directionStrings()
	return dir.PairingSheet(cat, lang, round)
}

// WriteDirectionPairingSheet writes the sheet and returns the file to open.
//
// It goes into the Direction's display folder when there is one, and into the system's
// temporary folder otherwise: printing must not require choosing a folder first.
func (d *Database) WriteDirectionPairingSheet(tournamentID int64, round int) (string, error) {
	dir, err := direction.Open(context.Background(), d.DirectionStore(), tournamentID)
	if err != nil {
		return "", err
	}
	cat, lang := d.directionStrings()
	sheet, err := dir.PairingSheet(cat, lang, round)
	if err != nil {
		return "", err
	}
	out := dir.Record().OutputDir
	if out == "" {
		out = os.TempDir()
	}
	return direction.WriteFileAtomically(out, direction.SheetName, sheet)
}

// DirectionUpcomingSheetHTML renders the sheet of the round the queue proposes, before it is
// launched, headed by the date the director typed. Nothing is written to the log.
func (d *Database) DirectionUpcomingSheetHTML(tournamentID int64, announced string) (string, error) {
	dir, err := direction.Open(context.Background(), d.DirectionStore(), tournamentID)
	if err != nil {
		return "", err
	}
	cat, lang := d.directionStrings()
	return dir.UpcomingSheet(cat, lang, announced, time.Now())
}

// WriteDirectionUpcomingSheet writes the announced sheet where the pairing sheet goes, under a
// name of its own, and returns the file to open.
func (d *Database) WriteDirectionUpcomingSheet(tournamentID int64, announced string) (string, error) {
	sheet, err := d.DirectionUpcomingSheetHTML(tournamentID, announced)
	if err != nil {
		return "", err
	}
	dir, err := direction.Open(context.Background(), d.DirectionStore(), tournamentID)
	if err != nil {
		return "", err
	}
	out := dir.Record().OutputDir
	if out == "" {
		out = os.TempDir()
	}
	return direction.WriteFileAtomically(out, direction.UpcomingSheetName, sheet)
}

// DirectionRounds counts the batches of the current phase: how many sheets there are to choose
// from. A batch is what a director calls a round in a Swiss by rounds, a block in a GSL, a
// round in a bracket — the engine has no word for it, and needs none.
func (d *Database) DirectionRounds(tournamentID int64) (int, error) {
	dir, err := direction.Open(context.Background(), d.DirectionStore(), tournamentID)
	if err != nil {
		return 0, err
	}
	return dir.Rounds(), nil
}

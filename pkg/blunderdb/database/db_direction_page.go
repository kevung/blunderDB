package database

import (
	"context"
	"os"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
)

// The standalone display page, written at every event (issue #386).
//
// The acceptance criterion is that after the folder is chosen once, the display updates WITHOUT
// A GESTURE. So the page is not a command: it is a consequence. The panel calls
// WriteDirectionPage after every action it takes, and this file makes that call cheap enough to
// be made every time and harmless when there is nothing to write.
//
// A failure — folder unmounted, disk full — is reported and NEVER interrupts the direction of
// the tournament. A director in a hall does not stop running their tournament because a USB key
// was pulled out (ADR-0004's posture: detect and degrade).

// SetDirectionStrings hands the frontend's `direction` catalogue to the backend, so the page it
// writes speaks the language the panel speaks.
//
// blunderDB translates in nine languages and Nicomaque in none: the engine emits codes, and the
// strings live in frontend/src/i18n/locales/<lang>.json. Rather than copy them into Go, the
// frontend passes its own block, and both sides render the same codes from the same words.
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
// It returns an empty path, and no error, when no folder has been chosen: writing the page is a
// consequence of every event, and a director who never asked for a display must not be told
// about it at each result.
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
// `tournament page` (#395) — needs.
func (d *Database) DirectionPageHTML(tournamentID int64) (string, error) {
	dir, err := direction.Open(context.Background(), d.DirectionStore(), tournamentID)
	if err != nil {
		return "", err
	}
	cat, lang := d.directionStrings()
	return dir.Page(cat, lang, time.Now())
}

// The printable pairing sheet (issue #387).
//
// Paper is still the director's tool: the sheet goes on the welcome desk, and the players come
// and read it rather than asking. It is the same rendering plumbing as the display page — same
// catalogue, same credit — with one difference of purpose: it exists to leave the screen, so it
// carries a print instruction and opens the system dialog by itself.

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
// temporary folder otherwise: printing must not require choosing a folder first — that would be
// a click, and the budget is three.
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

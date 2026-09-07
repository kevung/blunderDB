package database

import (
	"context"
	"sync"
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

// directionStrings holds the catalogue the frontend handed over, so the page speaks the user's
// language. It is per-process and not per-database: it is the interface's language, not the
// tournament's.
var directionStrings struct {
	mu   sync.RWMutex
	cat  *direction.Catalog
	lang string
}

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
	directionStrings.mu.Lock()
	defer directionStrings.mu.Unlock()
	directionStrings.cat, directionStrings.lang = cat, lang
	return nil
}

func currentDirectionStrings() (*direction.Catalog, string) {
	directionStrings.mu.RLock()
	defer directionStrings.mu.RUnlock()
	lang := directionStrings.lang
	if lang == "" {
		lang = "fr"
	}
	return directionStrings.cat, lang
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
	cat, lang := currentDirectionStrings()
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
	cat, lang := currentDirectionStrings()
	return dir.Page(cat, lang, time.Now())
}

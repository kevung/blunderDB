package direction

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
	"github.com/PileOfCells/backgammon-tournoi/render"
)

// The standalone display page (tasks/nicomaque/fonctionnel.md §9): ONE HTML FILE, no route, no
// player screen — ADR-0039 keeps tournaments out of the web front, and a file opens offline.
//
// PageName is the file's name inside the chosen folder. It never changes, so an open browser
// tab stays right: a refresh shows the new page.
const PageName = "tournoi.html"

// SheetName is the printable pairing sheet's file name.
const SheetName = "appariements.html"

// Page renders the display page in the host's language.
func (d *Direction) Page(cat *Catalog, lang string, now time.Time) (string, error) {
	if d.st == nil {
		return "", ErrNoDirection
	}
	r := d.renderer(cat, lang)
	return r.Page(d.st, d.ProposeAt(now), now), nil
}

// renderer builds the engine's renderer for this Direction, in the host's language.
//
// The credit comes from the host catalogue too: the engine's default is French only.
func (d *Direction) renderer(cat *Catalog, lang string) *render.Renderer {
	r := render.New(NewLabeler(cat, func(id tournoi.PlayerID) string { return d.playerName(id) }))
	r.Lang = lang
	if credit, ok := cat.t("credit.by", nil); ok && credit != "" {
		r.Credit = credit
	}
	return r
}

// playerName gives a player's name, falling back to the identifier for a player the state does
// not know — a page with a hole in it is worse than a page with an identifier in it.
func (d *Direction) playerName(id tournoi.PlayerID) string {
	if d.st == nil {
		return string(id)
	}
	if p := d.st.Players[id]; p != nil && p.Name != "" {
		return p.Name
	}
	return string(id)
}

// PairingSheet renders the printable pairing sheet of one batch — a round, a block, a bracket
// round, whatever the director calls it. round 0, or beyond the last, gives the most recent,
// which is the one printed in practice.
//
// The page carries a print instruction, so the print dialog opens on its own.
func (d *Direction) PairingSheet(cat *Catalog, lang string, round int) (string, error) {
	if d.st == nil {
		return "", ErrNoDirection
	}
	return printOnOpen(d.renderer(cat, lang).PairingSheet(d.st, round)), nil
}

// Rounds counts the batches of the current phase — how many sheets there are to choose from.
func (d *Direction) Rounds() int {
	if d.st == nil {
		return 0
	}
	return len(render.Batches(d.st))
}

// printOnOpen adds the one script blunderDB ever puts in a page it produces.
//
// The display page must carry NONE — it is a wall display, it reloads itself, and a test
// forbids a script there. A pairing sheet is the opposite: it exists to leave the screen, and
// without this the director opens a tab and then hunts for Ctrl+P.
func printOnOpen(doc string) string {
	const script = `<script>window.addEventListener("load",function(){window.print()})</script>`
	if i := strings.LastIndex(doc, "</body>"); i >= 0 {
		return doc[:i] + script + doc[i:]
	}
	return doc + script
}

// WritePage writes the page into dir, atomically: a display that is being refreshed must never
// be caught half-written by the browser reading it at that instant.
//
// It returns the path written, which is what "open in the browser" needs.
func WritePage(dir, page string) (string, error) {
	return WriteFileAtomically(dir, PageName, page)
}

// WriteFileAtomically writes one of the pages this package produces, atomically.
func WriteFileAtomically(dir, name, page string) (string, error) {
	if dir == "" {
		return "", fmt.Errorf("direction: no output folder chosen")
	}
	final := filepath.Join(dir, name)
	tmp, err := os.CreateTemp(dir, "."+strings.TrimSuffix(name, ".html")+"-*.html")
	if err != nil {
		return "", fmt.Errorf("direction: writing %s: %w", name, err)
	}
	partial := tmp.Name()
	if _, err := tmp.WriteString(page); err != nil {
		tmp.Close()
		os.Remove(partial)
		return "", fmt.Errorf("direction: writing %s: %w", name, err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(partial)
		return "", fmt.Errorf("direction: writing %s: %w", name, err)
	}
	if err := os.Rename(partial, final); err != nil {
		os.Remove(partial)
		return "", fmt.Errorf("direction: writing %s: %w", name, err)
	}
	return final, nil
}

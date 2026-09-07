package direction

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
	"github.com/PileOfCells/backgammon-tournoi/render"
)

// The standalone display page (issue #386, ADR-0047 §9).
//
// A tournament is watched: the players want to know who they play and at which table without
// coming to ask the director. The answer is ONE HTML FILE — no consultation route, no player
// screen. ADR-0039 closed the web front's perimeter to tournaments, and a file opens offline,
// from a USB key, on the hall's computer, which is the only thing one can count on a Sunday
// morning.
//
// PageName is the file's name inside the chosen folder. It never changes, so the browser tab
// the director opened at 9 h is still the right one at 19 h: a refresh shows the new page.
const PageName = "tournoi.html"

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
// The credit is taken from the SAME catalogue as everything else. The engine's default says it
// in French, and a page written in Japanese would say one sentence in French — the credit is
// owed to Nicolas Harmand in every language, not only in his own.
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

// WritePage writes the page into dir, atomically: a display that is being refreshed must never
// be caught half-written by the browser reading it at that instant.
//
// It returns the path written, which is what "open in the browser" needs.
func WritePage(dir, page string) (string, error) {
	if dir == "" {
		return "", fmt.Errorf("direction: no output folder chosen")
	}
	final := filepath.Join(dir, PageName)
	tmp, err := os.CreateTemp(dir, ".tournoi-*.html")
	if err != nil {
		return "", fmt.Errorf("direction: writing the display page: %w", err)
	}
	name := tmp.Name()
	if _, err := tmp.WriteString(page); err != nil {
		tmp.Close()
		os.Remove(name)
		return "", fmt.Errorf("direction: writing the display page: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return "", fmt.Errorf("direction: writing the display page: %w", err)
	}
	if err := os.Rename(name, final); err != nil {
		os.Remove(name)
		return "", fmt.Errorf("direction: writing the display page: %w", err)
	}
	return final, nil
}

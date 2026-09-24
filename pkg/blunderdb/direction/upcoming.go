package direction

import (
	"errors"
	"fmt"
	"html"
	"strings"
	"time"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
	"github.com/PileOfCells/backgammon-tournoi/render"
)

// The sheet of a round announced before it is launched (issue #451, D7.2).
//
// A club championship plays one round a week. The director pairs round 3 on Friday and wants
// the sheet on the club's board for Monday — but a sheet only existed for a LAUNCHED round,
// and launching on Friday would stamp the matches Friday: the engine reads the rounds from the
// start times.
//
// So the round is launched on a COPY: the log is replayed into a throwaway state, every match
// the queue proposes is started there at one instant, and the engine's own renderer prints that
// batch. The Direction's state and its log are not touched — nothing is written, not even a
// note. The sheet carries the date the director typed, marked as announced.

// UpcomingSheetName is the announced sheet's file name. It is not SheetName: the sheet of the
// round being played stays where the players read it.
const UpcomingSheetName = "appariements-annonce.html"

// ErrNothingProposed: the queue proposes no match, so there is no round to announce.
var ErrNothingProposed = errors.New("direction: no match is proposed")

// UpcomingSheet renders the sheet of the matches the queue proposes at now, as if they were
// launched together, headed by announced — the date and time the director typed.
func (d *Direction) UpcomingSheet(cat *Catalog, lang, announced string, now time.Time) (string, error) {
	if d.st == nil {
		return "", ErrNoDirection
	}
	st, err := tournoi.Replay(d.journal)
	if err != nil {
		return "", err
	}
	// The engine tells the rounds apart by their start time: the copy's round must start after
	// every match already launched, or it would be sorted among them. A host clock behind the
	// log (a laptop set wrong, a test) must not print the last round instead of the next.
	for _, m := range st.Matches {
		if !m.Start.Before(now) {
			now = m.Start.Add(time.Minute)
		}
	}
	started := 0
	for _, a := range d.st.ProposeAt(now) {
		if a.Kind != tournoi.ActStartMatch {
			continue
		}
		// A proposal waiting for a table starts without one on the copy: the sheet shows the
		// whole round, which is what the director announces, and "—" where no table is free.
		ev, err := st.EventFromAction(a, now)
		if err != nil {
			return "", err
		}
		if err := st.Apply(ev); err != nil {
			return "", fmt.Errorf("direction: announcing %s-%s: %w", a.A, a.B, err)
		}
		started++
	}
	if started == 0 {
		return "", ErrNothingProposed
	}
	sheet := d.renderer(cat, lang).PairingSheet(st, len(render.Batches(st)))
	// The renderer writes the table number as it is; a match with no table yet gets a dash.
	sheet = strings.ReplaceAll(sheet, "<tr><td>0</td>", "<tr><td>—</td>")
	return printOnOpen(announce(sheet, cat, announced)), nil
}

// announce puts the announced date right under the sheet's title.
func announce(sheet string, cat *Catalog, announced string) string {
	text := html.EscapeString(strings.TrimSpace(announced))
	if label, ok := cat.t("term.announced", map[string]any{"at": text}); ok && label != "" {
		text = label
	}
	if text == "" {
		return sheet
	}
	line := `<p class="resume">` + text + `</p>`
	if i := strings.Index(sheet, "</h2>"); i >= 0 {
		i += len("</h2>")
		return sheet[:i] + line + sheet[i:]
	}
	return line + sheet
}

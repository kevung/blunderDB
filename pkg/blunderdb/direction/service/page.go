package service

import (
	"context"
	"fmt"
	"os"
	"slices"
	"sync"
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
func (d *Service) SetDirectionStrings(ctx context.Context, lang, catalogJSON string) error {
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
func (d *Service) directionStrings(ctx context.Context) (*direction.Catalog, string) {
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
// It returns an empty path, and no error, when no folder has been chosen. When the Direction
// plays in a Rencontre with a folder of its own, this writes into that Rencontre's `<slug>/`
// subfolder instead (ADR-0056 §6) and regenerates the room's wall page too — a gesture in any
// member is a gesture on the whole room.
func (d *Service) WriteDirectionPage(ctx context.Context, tournamentID int64) (string, error) {
	defer d.regenerateRencontrePage(ctx, tournamentID)
	return d.writeOwnPage(ctx, tournamentID)
}

// writePages rewrites, after a gesture, the display page of each Tournament it wrote in (0s
// skipped), then the wall page of each room they play in — once, however many of its members
// the gesture touched. Called with no lock held: writing a file is not part of the gesture, and
// a failure does not undo it (ADR-0004); it is reported as a PageWarning instead.
func (d *Service) writePages(ctx context.Context, tournamentIDs ...int64) {
	var rooms []int64
	for _, tid := range tournamentIDs {
		if tid == 0 {
			continue
		}
		if _, err := d.writeOwnPage(ctx, tid); err != nil {
			d.warnPage(ctx, PageWarning{TournamentID: tid, Err: err.Error()})
		}
		if rid, err := d.st.Rencontres().Of(ctx, d.scope, tid); err == nil && rid != 0 && !slices.Contains(rooms, rid) {
			rooms = append(rooms, rid)
		}
	}
	for _, rid := range rooms {
		d.writeWallPage(ctx, rid)
	}
}

// writeWallPage is WriteRencontrePage after a gesture: a failure is a PageWarning.
func (d *Service) writeWallPage(ctx context.Context, rid int64) {
	if _, err := d.WriteRencontrePage(ctx, rid); err != nil {
		d.warnPage(ctx, PageWarning{RencontreID: rid, Err: err.Error()})
	}
}

// writeRencontrePages rewrites a room's wall page and each member's own page: what a gesture on
// the room changed.
func (d *Service) writeRencontrePages(ctx context.Context, rid int64) {
	if r, err := d.st.Rencontres().Get(ctx, d.scope, rid); err == nil && len(r.TournamentIDs) > 0 {
		d.writePages(ctx, r.TournamentIDs...)
		return
	}
	d.writeWallPage(ctx, rid)
}

// PageWarning reports a display page a gesture could not rewrite — a folder gone, a disk
// full. The gesture itself stands; the director is told so the wall is not left stale unseen.
type PageWarning struct {
	TournamentID int64  `json:"tournamentId,omitempty"`
	RencontreID  int64  `json:"rencontreId,omitempty"`
	Err          string `json:"error"`
}

func (w PageWarning) String() string {
	if w.RencontreID != 0 {
		return fmt.Sprintf("rencontre %d: page not written: %s", w.RencontreID, w.Err)
	}
	return fmt.Sprintf("tournament %d: page not written: %s", w.TournamentID, w.Err)
}

type pageWarningsKey struct{}

type pageWarnings struct {
	mu   sync.Mutex
	list []PageWarning
}

// CollectPageWarnings returns ctx collecting the PageWarnings of the gestures run under it, for
// a caller that answers them with its response (the serve daemon).
func CollectPageWarnings(ctx context.Context) context.Context {
	return context.WithValue(ctx, pageWarningsKey{}, &pageWarnings{})
}

// PageWarnings are the warnings collected under ctx (CollectPageWarnings).
func PageWarnings(ctx context.Context) []PageWarning {
	c, ok := ctx.Value(pageWarningsKey{}).(*pageWarnings)
	if !ok {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.list)
}

// OnPageWarning installs the function told of every PageWarning in this Memory — the desktop's
// status bar. nil removes it.
func (m *Memory) OnPageWarning(f func(PageWarning)) {
	m.directionMu.Lock()
	defer m.directionMu.Unlock()
	m.pageWarning = f
}

func (d *Service) warnPage(ctx context.Context, w PageWarning) {
	if c, ok := ctx.Value(pageWarningsKey{}).(*pageWarnings); ok {
		c.mu.Lock()
		c.list = append(c.list, w)
		c.mu.Unlock()
	}
	d.directionMu.RLock()
	f := d.pageWarning
	d.directionMu.RUnlock()
	if f != nil {
		f(w)
	}
}

// writeRoomPages is writePages for a gesture that may have changed the whole room: the
// Direction's page and, when it plays in a Rencontre, every sister's.
func (d *Service) writeRoomPages(ctx context.Context, tournamentID int64) {
	if rid, err := d.st.Rencontres().Of(ctx, d.scope, tournamentID); err == nil && rid != 0 {
		if r, err := d.st.Rencontres().Get(ctx, d.scope, rid); err == nil {
			d.writePages(ctx, r.TournamentIDs...)
			return
		}
	}
	d.writePages(ctx, tournamentID)
}

// writeOwnPage is WriteDirectionPage without the room's wall page.
func (d *Service) writeOwnPage(ctx context.Context, tournamentID int64) (string, error) {
	// The folder first, from the record alone: with none chosen — the daemon's usual case —
	// a gesture pays no replay for a page nobody reads.
	rec, err := d.st.Directions().Get(ctx, d.scope, tournamentID)
	if err != nil {
		return "", err
	}
	out := d.effectiveOutputDir(ctx, tournamentID, rec.OutputDir)
	if out == "" {
		return "", nil
	}
	dir, err := direction.Open(ctx, d.dirStore(), tournamentID)
	if err != nil {
		return "", err
	}
	cat, lang := d.directionStrings(ctx)
	page, err := d.directionPage(ctx, dir, tournamentID, cat, lang)
	if err != nil {
		return "", err
	}
	return direction.WritePage(out, page)
}

// DirectionPageHTML renders the page without writing it, which is what a test — and the CLI's
// `tournament page` — needs.
func (d *Service) DirectionPageHTML(ctx context.Context, tournamentID int64) (string, error) {
	dir, err := direction.Open(ctx, d.dirStore(), tournamentID)
	if err != nil {
		return "", err
	}
	cat, lang := d.directionStrings(ctx)
	return d.directionPage(ctx, dir, tournamentID, cat, lang)
}

// directionPage renders the display page, with the event's bracket in rotation when it has one.
func (d *Service) directionPage(ctx context.Context, dir *direction.Direction, tournamentID int64, cat *direction.Catalog, lang string) (string, error) {
	now := time.Now()
	cfg, err := dir.Config()
	if err != nil {
		return "", err
	}
	plan, err := d.planFor(ctx, tournamentID, cfg)
	if err != nil {
		return "", err
	}
	page, err := dir.PageIn(cat, lang, now, plan, d.memberNames(ctx, tournamentID))
	if err != nil {
		return "", err
	}
	name := fmt.Sprintf("#%d", tournamentID)
	if t, err := d.st.Tournaments().Get(ctx, d.scope, tournamentID); err == nil && t.Name != "" {
		name = t.Name
	}
	return direction.WithBracketView(page, d.wallBracket(ctx, dir, name), cat, now), nil
}

// The printable pairing sheet: the display page's plumbing, plus a print instruction that opens
// the system dialog by itself.

// DirectionPairingSheetHTML renders the sheet of one batch without writing it.
func (d *Service) DirectionPairingSheetHTML(ctx context.Context, tournamentID int64, round int) (string, error) {
	dir, err := direction.Open(ctx, d.dirStore(), tournamentID)
	if err != nil {
		return "", err
	}
	cat, lang := d.directionStrings(ctx)
	return dir.PairingSheet(cat, lang, round)
}

// WriteDirectionPairingSheet writes the sheet and returns the file to open.
//
// It goes into the Direction's display folder when there is one, and into the system's
// temporary folder otherwise: printing must not require choosing a folder first.
func (d *Service) WriteDirectionPairingSheet(ctx context.Context, tournamentID int64, round int) (string, error) {
	dir, err := direction.Open(ctx, d.dirStore(), tournamentID)
	if err != nil {
		return "", err
	}
	cat, lang := d.directionStrings(ctx)
	sheet, err := dir.PairingSheet(cat, lang, round)
	if err != nil {
		return "", err
	}
	out := d.effectiveOutputDir(ctx, tournamentID, dir.Record().OutputDir)
	if out == "" {
		out = os.TempDir()
	}
	return direction.WriteFileAtomically(out, direction.SheetName, sheet)
}

// DirectionUpcomingSheetHTML renders the sheet of the round the queue proposes, before it is
// launched, headed by the date the director typed. Nothing is written to the log.
func (d *Service) DirectionUpcomingSheetHTML(ctx context.Context, tournamentID int64, announced string) (string, error) {
	dir, err := direction.Open(ctx, d.dirStore(), tournamentID)
	if err != nil {
		return "", err
	}
	cat, lang := d.directionStrings(ctx)
	return dir.UpcomingSheet(cat, lang, announced, time.Now())
}

// WriteDirectionUpcomingSheet writes the announced sheet where the pairing sheet goes, under a
// name of its own, and returns the file to open.
func (d *Service) WriteDirectionUpcomingSheet(ctx context.Context, tournamentID int64, announced string) (string, error) {
	sheet, err := d.DirectionUpcomingSheetHTML(ctx, tournamentID, announced)
	if err != nil {
		return "", err
	}
	dir, err := direction.Open(ctx, d.dirStore(), tournamentID)
	if err != nil {
		return "", err
	}
	out := d.effectiveOutputDir(ctx, tournamentID, dir.Record().OutputDir)
	if out == "" {
		out = os.TempDir()
	}
	return direction.WriteFileAtomically(out, direction.UpcomingSheetName, sheet)
}

// DirectionRounds counts the batches of the current phase: how many sheets there are to choose
// from. A batch is what a director calls a round in a Swiss by rounds, a block in a GSL, a
// round in a bracket — the engine has no word for it, and needs none.
func (d *Service) DirectionRounds(ctx context.Context, tournamentID int64) (int, error) {
	dir, err := direction.Open(ctx, d.dirStore(), tournamentID)
	if err != nil {
		return 0, err
	}
	return dir.Rounds(), nil
}

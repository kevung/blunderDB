package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	tournoi "github.com/PileOfCells/backgammon-tournoi"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// The Rencontre's wall page (ADR-0056 §6): `<output_dir>/index.html` lists every table of the
// room whichever event plays on it, and links to each attached Tournament's own page, which this
// same gesture redirects into `<output_dir>/<slug>/` for as long as it stays attached. The
// Tournament's own output folder (direction.Record().OutputDir) is kept untouched in the row —
// detaching gives it back exactly, never asks the director to choose again.

var rencontreSlugNonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

// rencontreSlugs names the subfolder each member Tournament writes into, in id order so two
// members named alike get a stable, deterministic split (the second one's id appended).
func rencontreSlugs(ctx context.Context, stores storage.Stores, scope string, ids []int64) map[int64]string {
	sorted := slices.Clone(ids)
	slices.Sort(sorted)
	seen := map[string]bool{}
	out := map[int64]string{}
	for _, tid := range sorted {
		name := ""
		if t, err := stores.Tournaments().Get(ctx, scope, tid); err == nil {
			name = t.Name
		}
		slug := strings.Trim(rencontreSlugNonAlnum.ReplaceAllString(strings.ToLower(name), "-"), "-")
		if slug == "" {
			slug = fmt.Sprintf("epreuve-%d", tid)
		}
		if seen[slug] {
			slug = fmt.Sprintf("%s-%d", slug, tid)
		}
		seen[slug] = true
		out[tid] = slug
	}
	return out
}

// SetRencontreOutputDir remembers where the room's wall page is written, and writes it right
// away: choosing the folder is the whole gesture (D6.5's acceptance criterion), not the first of
// several.
func (d *Service) SetRencontreOutputDir(ctx context.Context, id int64, dir string) (*RencontreView, error) {
	defer d.lockRoom()()
	r, err := d.st.Rencontres().Get(ctx, d.scope, id)
	if err != nil {
		return nil, err
	}
	r.OutputDir = dir
	if err := d.st.Rencontres().Update(ctx, d.scope, *r); err != nil {
		return nil, err
	}
	if dir != "" {
		if _, err := d.WriteRencontrePage(ctx, id); err != nil {
			// A write failure is reported and never blocks the setting itself (ADR-0004): the
			// folder is remembered even when this first write fails (permissions, a removed
			// drive), and the next gesture in any member tries again.
			return d.GetRencontre(ctx, id)
		}
	}
	return d.GetRencontre(ctx, id)
}

// effectiveOutputDir is where a Tournament's own pages actually land: its Rencontre's
// `<output_dir>/<slug>/` while it is attached to one that has a folder, its own OutputDir
// otherwise. The Direction's own column is never rewritten by this — attaching and detaching
// change nothing there.
//
// A Rencontre folder that has gone away (an unplugged USB key) is left alone: the write that
// follows fails and reports it, exactly as it would for a Tournament's own folder — this must
// never paper over that by recreating the whole chain. Only the one subfolder an attached event
// writes into is created, and only once its parent is confirmed still there.
func (d *Service) effectiveOutputDir(ctx context.Context, tournamentID int64, ownDir string) string {
	rid, err := d.st.Rencontres().Of(ctx, d.scope, tournamentID)
	if err != nil || rid == 0 {
		return ownDir
	}
	r, err := d.st.Rencontres().Get(ctx, d.scope, rid)
	if err != nil || r.OutputDir == "" {
		return ownDir
	}
	slugs := rencontreSlugs(ctx, d.st, d.scope, r.TournamentIDs)
	dir := filepath.Join(r.OutputDir, slugs[tournamentID])
	if info, err := os.Stat(r.OutputDir); err == nil && info.IsDir() {
		_ = os.Mkdir(dir, 0o755) // best effort; a failure here surfaces at the write that follows
	}
	return dir
}

// afterRoomGesture regenerates the room's wall page — a gesture on the room is a gesture on
// every member too (ADR-0056 §6) — and returns the Rencontre as it now stands.
func (d *Service) afterRoomGesture(ctx context.Context, id int64) (*RencontreView, error) {
	_, _ = d.WriteRencontrePage(ctx, id)
	return d.GetRencontre(ctx, id)
}

// regenerateRencontrePage rewrites the room's wall page when tournamentID plays in one, best
// effort: a gesture in any member regenerates it (ADR-0056 §6), and a failure here must never be
// reported as a failure of the gesture that triggered it.
func (d *Service) regenerateRencontrePage(ctx context.Context, tournamentID int64) {
	rid, err := d.st.Rencontres().Of(ctx, d.scope, tournamentID)
	if err != nil || rid == 0 {
		return
	}
	_, _ = d.WriteRencontrePage(ctx, rid)
}

// wallPlayerName gives a player's name from a replayed state, falling back to the identifier.
func wallPlayerName(st *tournoi.State, id tournoi.PlayerID) string {
	if p := st.Players[id]; p != nil {
		return orID(p.Name, string(id))
	}
	return string(id)
}

// orID gives a player's name for the wall page, falling back to the identifier — a page with a
// hole in it is worse than one with an identifier in it (the same rule the per-event page
// follows).
func orID(name, id string) string {
	if name != "" {
		return name
	}
	return id
}

// RencontrePageHTML renders the room's wall page without writing it — what the CLI's
// `tournament page --rencontre` needs.
func (d *Service) RencontrePageHTML(ctx context.Context, id int64) (string, error) {
	r, err := d.st.Rencontres().Get(ctx, d.scope, id)
	if err != nil {
		return "", err
	}
	slugs := rencontreSlugs(ctx, d.st, d.scope, r.TournamentIDs)
	room := direction.Room{Tables: r.Tables}
	roomRead := false
	members := d.openMembers(ctx, r, 0)
	events := make([]direction.WallEvent, 0, len(members))
	var brackets []direction.WallBracket
	for _, m := range members {
		if m.dir == nil {
			events = append(events, direction.WallEvent{Name: m.name, Slug: slugs[m.tid]})
			continue
		}
		if cfg, err := m.dir.Config(); err == nil && !roomRead {
			room = direction.RoomOf(cfg)
			room.Tables = r.Tables
			roomRead = true
		}
		events = append(events, direction.WallEvent{Name: m.name, Slug: slugs[m.tid], Rounds: m.dir.Rounds()})
		if br := d.wallBracket(ctx, m.tid, m.name); br != nil {
			brackets = append(brackets, *br)
		}
	}
	// The wall page and the Hall read the room the same way: one line per table, the first
	// match on it, whatever its event.
	occupied := map[int]direction.WallTable{}
	for _, c := range d.hallOf(ctx, r, members, false).Cells {
		if c.MatchID == "" || c.NoTable || c.Table <= 0 {
			continue
		}
		if _, ok := occupied[c.Table]; ok {
			continue
		}
		occupied[c.Table] = direction.WallTable{Number: c.Table, Event: c.Event, A: orID(c.AName, c.A), B: orID(c.BName, c.B)}
	}
	tables := make([]direction.WallTable, 0, room.Tables)
	for t := 1; t <= room.Tables; t++ {
		if w, ok := occupied[t]; ok {
			tables = append(tables, w)
			continue
		}
		tables = append(tables, direction.WallTable{Number: t, Unavailable: slices.Contains(room.Unavailable, t)})
	}
	cat, lang := d.directionStrings(ctx)
	return direction.WallPage(direction.WallPageInput{Name: r.Name, Tables: tables, Events: events, Brackets: brackets, Now: time.Now()}, cat, lang), nil
}

// WriteRencontrePage rewrites the wall page and returns the file written. It writes nothing, and
// returns no error, when the Rencontre has no folder chosen — the same posture as a Direction's
// own display page.
func (d *Service) WriteRencontrePage(ctx context.Context, id int64) (string, error) {
	r, err := d.st.Rencontres().Get(ctx, d.scope, id)
	if err != nil {
		return "", err
	}
	if r.OutputDir == "" {
		return "", nil
	}
	page, err := d.RencontrePageHTML(ctx, id)
	if err != nil {
		return "", err
	}
	return direction.WriteWallPage(r.OutputDir, page)
}

// Package migrate copies a single-user blunderDB database (typically SQLite,
// scope "") into another backend (typically PostgreSQL) under a chosen tenant
// scope. It is backend-agnostic: Run takes two storage.Storage values, reads
// every family from the source and writes it to the destination through the
// same interface, remapping the auto-assigned primary keys (and the foreign
// keys that reference them) as it goes.
//
// Scope: it covers the core user data — positions, their analyses and comments,
// matches (games + moves), tournaments (+ match links) and collections (+
// membership). App-state families (anki decks/cards, filter library, search and
// command history, session state) are intentionally NOT migrated yet: they
// are lower-value for a data migration. Their per-tenant scoping is in place
// (each has a tenant-scoped table, session_state since schema 2.17.0). See
// tasks/headless/10-sqlite-to-postgres-tool.md.
package migrate

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// Options tunes a migration.
type Options struct {
	// DryRun reads and counts the source without writing to the destination.
	DryRun bool

	// OnConflict controls what happens when the destination scope already
	// holds data. "" (default) aborts with a clear error; "skip" proceeds and
	// lets position Zobrist de-duplication merge silently.
	OnConflict string

	// Progress, when set, is called once per family with the running totals.
	Progress func(Report)
}

// NotMigrated counts what the source holds and the migration deliberately
// leaves behind (see the package doc). A migration that says "done" while a
// user's Anki decks stayed on the old machine is a migration that lied by
// omission; Run fills this in from the source's own counts so the caller can
// name the numbers rather than a category.
type NotMigrated struct {
	AnkiCards int `json:"anki_cards"`
}

// Any reports whether anything at all was left behind.
func (n NotMigrated) Any() bool { return n.AnkiCards > 0 }

// Report tallies what was copied (or, in a dry run, what would be).
type Report struct {
	Positions   int `json:"positions"`
	Analyses    int `json:"analyses"`
	Comments    int `json:"comments"`
	Tournaments int `json:"tournaments"`
	Matches     int `json:"matches"`
	Games       int `json:"games"`
	Moves       int `json:"moves"`
	Collections int `json:"collections"`

	// NotMigrated is what the source holds and this tool does not copy.
	NotMigrated NotMigrated `json:"not_migrated"`
}

// Run migrates src into dst under scope. All destination writes happen inside a
// single transaction: on any error nothing is committed, so a failed migration
// leaves dst untouched and the user can simply re-run.
func Run(ctx context.Context, src, dst storage.Storage, scope string, opts Options) (Report, error) {
	var rep Report

	// A tenant is a positive decimal integer (ADR-0005, amendment
	// 2026-09-03); the PostgreSQL backend panics on anything else rather than
	// collapsing it onto tenant 0, so validate up front and fail cleanly.
	if _, err := storage.ParseTenant(scope); err != nil {
		return rep, fmt.Errorf("migrate: %w", err)
	}

	if !opts.DryRun && opts.OnConflict != "skip" {
		// Ask for the headline counts rather than listing one position: a
		// destination holding matches but no position — a partial run, or a
		// tenant used for something else — used to read as empty and be
		// written into (#240).
		counts, err := dst.Metadata().Counts(ctx, scope)
		if err != nil {
			return rep, fmt.Errorf("migrate: probe destination: %w", err)
		}
		if held := nonEmpty(counts); held != "" {
			return rep, fmt.Errorf("migrate: destination scope %q already holds %s; use --on-conflict skip to merge", scope, held)
		}
	}

	if opts.DryRun {
		return dryRun(ctx, src, scope)
	}

	tx, err := dst.BeginTx(ctx)
	if err != nil {
		return rep, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	m := &mover{ctx: ctx, src: src, dst: tx, scope: scope, opts: opts}
	if err := m.run(&rep); err != nil {
		return rep, err
	}

	if err := ctx.Err(); err != nil {
		return rep, err
	}
	if err := tx.Commit(); err != nil {
		return rep, fmt.Errorf("migrate: commit: %w", err)
	}
	committed = true
	rep.NotMigrated = leftBehind(ctx, src)
	return rep, nil
}

// mover holds the migration state: the source/destination handles and the
// per-table oldID→newID remaps.
type mover struct {
	ctx   context.Context
	src   storage.Storage
	dst   storage.Tx
	scope string
	opts  Options

	posID  map[int64]int64
	tourID map[int64]int64
}

func (m *mover) progress(rep *Report) {
	if m.opts.Progress != nil {
		m.opts.Progress(*rep)
	}
}

func (m *mover) run(rep *Report) error {
	m.posID = make(map[int64]int64)
	m.tourID = make(map[int64]int64)

	if err := m.copyPositions(rep); err != nil {
		return err
	}
	if err := m.copyTournaments(rep); err != nil {
		return err
	}
	if err := m.copyMatches(rep); err != nil {
		return err
	}
	if err := m.copyCollections(rep); err != nil {
		return err
	}
	return nil
}

// copyPositions copies every position (and its analysis + comments), building
// the position id remap that later families rely on. The source list is drained
// before issuing per-position follow-up reads to avoid grabbing a second
// connection inside the iterator.
func (m *mover) copyPositions(rep *Report) error {
	var positions []*domain.Position
	for p, err := range m.src.Positions().List(m.ctx, "", storage.ListOpts{}) {
		if err != nil {
			return fmt.Errorf("migrate: list positions: %w", err)
		}
		pc := *p
		positions = append(positions, &pc)
	}

	for _, p := range positions {
		oldID := p.ID
		pc := *p
		pc.ID = 0
		newID, err := m.dst.Positions().Save(m.ctx, m.scope, &pc)
		if err != nil {
			return fmt.Errorf("migrate: save position %d: %w", oldID, err)
		}
		m.posID[oldID] = newID
		rep.Positions++

		a, err := m.src.Analyses().Load(m.ctx, "", oldID)
		if err == nil {
			if err := m.dst.Analyses().Save(m.ctx, m.scope, newID, a); err != nil {
				return fmt.Errorf("migrate: save analysis for position %d: %w", oldID, err)
			}
			rep.Analyses++
		} else if !isNotFound(err) {
			return fmt.Errorf("migrate: load analysis for position %d: %w", oldID, err)
		}

		for c, err := range m.src.Comments().ByPosition(m.ctx, "", oldID) {
			if err != nil {
				return fmt.Errorf("migrate: read comments for position %d: %w", oldID, err)
			}
			if _, err := m.dst.Comments().Add(m.ctx, m.scope, newID, c.Text); err != nil {
				return fmt.Errorf("migrate: add comment for position %d: %w", oldID, err)
			}
			rep.Comments++
		}
	}
	m.progress(rep)
	return nil
}

func (m *mover) copyTournaments(rep *Report) error {
	var tournaments []*domain.Tournament
	for tr, err := range m.src.Tournaments().List(m.ctx, "") {
		if err != nil {
			return fmt.Errorf("migrate: list tournaments: %w", err)
		}
		tc := *tr
		tournaments = append(tournaments, &tc)
	}

	for _, tr := range tournaments {
		newID, err := m.dst.Tournaments().Create(m.ctx, m.scope, tr.Name, tr.Date, tr.Location)
		if err != nil {
			return fmt.Errorf("migrate: create tournament %q: %w", tr.Name, err)
		}
		if tr.Comment != "" {
			if err := m.dst.Tournaments().UpdateComment(m.ctx, m.scope, newID, tr.Comment); err != nil {
				return fmt.Errorf("migrate: set tournament comment: %w", err)
			}
		}
		m.tourID[tr.ID] = newID
		rep.Tournaments++
	}
	m.progress(rep)
	return nil
}

// copyMatches copies matches with their games and moves, remapping match→game→
// move foreign keys and each move's position id, then re-links the match to its
// (already-remapped) tournament.
func (m *mover) copyMatches(rep *Report) error {
	var matches []*domain.Match
	for mt, err := range m.src.Matches().List(m.ctx, "", storage.MatchListOpts{}) {
		if err != nil {
			return fmt.Errorf("migrate: list matches: %w", err)
		}
		mc := *mt
		matches = append(matches, &mc)
	}

	for _, mt := range matches {
		srcTournament := mt.TournamentID

		mc := *mt
		mc.ID = 0
		mc.TournamentID = nil // re-linked below via the tournament remap
		newMatchID, err := m.dst.Matches().Save(m.ctx, m.scope, &mc)
		if err != nil {
			return fmt.Errorf("migrate: save match %d: %w", mt.ID, err)
		}
		rep.Matches++

		if err := m.copyGames(rep, mt.ID, newMatchID); err != nil {
			return err
		}

		if srcTournament != nil {
			if newTID, ok := m.tourID[*srcTournament]; ok {
				if err := m.dst.Tournaments().AddMatch(m.ctx, m.scope, newTID, newMatchID); err != nil {
					return fmt.Errorf("migrate: link match %d to tournament: %w", mt.ID, err)
				}
			}
		}
	}
	m.progress(rep)
	return nil
}

func (m *mover) copyGames(rep *Report, oldMatchID, newMatchID int64) error {
	var games []*domain.Game
	for g, err := range m.src.Matches().Games(m.ctx, "", oldMatchID) {
		if err != nil {
			return fmt.Errorf("migrate: list games of match %d: %w", oldMatchID, err)
		}
		gc := *g
		games = append(games, &gc)
	}

	for _, g := range games {
		gc := *g
		gc.ID = 0
		gc.MatchID = newMatchID
		newGameID, err := m.dst.Matches().CreateGame(m.ctx, m.scope, &gc)
		if err != nil {
			return fmt.Errorf("migrate: create game %d: %w", g.ID, err)
		}
		rep.Games++

		if err := m.copyMoves(rep, g.ID, newGameID); err != nil {
			return err
		}
	}
	return nil
}

func (m *mover) copyMoves(rep *Report, oldGameID, newGameID int64) error {
	var moves []*domain.Move
	for mv, err := range m.src.Matches().Moves(m.ctx, "", oldGameID) {
		if err != nil {
			return fmt.Errorf("migrate: list moves of game %d: %w", oldGameID, err)
		}
		mc := *mv
		moves = append(moves, &mc)
	}

	for _, mv := range moves {
		mc := *mv
		mc.ID = 0
		mc.GameID = newGameID
		if mv.PositionID > 0 {
			newPos, ok := m.posID[mv.PositionID]
			if !ok {
				return fmt.Errorf("migrate: move %d references unknown position %d", mv.ID, mv.PositionID)
			}
			mc.PositionID = newPos
		}
		if _, err := m.dst.Matches().CreateMove(m.ctx, m.scope, &mc); err != nil {
			return fmt.Errorf("migrate: create move %d: %w", mv.ID, err)
		}
		rep.Moves++
	}
	return nil
}

func (m *mover) copyCollections(rep *Report) error {
	type coll struct {
		id   int64
		name string
		desc string
	}
	var colls []coll
	for c, err := range m.src.Collections().List(m.ctx, "") {
		if err != nil {
			return fmt.Errorf("migrate: list collections: %w", err)
		}
		colls = append(colls, coll{c.ID, c.Name, c.Description})
	}

	for _, c := range colls {
		newID, err := m.dst.Collections().Create(m.ctx, m.scope, c.name, c.desc)
		if err != nil {
			return fmt.Errorf("migrate: create collection %q: %w", c.name, err)
		}
		rep.Collections++

		// Drain the membership before mapping ids (avoid nested source reads).
		var posIDs []int64
		for p, err := range m.src.Collections().Positions(m.ctx, "", c.id) {
			if err != nil {
				return fmt.Errorf("migrate: list collection %d positions: %w", c.id, err)
			}
			if newPos, ok := m.posID[p.ID]; ok {
				posIDs = append(posIDs, newPos)
			}
		}
		if len(posIDs) > 0 {
			if err := m.dst.Collections().AddPositions(m.ctx, m.scope, newID, posIDs); err != nil {
				return fmt.Errorf("migrate: add positions to collection %q: %w", c.name, err)
			}
		}
	}
	m.progress(rep)
	return nil
}

// dryRun counts what a migration would copy without touching the destination.
func dryRun(ctx context.Context, src storage.Storage, scope string) (Report, error) {
	_ = scope
	var rep Report
	var posIDs []int64
	for p, err := range src.Positions().List(ctx, "", storage.ListOpts{}) {
		if err != nil {
			return rep, err
		}
		posIDs = append(posIDs, p.ID)
		rep.Positions++
	}
	for _, id := range posIDs {
		if _, err := src.Analyses().Load(ctx, "", id); err == nil {
			rep.Analyses++
		} else if !isNotFound(err) {
			return rep, err
		}
		for c, err := range src.Comments().ByPosition(ctx, "", id) {
			if err != nil {
				return rep, err
			}
			_ = c
			rep.Comments++
		}
	}
	for _, err := range src.Tournaments().List(ctx, "") {
		if err != nil {
			return rep, err
		}
		rep.Tournaments++
	}
	matchIDs := []int64{}
	for mt, err := range src.Matches().List(ctx, "", storage.MatchListOpts{}) {
		if err != nil {
			return rep, err
		}
		matchIDs = append(matchIDs, mt.ID)
		rep.Matches++
	}
	for _, mid := range matchIDs {
		gameIDs := []int64{}
		for g, err := range src.Matches().Games(ctx, "", mid) {
			if err != nil {
				return rep, err
			}
			gameIDs = append(gameIDs, g.ID)
			rep.Games++
		}
		for _, gid := range gameIDs {
			for _, err := range src.Matches().Moves(ctx, "", gid) {
				if err != nil {
					return rep, err
				}
				rep.Moves++
			}
		}
	}
	for _, err := range src.Collections().List(ctx, "") {
		if err != nil {
			return rep, err
		}
		rep.Collections++
	}
	rep.NotMigrated = leftBehind(ctx, src)
	return rep, nil
}

// leftBehind counts, in the source, what this tool does not copy. Errors are
// swallowed on purpose: a count that cannot be read must not fail a migration
// that has otherwise succeeded — the warning is a courtesy, and its absence is
// the worst it can cost.
func leftBehind(ctx context.Context, src storage.Storage) NotMigrated {
	counts, err := src.Metadata().Counts(ctx, "")
	if err != nil {
		return NotMigrated{}
	}
	return NotMigrated{AnkiCards: counts.AnkiCards}
}

// isNotFound reports whether err is the storage "not found" sentinel.
func isNotFound(err error) bool {
	return errors.Is(err, storage.ErrNotFound)
}

// nonEmpty describes what a destination already holds, or "" when it holds
// nothing. The message names the families rather than saying "not empty": the
// user's next question is always which ones.
func nonEmpty(c storage.Counts) string {
	var parts []string
	for _, f := range []struct {
		n     int
		label string
	}{
		{c.Positions, "position"},
		{c.Analyses, "analysis"},
		{c.Matches, "match"},
		{c.AnkiCards, "Anki card"},
	} {
		if f.n > 0 {
			parts = append(parts, plural(f.n, f.label))
		}
	}
	return strings.Join(parts, ", ")
}

// plural renders "1 match" / "3 matches" without a table of irregulars: the
// four labels above are all regular but for "analysis".
func plural(n int, label string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, label)
	}
	if strings.HasSuffix(label, "is") {
		return fmt.Sprintf("%d %ses", n, strings.TrimSuffix(label, "is"))
	}
	return fmt.Sprintf("%d %ss", n, label)
}

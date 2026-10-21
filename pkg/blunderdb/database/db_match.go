package database

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// GetAllMatches returns all matches from the database
func (d *Database) GetAllMatches() ([]Match, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var matches []Match
	for m, err := range d.store.Matches().List(context.Background(), "", storage.MatchListOpts{}) {
		if err != nil {
			return nil, err
		}
		matches = append(matches, *m)
	}

	if err := d.applyMatchBadges(matches); err != nil {
		slog.Warn("computing match stats", "err", err)
	}

	return matches, nil
}

// ListMatches returns one page of matches, filtered and ordered in SQL per
// opts, with the PR/MWC badges of that page only. A zero opts is GetAllMatches.
func (d *Database) ListMatches(opts storage.MatchListOpts) ([]Match, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	matches := []Match{}
	for m, err := range d.store.Matches().List(context.Background(), "", opts) {
		if err != nil {
			return nil, err
		}
		matches = append(matches, *m)
	}
	if err := d.applyMatchBadges(matches); err != nil {
		slog.Warn("computing match stats", "err", err)
	}
	return matches, nil
}

// CountMatches returns how many matches satisfy the filters of opts (the page
// bounds are ignored): the total behind a ListMatches page.
func (d *Database) CountMatches(opts storage.MatchListOpts) (int, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.store.Matches().Count(context.Background(), "", opts)
}

// GetMatchByID returns a specific match by ID, including the name of the
// tournament it is assigned to (empty when unassigned) so callers can display
// the full provenance without a second lookup, and the PR/MWC badges a list
// row shows, so that a row re-read after an edit shows them as they now are.
func (d *Database) GetMatchByID(matchID int64) (*Match, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	m, err := d.store.Matches().Get(context.Background(), "", matchID)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, fmt.Errorf("match not found")
	}
	if err != nil {
		return nil, err
	}
	one := []Match{*m}
	if err := d.applyMatchBadges(one); err != nil {
		slog.Warn("computing match stats", "err", err)
	}
	return &one[0], nil
}

// GetPositionProvenance returns the matches that reference the given position
// through their move graph (move → game → match) — the position's provenance.
// Positions are deduplicated by Zobrist hash, so several matches can reach
// one; the list is empty for an individually imported position. Each match
// carries its tournament name; in the match list's order, most recent first
// (storage.MatchStore.MatchesByPosition, then List).
func (d *Database) GetPositionProvenance(positionID int64) ([]Match, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	ctx := context.Background()
	ids, err := d.store.Matches().MatchesByPosition(ctx, "", positionID)
	if err != nil || len(ids) == 0 {
		return nil, err
	}
	var matches []Match
	for m, err := range d.store.Matches().List(ctx, "", storage.MatchListOpts{IDs: ids}) {
		if err != nil {
			return nil, err
		}
		matches = append(matches, *m)
	}
	return matches, nil
}

// SaveLastVisitedPosition saves the last visited position index for a match
// (storage.MatchStore.SetLastVisitedPosition).
func (d *Database) SaveLastVisitedPosition(matchID int64, positionIndex int) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.store.Matches().SetLastVisitedPosition(context.Background(), "", matchID, positionIndex)
}

// GetLastVisitedMatch returns the most recently visited match or, when none
// has been visited, the match the list shows first
// (storage.MatchStore.LastVisited). The "no matches" error text is read by
// the GUI.
func (d *Database) GetLastVisitedMatch() (*Match, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	m, err := d.store.Matches().LastVisited(context.Background(), "")
	if errors.Is(err, storage.ErrNotFound) {
		return nil, fmt.Errorf("no matches in database")
	}
	return m, err
}

// GetGamesByMatch returns all games in a match, by game number
// (storage.MatchStore.Games).
func (d *Database) GetGamesByMatch(matchID int64) ([]Game, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var games []Game
	for g, err := range d.store.Matches().Games(context.Background(), "", matchID) {
		if err != nil {
			return nil, err
		}
		games = append(games, *g)
	}
	return games, nil
}

// GetMovesByGame returns all moves in a game, by move number, as the match
// recorded them, without the cost of the played move: its one caller, the
// CSV export, walks every move of the library and writes no cost
// (storage.MatchStore.RecordedMoves).
func (d *Database) GetMovesByGame(gameID int64) ([]Move, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var moves []Move
	for mv, err := range d.store.Matches().RecordedMoves(context.Background(), "", gameID) {
		if err != nil {
			return nil, err
		}
		moves = append(moves, *mv)
	}
	return moves, nil
}

// DeleteMatch deletes a match and all associated games, moves and move
// analyses, then the positions nothing else holds any more
// (storage.MatchStore.DeleteCascade, whose retention predicate decides which).
func (d *Database) DeleteMatch(matchID int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.store.Matches().DeleteCascade(context.Background(), "", matchID)
}

// GetMatchMovePositions returns all positions from a match in chronological order
// Positions are returned as they were stored (from player on roll POV)
// The frontend is responsible for mirroring display if needed
func (d *Database) GetMatchMovePositions(matchID int64) ([]MatchMovePosition, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var movePositions []MatchMovePosition
	for mp, err := range d.store.Matches().MovePositions(context.Background(), "", matchID) {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, fmt.Errorf("match not found: %w", err)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to query moves: %w", err)
		}
		movePositions = append(movePositions, *mp)
	}
	return movePositions, nil
}

// GetDatabaseStats returns statistics about the database
//
// Every number comes from the store's Counts, read once: the table counts
// were counted here a second time before, which on a 15 M-row library
// doubled the wait of `info`.
func (d *Database) GetDatabaseStats() (map[string]interface{}, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Blunders are counted at the library's threshold (ADR-0046), by the
	// store's rule — the same as the status bar link's search, a
	// multi-played Position scored by its largest play. Not restated here,
	// so it cannot drift. It counts POSITIONS, not PR decisions: "how much
	// to look at".
	c, err := d.store.Metadata().Counts(context.Background(), "")
	if err == nil {
		return map[string]interface{}{
			"position_count": int64(c.Positions),
			"analysis_count": int64(c.Analyses),
			"match_count":    int64(c.Matches),
			"game_count":     int64(c.Games),
			"move_count":     int64(c.Moves),
			"blunder_count":  int64(c.Blunders),
		}, nil
	}
	// The blunder count is the fragile part (it reads every analysis): its
	// failure must not take the whole of `info` down. The tables are counted
	// one by one and the blunders reported as 0.
	slog.Warn("database stats: counting at once failed; counting table by table, blunders at 0", "err", err)
	stats := map[string]interface{}{"blunder_count": int64(0)}
	for _, t := range []struct {
		table, key string
		required   bool
	}{
		{"position", "position_count", true}, {"analysis", "analysis_count", true},
		{"match", "match_count", false}, {"game", "game_count", false}, {"move", "move_count", false},
	} {
		var n int64
		if err := d.db.QueryRow(`SELECT COUNT(*) FROM ` + t.table).Scan(&n); err != nil {
			if t.required {
				return nil, err
			}
			slog.Warn("database stats: counting a table", "table", t.table, "err", err)
		}
		stats[t.key] = n
	}
	return stats, nil
}

// EstimateExactBelow is the highest table id up to which the library counter
// counts instead of estimating: under it a COUNT is instantaneous.
const EstimateExactBelow = 200_000

// GetDatabaseStatsEstimate is GetDatabaseStats for the status bar: it never
// scans a large table. Counts of tables beyond EstimateExactBelow rows are
// estimates, named in "approximate". "blunder_count" is there when the
// positions are counted exactly, or when the cached count still holds for the
// library's threshold; otherwise it is absent, never estimated, and a
// recount starts in the background (startBlunderRecount) to fill the cache.
// GetDatabaseStats is the exact form.
func (d *Database) GetDatabaseStatsEstimate() (map[string]interface{}, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	est, err := d.store.Metadata().EstimatedCounts(context.Background(), "", EstimateExactBelow)
	if err != nil {
		return nil, err
	}
	stats := map[string]interface{}{
		"position_count": int64(est.Positions),
		"analysis_count": int64(est.Analyses),
		"match_count":    int64(est.Matches),
		"game_count":     int64(est.Games),
		"move_count":     int64(est.Moves),
		"approximate":    est.Approximate,
	}
	if est.BlundersKnown {
		stats["blunder_count"] = int64(est.Blunders)
	} else {
		d.startBlunderRecount()
	}
	return stats, nil
}

// GetLibrarySettings returns the library's error and blunder thresholds
// (ADR-0046). A library that has never set them reads the defaults.
func (d *Database) GetLibrarySettings() (storage.LibrarySettings, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.store.LibrarySettings().Load(context.Background(), "")
}

// SaveLibrarySettings records the library's thresholds, refusing a pair whose
// error threshold sits above its blunder threshold — every Blunder is an
// Error, so the inverted pair names a set that cannot exist.
func (d *Database) SaveLibrarySettings(settings storage.LibrarySettings) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.store.LibrarySettings().Save(context.Background(), "", settings)
}

// SetMatchVideoSource attaches the video a match was transcribed from (an
// http(s) URL or a local path), or detaches it when source is empty.
func (d *Database) SetMatchVideoSource(matchID int64, source string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}
	return d.store.Matches().SetVideoSource(context.Background(), "", matchID, strings.TrimSpace(source))
}

// UpdateMatch updates editable metadata for a match (player names and date)
// and re-dates the positions it reaches (storage.MatchStore.Update).
// matchDate should be an empty string or a date string parseable by time.Parse ("2006-01-02").
func (d *Database) UpdateMatch(matchID int64, player1Name, player2Name, matchDate string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}
	if matchDate != "" {
		if _, err := time.Parse("2006-01-02", matchDate); err != nil {
			return fmt.Errorf("invalid date format: %w", err)
		}
	}
	return d.store.Matches().Update(context.Background(), "", matchID,
		strings.TrimSpace(player1Name), strings.TrimSpace(player2Name), matchDate)
}

// SwapMatchPlayers swaps the two players in a match: player1 becomes player2
// and vice versa (storage.MatchStore.SwapPlayers).
func (d *Database) SwapMatchPlayers(matchID int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}
	return d.store.Matches().SwapPlayers(context.Background(), "", matchID)
}

// MergePlayers makes every other name of names an alias of canonicalName
// (storage.AliasStore): the matches keep the names their files wrote, the
// stats, the players table and the search read them as one person, and every
// later import stores canonicalName. names must be non-empty and
// canonicalName must not be blank.
func (d *Database) MergePlayers(names []string, canonicalName string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}
	return d.store.Matches().MergePlayers(context.Background(), "", names, canonicalName)
}

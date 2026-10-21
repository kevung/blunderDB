package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	sqlitedriver "modernc.org/sqlite"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// Tournament methods take d.mu and delegate to d.store.Tournaments() with the
// wrapper's implicit scope.

// CreateTournament creates a new tournament
func (d *Database) CreateTournament(name string, date string, location string) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return 0, fmt.Errorf("no database is currently open")
	}
	return d.store.Tournaments().Create(context.Background(), "", name, date, location)
}

// matchStatsFillLot is how many matches one hold of Database.mu fills.
var matchStatsFillLot = 200

// fillBusyRetries bounds how often in a row one lot is redone after SQLITE_BUSY.
const fillBusyRetries = 20

// isSQLiteBusy reports whether err is SQLite's SQLITE_BUSY, extended codes included.
func isSQLiteBusy(err error) bool {
	var e *sqlitedriver.Error
	return errors.As(err, &e) && e.Code()&0xff == 5
}

// sleepCtx waits for d or for ctx's end, whichever comes first.
func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// betweenFillLots runs while prefillMatchStats holds no lock, after each lot; tests pause the fill there.
var betweenFillLots = func() {}

// prefillMatchStats computes the match_stats rows the tournament badges read,
// a lot at a time, releasing Database.mu between lots. The badges would fill
// them all inside one read hold, and a writer queued on Lock (an import, a
// tournament edit) then keeps every later reader, the match list's page
// among them, waiting for the whole library. Best effort: a backend that
// cannot fill, or a read-only database, leaves the badges to their own path.
// It is the CLI's path; the desktop fills in the background from the open on
// (FillMatchStatsInBackground).
func (d *Database) prefillMatchStats() {
	_ = d.fillMatchStatsLots(context.Background(), nil, nil)
}

// fillMatchStatsLots is the lot loop under prefillMatchStats and the
// background fill. Each lot holds Database.mu for reading; keep, nil for
// none, is asked under that hold whether the lot may still write (the open
// file is the one the fill started on). progress receives the matches still
// missing after each lot. It returns ctx's error once ctx ends, and
// ErrDatabaseChanged once keep refuses.
func (d *Database) fillMatchStatsLots(ctx context.Context, keep func() bool, progress func(missing int)) error {
	busy := 0
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		d.mu.RLock()
		if d.db == nil || (keep != nil && !keep()) {
			d.mu.RUnlock()
			return ErrDatabaseChanged
		}
		missing, err := d.store.Stats().MissingMatchStats(ctx, "")
		if err == nil && len(missing) > 0 {
			err = d.store.Stats().RefreshMatchStats(ctx, "", missing[:min(matchStatsFillLot, len(missing))])
		}
		d.mu.RUnlock()
		if err != nil && isSQLiteBusy(err) && busy < fillBusyRetries {
			// Another writer (an import, a page filling its own matches) took
			// the file between this lot's read and its write: the lot is
			// redone, outside the lock.
			busy++
			if err := sleepCtx(ctx, time.Duration(busy)*50*time.Millisecond); err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		busy = 0
		if progress != nil {
			progress(max(len(missing)-matchStatsFillLot, 0))
		}
		if len(missing) == 0 {
			return nil
		}
		betweenFillLots()
	}
}

// GetAllTournaments returns all tournaments with their match counts
//
// While the background fill runs, the list comes back at once without the
// badges, which would otherwise fill the rest of the library in one read
// hold; the desktop reads it again once the fill ends.
func (d *Database) GetAllTournaments() ([]Tournament, error) {
	filling := d.MatchStatsFilling()
	if !filling {
		d.prefillMatchStats()
	}
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}

	var tournaments []Tournament
	for t, err := range d.store.Tournaments().List(context.Background(), "", storage.ListOpts{}) {
		if err != nil {
			return nil, err
		}
		tournaments = append(tournaments, *t)
	}

	if filling {
		return tournaments, nil
	}
	if err := d.applyTournamentBadges(tournaments); err != nil {
		// Non-fatal
		_ = err
	}

	return tournaments, nil
}

// UpdateTournament updates a tournament's details
func (d *Database) UpdateTournament(id int64, name string, date string, location string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}
	return d.store.Tournaments().Update(context.Background(), "", id, name, date, location)
}

// DeleteTournament deletes a tournament (matches are unlinked, not deleted)
func (d *Database) DeleteTournament(id int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}
	return d.store.Tournaments().Delete(context.Background(), "", id)
}

// AddMatchToTournament adds a match to a tournament
func (d *Database) AddMatchToTournament(tournamentID int64, matchID int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}
	return d.store.Tournaments().AddMatch(context.Background(), "", tournamentID, matchID)
}

// RemoveMatchFromTournament removes a match from a tournament
func (d *Database) RemoveMatchFromTournament(matchID int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}
	return d.store.Tournaments().RemoveMatch(context.Background(), "", matchID)
}

// UpdateMatchComment updates the comment of a match, signed like a position
// comment by the author SetCommentAuthor named.
func (d *Database) UpdateMatchComment(matchID int64, comment string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}
	return d.store.Matches().UpdateComment(d.authorCtx(), "", matchID, comment)
}

// UpdateTournamentComment updates the comment of a tournament
func (d *Database) UpdateTournamentComment(tournamentID int64, comment string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}
	return d.store.Tournaments().UpdateComment(context.Background(), "", tournamentID, comment)
}

// ReorderTournamentMatches sets the sort order for matches in a tournament.
// matchIDs should be in the desired order.
func (d *Database) ReorderTournamentMatches(tournamentID int64, matchIDs []int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}
	return d.store.Tournaments().ReorderMatches(context.Background(), "", tournamentID, matchIDs)
}

// SetMatchTournamentByName assigns a match to a tournament by name.
// If tournamentName is empty, the match is unlinked from any tournament.
// If no tournament with that name exists, one is created.
func (d *Database) SetMatchTournamentByName(matchID int64, tournamentName string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}
	return d.store.Tournaments().SetMatchByName(context.Background(), "", matchID, tournamentName)
}

// GetTournamentMatches returns all matches in a tournament
func (d *Database) GetTournamentMatches(tournamentID int64) ([]Match, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}

	var matches []Match
	for m, err := range d.store.Tournaments().Matches(context.Background(), "", tournamentID) {
		if err != nil {
			return nil, err
		}
		matches = append(matches, *m)
	}

	if err := d.applyMatchBadges(matches); err != nil {
		// Non-fatal: log but continue without stats
		_ = err
	}

	return matches, nil
}

// GetMatchTournament returns the tournament a match belongs to (if any).
// A match that exists but is not in any tournament yields (nil, nil), as does
// an unknown match id: the backend reports both as storage.ErrNotFound and the
// GUI treats "no tournament" as a plain absence, not an error.
func (d *Database) GetMatchTournament(matchID int64) (*Tournament, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}

	t, err := d.store.Tournaments().TournamentOf(context.Background(), "", matchID)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, nil
	}
	return t, err
}

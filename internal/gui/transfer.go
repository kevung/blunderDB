package gui

import (
	"context"
	"errors"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// TransferBatch is what one library group gives to a transfer: rows named by
// their ids in that group's library. A drag carries one batch, the basket one
// per group it was filled from.
type TransferBatch struct {
	Group         int     `json:"group"`
	PositionIDs   []int64 `json:"positionIDs"`
	MatchIDs      []int64 `json:"matchIDs"`
	CollectionIDs []int64 `json:"collectionIDs"`
	LessonIDs     []int64 `json:"lessonIDs"`
}

func (b TransferBatch) empty() bool {
	return len(b.PositionIDs)+len(b.MatchIDs)+len(b.CollectionIDs)+len(b.LessonIDs) == 0
}

// options is the export selection of the batch, with what travels beside each
// row: its analyses, comments and played moves, a collection's positions, a
// match's transcription and aliases. WholeLibrary stays off, so that the ids
// alone say which rows leave.
func (b TransferBatch) options() database.ExportOptions {
	return database.ExportOptions{
		PositionIDs:                b.PositionIDs,
		IncludeAnalysis:            true,
		IncludeComments:            true,
		IncludePlayedMoves:         true,
		IncludeMatches:             len(b.MatchIDs) > 0,
		MatchIDs:                   b.MatchIDs,
		IncludeTranscriptions:      len(b.MatchIDs) > 0,
		IncludeAliases:             len(b.MatchIDs) > 0,
		IncludeCollections:         len(b.CollectionIDs) > 0,
		CollectionIDs:              b.CollectionIDs,
		IncludeCollectionPositions: len(b.CollectionIDs) > 0,
		IncludeLessons:             len(b.LessonIDs) > 0,
		LessonIDs:                  b.LessonIDs,
	}
}

// TransferSummary is the account of a transfer, told on the destination's
// header and in the status bar once that group is seen.
type TransferSummary struct {
	Group int `json:"group"`
	// Positions counts the positions the destination did not hold before,
	// -1 when they could not be counted; AlreadyHeld the matches it held
	// already.
	Positions   int      `json:"positions"`
	Matches     int      `json:"matches"`
	AlreadyHeld int      `json:"alreadyHeld"`
	Collections int      `json:"collections"`
	Lessons     int      `json:"lessons"`
	Errors      []string `json:"errors"`
	Cancelled   bool     `json:"cancelled"`
}

// ErrLibraryGroupLocked refuses a transfer into a group its user locked.
var ErrLibraryGroupLocked = errors.New("this library group is locked: nothing can be poured into it")

// SetLibraryGroupLocked locks or unlocks group id against transfers into it.
// The lock is the desktop's, held in memory: no library records it.
func (a *App) SetLibraryGroupLocked(id int, locked bool) error {
	a.libs.mu.Lock()
	defer a.libs.mu.Unlock()
	a.libs.init()
	if id != a.libs.active && a.libs.parked[id] == nil {
		return fmt.Errorf("no library group %d", id)
	}
	if locked {
		a.libs.locked[id] = true
	} else {
		delete(a.libs.locked, id)
	}
	return nil
}

// TransferToGroup copies the rows each batch names, from its group's library,
// into the library of group dst, through Database.TransferTo: the export's
// selection merged by the native import, a row dst already holds counted, not
// added. It runs on handles borrowed from both libraries, so it keeps writing
// dst whichever group comes on screen; closing either group stops it.
// A batch of dst itself has nothing to copy and is skipped.
func (a *App) TransferToGroup(dst int, batches []TransferBatch) (TransferSummary, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a.libs.mu.Lock()
	a.libs.init()
	if dst == 0 {
		dst = a.libs.active
	}
	if a.libs.locked[dst] {
		a.libs.mu.Unlock()
		return TransferSummary{Group: dst}, ErrLibraryGroupLocked
	}
	_, to, err := a.borrowLibraryLocked(dst)
	if err != nil {
		a.libs.mu.Unlock()
		return TransferSummary{Group: dst}, err
	}
	defer to.Close() // lets the borrowed library go; closes nothing
	if to.IsReadOnly() {
		a.libs.mu.Unlock()
		return TransferSummary{Group: dst}, errors.New("the destination library is open read-only")
	}
	type source struct {
		batch TransferBatch
		from  *database.Database
	}
	var sources []source
	release := []func(){a.imports.addCancel(dst, to, cancel)}
	defer func() {
		for _, r := range release {
			r()
		}
	}()
	for _, b := range batches {
		if b.Group == dst || b.empty() {
			continue
		}
		_, from, err := a.borrowLibraryLocked(b.Group)
		if err != nil {
			a.libs.mu.Unlock()
			return TransferSummary{Group: dst}, err
		}
		defer from.Close()
		release = append(release, a.imports.addCancel(b.Group, from, cancel))
		sources = append(sources, source{b, from})
	}
	a.libs.mu.Unlock()

	sum := TransferSummary{Group: dst, Errors: []string{}}
	// The import counts every position it merged, held or not: the account
	// tells what dst did not hold before, by difference. Another import
	// running into dst meanwhile is counted with it; without a first count
	// the account says nothing rather than a false number.
	held, heldErr := to.CountPositions()
	for _, s := range sources {
		rep, err := s.from.TransferTo(ctx, to, s.batch.options())
		im := rep.Imported
		sum.Matches += im.Matches
		sum.AlreadyHeld += im.SkippedDuplicates
		sum.Collections += im.Collections
		sum.Lessons += im.Lessons
		if errors.Is(err, context.Canceled) {
			sum.Cancelled = true
			break
		}
		if err != nil {
			sum.Errors = append(sum.Errors, err.Error())
		}
	}
	sum.Positions = -1
	if now, err := to.CountPositions(); heldErr == nil && err == nil {
		sum.Positions = max(now-held, 0)
	}
	if sum.Positions != 0 || sum.Matches > 0 {
		to.RefreshSearchStatistics()
	}
	return sum, nil
}

// PositionCopy is the account of one position carried from a split-screen
// pane onto the other: the transfer's summary, and the position's id in the
// destination when this copy added it (0 when dst held it already), the only
// case the copy can be taken back.
type PositionCopy struct {
	TransferSummary
	Created int64 `json:"created"`
}

// CopyPositionToGroup copies position id of group src into the library of
// group dst through TransferToGroup, and says whether dst held it before, by
// its Zobrist hash: a position is one per hash in a library.
func (a *App) CopyPositionToGroup(src int, id int64, dst int) (PositionCopy, error) {
	var hash uint64
	if err := a.readGroup(src, func(d *database.Database) error {
		p, err := d.LoadPosition(int(id))
		if err != nil {
			return err
		}
		hash = storage.ZobristOf(p)
		return nil
	}); err != nil {
		return PositionCopy{TransferSummary: TransferSummary{Group: dst}}, err
	}
	_, heldBefore, err := a.positionInGroup(dst, hash)
	if err != nil {
		return PositionCopy{TransferSummary: TransferSummary{Group: dst}}, err
	}
	sum, err := a.TransferToGroup(dst, []TransferBatch{{Group: src, PositionIDs: []int64{id}}})
	out := PositionCopy{TransferSummary: sum}
	if err != nil || heldBefore || sum.Cancelled || len(sum.Errors) > 0 {
		return out, err
	}
	if created, found, err := a.positionInGroup(dst, hash); err == nil && found {
		out.Created = created
	}
	return out, nil
}

// UndoPositionCopy takes back a copy CopyPositionToGroup made: the position
// it added to dst leaves dst, with what came with it.
func (a *App) UndoPositionCopy(dst int, id int64) error {
	if id <= 0 {
		return errors.New("nothing to take back")
	}
	a.libs.mu.Lock()
	locked := a.libs.locked[dst]
	a.libs.mu.Unlock()
	if locked {
		return ErrLibraryGroupLocked
	}
	return a.readGroup(dst, func(d *database.Database) error {
		if d.IsReadOnly() {
			return errors.New("the library is open read-only")
		}
		return d.DeletePosition(id)
	})
}

// positionInGroup looks up the position stored under hash in group's library.
func (a *App) positionInGroup(group int, hash uint64) (int64, bool, error) {
	var id int64
	var found bool
	err := a.readGroup(group, func(d *database.Database) error {
		st, _ := database.CurrentStore(d)
		if st == nil {
			return errors.New("no database open")
		}
		var err error
		id, found, err = st.Positions().Exists(context.Background(), "", hash)
		return err
	})
	return id, found, err
}

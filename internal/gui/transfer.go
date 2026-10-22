package gui

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
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
	var groups []int
	for _, b := range batches {
		if !b.empty() {
			groups = append(groups, b.Group)
		}
	}
	ends, err := a.borrowTransferEnds(dst, groups)
	if err != nil {
		return TransferSummary{Group: ends.dst}, err
	}
	defer ends.release()
	dst, to, ctx := ends.dst, ends.to, ends.ctx
	type source struct {
		batch TransferBatch
		from  *database.Database
	}
	var sources []source
	for _, b := range batches {
		if from := ends.from[b.Group]; from != nil && !b.empty() {
			sources = append(sources, source{b, from})
		}
	}

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

// transferEnds are the libraries of a transfer, borrowed: the destination's
// and one per source group, so the transfer keeps writing whichever group
// comes on screen; closing either group stops it through ctx.
type transferEnds struct {
	dst     int
	to      *database.Database
	from    map[int]*database.Database
	ctx     context.Context
	release func()
}

// borrowTransferEnds borrows dst's library (0: the active group) for writing
// and the libraries of srcs, dst itself left out, refusing a locked or
// read-only destination.
func (a *App) borrowTransferEnds(dst int, srcs []int) (transferEnds, error) {
	ctx, cancel := context.WithCancel(context.Background())
	var release []func()
	ends := transferEnds{ctx: ctx, from: map[int]*database.Database{}, release: func() {
		for i := len(release) - 1; i >= 0; i-- {
			release[i]()
		}
		cancel()
	}}
	a.libs.mu.Lock()
	defer a.libs.mu.Unlock()
	a.libs.init()
	if dst == 0 {
		dst = a.libs.active
	}
	ends.dst = dst
	fail := func(err error) (transferEnds, error) {
		ends.release()
		return transferEnds{dst: dst}, err
	}
	if a.libs.locked[dst] {
		return fail(ErrLibraryGroupLocked)
	}
	_, to, err := a.borrowLibraryLocked(dst)
	if err != nil {
		return fail(err)
	}
	release = append(release, func() { to.Close() }) // lets the borrowed library go; closes nothing
	if to.IsReadOnly() {
		return fail(errors.New("the destination library is open read-only"))
	}
	ends.to = to
	release = append(release, a.imports.addCancel(dst, to, cancel))
	for _, g := range srcs {
		if g == dst || ends.from[g] != nil {
			continue
		}
		_, from, err := a.borrowLibraryLocked(g)
		if err != nil {
			return fail(err)
		}
		release = append(release, func() { from.Close() })
		release = append(release, a.imports.addCancel(g, from, cancel))
		ends.from[g] = from
	}
	return ends, nil
}

// PositionCopy is the account of one position carried from a split-screen
// pane onto the other: the transfer's summary, and, when this copy added the
// position to the destination, the receipt that takes it back
// (UndoPositionCopy); 0 when the destination held it already.
type PositionCopy struct {
	TransferSummary
	Receipt int `json:"receipt"`
}

// paneCopies keeps the last copy between split panes the desktop can take
// back; its receipt names the file the copy went into.
type paneCopies struct {
	mu   sync.Mutex
	next int
	last *paneCopy
}

type paneCopy struct {
	token   int
	group   int
	receipt database.PositionCopyReceipt
}

// CopyPositionToGroup copies position id of group src into the library of
// group dst (Database.CopyPositionTo, under dst's lock from the look-up to
// the write), and keeps its receipt when the copy added the position.
func (a *App) CopyPositionToGroup(src int, id int64, dst int) (PositionCopy, error) {
	ends, err := a.borrowTransferEnds(dst, []int{src})
	if err != nil {
		return PositionCopy{TransferSummary: TransferSummary{Group: ends.dst}}, err
	}
	defer ends.release()
	sum := TransferSummary{Group: ends.dst, Errors: []string{}}
	from := ends.from[src]
	if from == nil {
		return PositionCopy{TransferSummary: sum}, nil
	}
	held, heldErr := ends.to.CountPositions()
	r, err := database.CopyPosition(ends.ctx, from, ends.to, id)
	im := r.Report.Imported
	sum.AlreadyHeld = im.SkippedDuplicates
	sum.Cancelled = errors.Is(err, context.Canceled)
	if err != nil && !sum.Cancelled {
		sum.Errors = append(sum.Errors, err.Error())
	}
	sum.Positions = -1
	if now, err := ends.to.CountPositions(); heldErr == nil && err == nil {
		sum.Positions = max(now-held, 0)
	}
	if sum.Positions != 0 {
		ends.to.RefreshSearchStatistics()
	}
	out := PositionCopy{TransferSummary: sum}
	if err != nil || !r.Created {
		return out, nil
	}
	a.paneCopy.mu.Lock()
	defer a.paneCopy.mu.Unlock()
	a.paneCopy.next++
	a.paneCopy.last = &paneCopy{token: a.paneCopy.next, group: ends.dst, receipt: r}
	out.Receipt = a.paneCopy.next
	return out, nil
}

// UndoPositionCopy takes back the copy whose receipt CopyPositionToGroup
// gave, the last one only: the position leaves the destination with what
// came with it, provided that group still has the same library open and the
// position is as the copy left it (database.ErrPositionChanged otherwise).
func (a *App) UndoPositionCopy(receipt int) error {
	a.paneCopy.mu.Lock()
	c := a.paneCopy.last
	if c == nil || receipt <= 0 || c.token != receipt {
		a.paneCopy.mu.Unlock()
		return errors.New("nothing to take back")
	}
	a.paneCopy.last = nil
	a.paneCopy.mu.Unlock()
	a.libs.mu.Lock()
	locked := a.libs.locked[c.group]
	a.libs.mu.Unlock()
	if locked {
		return ErrLibraryGroupLocked
	}
	return a.readGroup(c.group, func(d *database.Database) error {
		if d.IsReadOnly() {
			return errors.New("the library is open read-only")
		}
		return database.TakeBackPositionCopy(d, c.receipt)
	})
}

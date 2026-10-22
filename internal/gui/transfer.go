package gui

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

// TransferBatch is what one library group gives to a transfer: rows named by
// their ids in that group's library. A drag carries one batch, the lot one
// per group it was filled from.
type TransferBatch struct {
	Group         int     `json:"group"`
	PositionIDs   []int64 `json:"positionIDs"`
	MatchIDs      []int64 `json:"matchIDs"`
	TournamentIDs []int64 `json:"tournamentIDs"`
	CollectionIDs []int64 `json:"collectionIDs"`
	LessonIDs     []int64 `json:"lessonIDs"`
}

func (b TransferBatch) empty() bool {
	return len(b.PositionIDs)+len(b.MatchIDs)+len(b.TournamentIDs)+len(b.CollectionIDs)+len(b.LessonIDs) == 0
}

// parts splits off the batch's tournaments: ingest.Transfer gives a match
// named alone its tournament's record only when no tournament is named beside
// it, so a lot holding both goes in two transfers.
func (b TransferBatch) parts() []TransferBatch {
	if len(b.TournamentIDs) == 0 || len(b.PositionIDs)+len(b.MatchIDs)+len(b.CollectionIDs)+len(b.LessonIDs) == 0 {
		return []TransferBatch{b}
	}
	rest := b
	rest.TournamentIDs = nil
	return []TransferBatch{{Group: b.Group, TournamentIDs: b.TournamentIDs}, rest}
}

// options is the export selection of the batch, with what travels beside each
// row: its analyses, comments and played moves, a collection's positions, a
// match's transcription and aliases. A tournament leaves with all its
// matches; a match named alone brings its tournament's record and none of the
// tournament's other matches (ingest.Transfer's rule). WholeLibrary stays
// off, so that the ids alone say which rows leave.
func (b TransferBatch) options() database.ExportOptions {
	matches := len(b.MatchIDs)+len(b.TournamentIDs) > 0
	return database.ExportOptions{
		PositionIDs:                b.PositionIDs,
		IncludeAnalysis:            true,
		IncludeComments:            true,
		IncludePlayedMoves:         true,
		IncludeMatches:             len(b.MatchIDs) > 0,
		MatchIDs:                   b.MatchIDs,
		IncludeTournaments:         matches,
		TournamentIDs:              b.TournamentIDs,
		IncludeTranscriptions:      matches,
		IncludeAliases:             matches,
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
	Tournaments int      `json:"tournaments"`
	AlreadyHeld int      `json:"alreadyHeld"`
	Collections int      `json:"collections"`
	Lessons     int      `json:"lessons"`
	Errors      []string `json:"errors"`
	Cancelled   bool     `json:"cancelled"`
	// Receipt names, in the session's history of copies, what this transfer
	// created and can take back (UndoPositionCopy); 0 when it created
	// nothing, or created Lessons, whose steps a take-back would cut.
	Receipt int `json:"receipt"`
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
			for _, part := range b.parts() {
				sources = append(sources, source{part, from})
			}
		}
	}

	sum := TransferSummary{Group: dst, Errors: []string{}}
	// The import counts every position it merged, held or not: the account
	// tells what dst did not hold before, by difference. Another import
	// running into dst meanwhile is counted with it; without a first count
	// the account says nothing rather than a false number.
	held, heldErr := to.CountPositions()
	var reports []ingest.TransferReport
	for _, s := range sources {
		rep, err := s.from.TransferTo(ctx, to, s.batch.options())
		reports = append(reports, rep)
		im := rep.Imported
		sum.Matches += im.Matches
		sum.Tournaments += im.Tournaments
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
	if sum.Lessons > 0 {
		return sum, nil
	}
	r, err := database.SealCopy(ctx, to, reports...)
	if err != nil {
		sum.Errors = append(sum.Errors, err.Error())
		return sum, nil
	}
	if !r.Empty() {
		sum.Receipt = a.paneCopy.keep(dst, func(d *database.Database) error { return database.TakeBackCopy(d, r) })
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

// paneCopies is the session's history of copies the desktop can take back,
// positions or matches, tournaments and collections, oldest first, held until
// the application closes; each receipt names the file the copy went into.
type paneCopies struct {
	mu      sync.Mutex
	next    int
	history []paneCopy
}

type paneCopy struct {
	token int
	group int
	// takeBack removes the copy from the library it went into, or refuses
	// with database.ErrPositionChanged / database.ErrCopyChanged.
	takeBack func(d *database.Database) error
}

// keep adds a copy into group to the history and returns its token.
func (p *paneCopies) keep(group int, takeBack func(d *database.Database) error) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.next++
	p.history = append(p.history, paneCopy{token: p.next, group: group, takeBack: takeBack})
	return p.next
}

// find is the index of the copy whose token is receipt, -1 when the history
// does not hold it.
func (p *paneCopies) find(receipt int) int {
	for i, c := range p.history {
		if c.token == receipt {
			return i
		}
	}
	return -1
}

// drop removes the copy at index i from the history.
func (p *paneCopies) drop(i int) {
	p.history = append(p.history[:i], p.history[i+1:]...)
}

// CopyPositionToGroup copies position id of group src into the library of
// group dst (database.CopyPosition, which says whether its own import
// transaction inserted the position), and keeps its receipt in the session's
// history when it did.
func (a *App) CopyPositionToGroup(src int, id int64, dst int) (TransferSummary, error) {
	ends, err := a.borrowTransferEnds(dst, []int{src})
	if err != nil {
		return TransferSummary{Group: ends.dst}, err
	}
	defer ends.release()
	sum := TransferSummary{Group: ends.dst, Errors: []string{}}
	from := ends.from[src]
	if from == nil {
		return sum, nil
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
	out := sum
	if err != nil || !r.Created {
		return out, nil
	}
	out.Receipt = a.paneCopy.keep(ends.dst, func(d *database.Database) error { return database.TakeBackPositionCopy(d, r) })
	return out, nil
}

// ErrLaterCopyFirst refuses to take back a copy while a later copy into the
// same group is still held: copies are taken back in the reverse order.
var ErrLaterCopyFirst = errors.New("a later copy into this library must be taken back first")

// UndoPositionCopy takes back a copy of the session's history by the receipt
// CopyPositionToGroup or TransferToGroup gave: what the copy created leaves
// the destination with what came with it, provided that group still has the
// same library open and holds it as the copy left it
// (database.ErrPositionChanged / database.ErrCopyChanged otherwise,
// and the copy leaves the history: it can no longer be taken back). Copies
// into one group come back newest first (ErrLaterCopyFirst). After a
// transient failure the same receipt can be tried again.
func (a *App) UndoPositionCopy(receipt int) error {
	a.paneCopy.mu.Lock()
	defer a.paneCopy.mu.Unlock()
	i := a.paneCopy.find(receipt)
	if receipt <= 0 || i < 0 {
		return errors.New("nothing to take back")
	}
	c := a.paneCopy.history[i]
	for _, later := range a.paneCopy.history[i+1:] {
		if later.group == c.group {
			return ErrLaterCopyFirst
		}
	}
	if err := a.writeGroup(c.group, func(d *database.Database) error {
		return c.takeBack(d)
	}); err != nil {
		if errors.Is(err, database.ErrPositionChanged) || errors.Is(err, database.ErrCopyChanged) {
			a.paneCopy.drop(i)
		}
		return err
	}
	a.paneCopy.drop(i)
	return nil
}

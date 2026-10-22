package database

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// ErrPositionChanged refuses to take back a copied position the destination
// has changed or come to hold otherwise since the copy: removing it would
// take what was added since with it (the delete cascades).
var ErrPositionChanged = errors.New("the copied position has changed since the copy")

// PositionCopyReceipt is what CopyPosition wrote, enough to take it back
// with TakeBackPositionCopy and only that: the position's id and hash in the
// destination, and a print of the position, its analysis, its comments and
// every row that holds it, taken right after the copy.
type PositionCopyReceipt struct {
	Report ingest.TransferReport
	// Created is set when the copy's own transaction inserted the position; a
	// destination that held it already has nothing to take back.
	Created bool
	ID      int64
	Hash    uint64
	Print   [sha256.Size]byte
	// Path is the destination's file: a Database that opened another one
	// since names other rows by the same id.
	Path string
}

// CopyPosition copies position id of library src into dst through
// TransferTo's selection, with its analysis, comments and played moves.
// Whether the copy created the position is the import transaction's own
// answer (Summary.CreatedPositions), not a look-up around it: dst's lock does
// not exclude a borrowed handle or another Database on the same file, which
// write through the same storage. The print is taken once the copy is
// committed; a write another handle lands on the new position in between is
// counted as the copy's, and taken back with it. It and TakeBackPositionCopy
// are functions, not methods, so that the desktop's binding of Database does
// not offer them to the page.
func CopyPosition(ctx context.Context, src, dst *Database, id int64) (PositionCopyReceipt, error) {
	if src == nil {
		return PositionCopyReceipt{}, fmt.Errorf("%w: no source library", storage.ErrInvalid)
	}
	var r PositionCopyReceipt
	before := func(ctx context.Context) error {
		p, err := src.store.Positions().Load(ctx, "", id)
		if err != nil {
			return err
		}
		r.Hash = storage.ZobristOf(p)
		return nil
	}
	after := func(ctx context.Context, rep ingest.TransferReport) error {
		if len(rep.Imported.CreatedPositions) == 0 {
			return nil
		}
		held, found, err := dst.store.Positions().Exists(ctx, "", r.Hash)
		if err != nil || !found || !slices.Contains(rep.Imported.CreatedPositions, held) {
			return err
		}
		if r.Print, err = positionPrint(ctx, dst.store, held); err != nil {
			return err
		}
		r.ID, r.Created = held, true
		if open := dst.openPath.Load(); open != nil {
			r.Path = *open
		}
		return nil
	}
	opts := ExportOptions{
		PositionIDs:        []int64{id},
		IncludeAnalysis:    true,
		IncludeComments:    true,
		IncludePlayedMoves: true,
	}
	rep, err := src.transferWith(ctx, dst, opts, before, after)
	r.Report = rep
	if err != nil {
		r.Created, r.ID = false, 0
	}
	return r, err
}

// TakeBackPositionCopy removes from d the position r created, with what came
// with it, provided d is the library the copy went into and still holds the
// position as the copy left it: same file, id, hash and print. Otherwise it is
// ErrPositionChanged and nothing is removed. The check and the delete run in
// one transaction that holds the position (HoldForDelete): a write landing
// between them would be removed unseen.
func TakeBackPositionCopy(d *Database, r PositionCopyReceipt) error {
	if !r.Created || r.ID <= 0 {
		return errors.New("nothing to take back")
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}
	if r.Path == "" || !d.IsOpenDatabase(r.Path) {
		return fmt.Errorf("%w: the library the copy went into is no longer open", ErrPositionChanged)
	}
	ctx := context.Background()
	tx, err := d.store.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := tx.Positions().HoldForDelete(ctx, "", r.ID); errors.Is(err, storage.ErrNotFound) {
		return ErrPositionChanged
	} else if err != nil {
		return err
	}
	p, err := tx.Positions().Load(ctx, "", r.ID)
	if err != nil {
		return err
	}
	if storage.ZobristOf(p) != r.Hash {
		return ErrPositionChanged
	}
	print, err := positionPrint(ctx, tx, r.ID)
	if err != nil {
		return err
	}
	if print != r.Print {
		return ErrPositionChanged
	}
	if err := tx.Positions().Delete(ctx, "", r.ID); err != nil {
		return err
	}
	return tx.Commit()
}

// positionPrint digests what the library says of position id and what holds
// it: the row itself, its analysis, its comments, the collections, matches,
// Anki decks and Lesson steps that name it. Any write that the delete of the
// position would undo changes the print.
func positionPrint(ctx context.Context, st storage.Stores, id int64) ([sha256.Size]byte, error) {
	var zero [sha256.Size]byte
	var v struct {
		Position    any
		Analysis    any
		Comments    []any
		Collections []int64
		Matches     []int64
		Decks       []int64
		Steps       []int64
	}
	p, err := st.Positions().Load(ctx, "", id)
	if err != nil {
		return zero, err
	}
	v.Position = p
	a, err := st.Analyses().Load(ctx, "", id)
	switch {
	case errors.Is(err, storage.ErrNotFound):
	case err != nil:
		return zero, err
	default:
		v.Analysis = a
	}
	for c, err := range st.Comments().ByPosition(ctx, "", id) {
		if err != nil {
			return zero, err
		}
		v.Comments = append(v.Comments, c)
	}
	for c, err := range st.Collections().CollectionsOf(ctx, "", id) {
		if err != nil {
			return zero, err
		}
		v.Collections = append(v.Collections, c.ID)
	}
	if v.Matches, err = st.Matches().MatchesByPosition(ctx, "", id); err != nil {
		return zero, err
	}
	var decks []int64
	for deck, err := range st.Anki().ListDecks(ctx, "") {
		if err != nil {
			return zero, err
		}
		decks = append(decks, deck.ID)
	}
	for _, deck := range decks {
		_, held, err := st.Anki().IndexOfDeckPosition(ctx, "", deck, id)
		if err != nil {
			return zero, err
		}
		if held {
			v.Decks = append(v.Decks, deck)
		}
	}
	lessons, err := st.Lessons().List(ctx, "")
	if err != nil {
		return zero, err
	}
	for _, l := range lessons {
		full, err := st.Lessons().Get(ctx, "", l.ID)
		if err != nil {
			return zero, err
		}
		for _, s := range full.Steps {
			if s.PositionID == id {
				v.Steps = append(v.Steps, s.ID)
			}
		}
	}
	b, err := json.Marshal(v)
	if err != nil {
		return zero, err
	}
	return sha256.Sum256(b), nil
}

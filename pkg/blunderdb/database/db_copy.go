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

// ErrCopyChanged refuses to take back a copy of matches, tournaments or
// collections that the destination has changed since: removing what the copy
// created would take what was added since with it.
var ErrCopyChanged = errors.New("what the copy added has changed since the copy")

// CopyReceipt is what one or more transfers into a library created there,
// enough to take it back with TakeBackCopy and only that: the rows each
// import transaction inserted (never a row the destination held already),
// the members it gave to collections the destination held, and a print of
// each, taken once the copies are written. The rows a copy merged into what
// the destination held — an analysis, a comment, a match filed into a
// tournament of the same name — stay: they were the destination's.
type CopyReceipt struct {
	Path        string
	Positions   []printedRow
	Matches     []printedRow
	Tournaments []printedRow
	Collections []printedRow
	// Members are the positions given to collections the destination held,
	// by collection id, with the print of each such collection.
	Members []addedMembers
}

type printedRow struct {
	ID    int64
	Hash  uint64 // positions only: the Zobrist hash the row was stored under
	Print [sha256.Size]byte
}

type addedMembers struct {
	Collection int64
	Positions  []int64
	Print      [sha256.Size]byte
}

// Empty reports whether the copies created nothing to take back.
func (r CopyReceipt) Empty() bool {
	return len(r.Positions)+len(r.Matches)+len(r.Tournaments)+len(r.Collections)+len(r.Members) == 0
}

// SealCopy is the receipt of reports, the transfers just made into d: what
// their import transactions created, printed as d holds it now. A write
// another handle landed on those rows in between is counted as the copy's,
// and taken back with it. A position created by one transfer and found by a
// later one stays the first one's; nothing is printed twice.
func SealCopy(ctx context.Context, d *Database, reports ...ingest.TransferReport) (CopyReceipt, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.db == nil {
		return CopyReceipt{}, fmt.Errorf("no database is currently open")
	}
	var r CopyReceipt
	if open := d.openPath.Load(); open != nil {
		r.Path = *open
	}
	members := map[int64][]int64{}
	var order []int64
	var positions, matches, tournaments, collections []int64
	for _, rep := range reports {
		im := rep.Imported
		positions = append(positions, im.CreatedPositions...)
		matches = append(matches, im.CreatedMatches...)
		tournaments = append(tournaments, im.CreatedTournaments...)
		collections = append(collections, im.CreatedCollections...)
		for c, ids := range im.AddedMembers {
			if _, seen := members[c]; !seen {
				order = append(order, c)
			}
			members[c] = append(members[c], ids...)
		}
	}
	st := d.store
	for _, id := range dedup(positions) {
		p, err := st.Positions().Load(ctx, "", id)
		if err != nil {
			return CopyReceipt{}, err
		}
		pr, err := positionPrint(ctx, st, id)
		if err != nil {
			return CopyReceipt{}, err
		}
		r.Positions = append(r.Positions, printedRow{ID: id, Hash: storage.ZobristOf(p), Print: pr})
	}
	for _, f := range []struct {
		ids   []int64
		print func(context.Context, storage.Stores, int64) ([sha256.Size]byte, error)
		into  *[]printedRow
	}{
		{matches, matchPrint, &r.Matches},
		{tournaments, tournamentPrint, &r.Tournaments},
		{collections, collectionPrint, &r.Collections},
	} {
		for _, id := range dedup(f.ids) {
			pr, err := f.print(ctx, st, id)
			if err != nil {
				return CopyReceipt{}, err
			}
			*f.into = append(*f.into, printedRow{ID: id, Print: pr})
		}
	}
	for _, c := range order {
		if slices.Contains(collections, c) {
			continue
		}
		pr, err := collectionPrint(ctx, st, c)
		if err != nil {
			return CopyReceipt{}, err
		}
		r.Members = append(r.Members, addedMembers{Collection: c, Positions: dedup(members[c]), Print: pr})
	}
	return r, nil
}

func dedup(ids []int64) []int64 {
	out := make([]int64, 0, len(ids))
	seen := make(map[int64]bool, len(ids))
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

// TakeBackCopy removes from d what r says the copies created, provided d is
// the library they went into and holds every such row as the copies left it:
// same file, ids, hashes and prints. Otherwise it is ErrCopyChanged and
// nothing is removed. Every print is checked before anything is deleted, in
// the one transaction that deletes: SQLite's transaction holds the write lock
// from its first statement, and each created position is held as
// TakeBackPositionCopy holds it.
//
// The order is the reverse of the import's: members, collections, matches
// (their games first, so that the match delete has no position left to purge
// and no position the copy did not create is touched), tournaments, then the
// created positions, deleted explicitly with what hangs on them.
func TakeBackCopy(d *Database, r CopyReceipt) error {
	if r.Empty() {
		return errors.New("nothing to take back")
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}
	if r.Path == "" || !d.IsOpenDatabase(r.Path) {
		return fmt.Errorf("%w: the library the copy went into is no longer open", ErrCopyChanged)
	}
	ctx := context.Background()
	tx, err := d.store.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	for _, p := range r.Positions {
		if err := tx.Positions().HoldForDelete(ctx, "", p.ID); errors.Is(err, storage.ErrNotFound) {
			return ErrCopyChanged
		} else if err != nil {
			return err
		}
		pos, err := tx.Positions().Load(ctx, "", p.ID)
		if err != nil {
			return err
		}
		if storage.ZobristOf(pos) != p.Hash {
			return ErrCopyChanged
		}
	}
	check := func(rows []printedRow, print func(context.Context, storage.Stores, int64) ([sha256.Size]byte, error)) error {
		for _, row := range rows {
			pr, err := print(ctx, tx, row.ID)
			if errors.Is(err, storage.ErrNotFound) || err == nil && pr != row.Print {
				return ErrCopyChanged
			}
			if err != nil {
				return err
			}
		}
		return nil
	}
	if err := check(r.Positions, positionPrint); err != nil {
		return err
	}
	if err := check(r.Matches, matchPrint); err != nil {
		return err
	}
	if err := check(r.Tournaments, tournamentPrint); err != nil {
		return err
	}
	if err := check(r.Collections, collectionPrint); err != nil {
		return err
	}
	for _, m := range r.Members {
		pr, err := collectionPrint(ctx, tx, m.Collection)
		if errors.Is(err, storage.ErrNotFound) || err == nil && pr != m.Print {
			return ErrCopyChanged
		}
		if err != nil {
			return err
		}
	}

	for _, m := range r.Members {
		if err := tx.Collections().RemovePositions(ctx, "", m.Collection, m.Positions); err != nil {
			return err
		}
	}
	for _, c := range r.Collections {
		if err := tx.Collections().Delete(ctx, "", c.ID); err != nil {
			return err
		}
	}
	for _, m := range r.Matches {
		if _, err := tx.Matches().DeleteGames(ctx, "", m.ID); err != nil {
			return err
		}
		if err := tx.Matches().DeleteCascade(ctx, "", m.ID); err != nil {
			return err
		}
	}
	for _, t := range r.Tournaments {
		if err := tx.Tournaments().Delete(ctx, "", t.ID); err != nil {
			return err
		}
	}
	for _, p := range r.Positions {
		if err := tx.Positions().Delete(ctx, "", p.ID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// matchPrint digests match id as a take-back would remove it: its row and
// every move of its games.
func matchPrint(ctx context.Context, st storage.Stores, id int64) ([sha256.Size]byte, error) {
	var v struct {
		Match any
		Moves []any
	}
	m, err := st.Matches().Get(ctx, "", id)
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	v.Match = m
	for mv, err := range st.Matches().MovesByMatch(ctx, "", id) {
		if err != nil {
			return [sha256.Size]byte{}, err
		}
		v.Moves = append(v.Moves, mv)
	}
	return digest(v)
}

// tournamentPrint digests tournament id and the matches filed in it: a match
// filed since would be left without its tournament.
func tournamentPrint(ctx context.Context, st storage.Stores, id int64) ([sha256.Size]byte, error) {
	var v struct {
		Tournament any
		Matches    []int64
	}
	t, err := st.Tournaments().Get(ctx, "", id)
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	v.Tournament = t
	for m, err := range st.Tournaments().Matches(ctx, "", id) {
		if err != nil {
			return [sha256.Size]byte{}, err
		}
		v.Matches = append(v.Matches, m.ID)
	}
	return digest(v)
}

// collectionPrint digests collection id and its members in order.
func collectionPrint(ctx context.Context, st storage.Stores, id int64) ([sha256.Size]byte, error) {
	var v struct {
		Collection any
		Members    []int64
	}
	c, err := st.Collections().Get(ctx, "", id)
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	v.Collection = c
	if v.Members, err = st.Collections().PositionIDs(ctx, "", id, storage.ListOpts{}); err != nil {
		return [sha256.Size]byte{}, err
	}
	return digest(v)
}

func digest(v any) ([sha256.Size]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	return sha256.Sum256(b), nil
}

// Package changeset moves the change journal between two libraries: an export
// of what changed after a cursor, and an import that merges it, the last
// write winning.
//
// A Set carries, per object, the last thing that happened to it
// (storage.Change) and, for a create or an update, the object's content. The
// families it speaks are the ones in codecs. The journal is read for those
// families only, so the cursor never passes an entry of a family not carried
// yet: adding a family means peers export again from 0 (an import is
// idempotent). Everything is written against storage.Stores, so the desktop,
// the CLI and the daemon share it.
//
// Concurrency of the cursor: seq is monotone but a PostgreSQL writer can commit
// a lower seq after a higher one was read (see storage.ChangeJournalStore).
package changeset

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/trash"
)

// floor is the date of an object that existed before the journal: older than
// any change, so it never overwrites an object the other side holds.
const floor = "1970-01-01T00:00:00.000Z"

// Record is one object's last change and, for a create or an update, its content.
type Record struct {
	storage.Change
	Payload json.RawMessage `json:"payload,omitempty"`
}

// Set is the changes after a cursor. Cursor is the seq of the last journal
// entry read of a carried family: the next export starts there, and an entry
// of a family not carried is never behind it. More says the limit cut the
// read short.
//
// An export from cursor 0 also carries the objects that predate the journal
// (they have no entry), dated at the floor, all in the one set: its size is
// that of those rows, not bounded by the limit.
type Set struct {
	Cursor  int64    `json:"cursor"`
	More    bool     `json:"more"`
	Records []Record `json:"records"`
}

// Report counts what an import did with each record.
type Report struct {
	Applied int `json:"applied"`
	// Older records lost against a newer local change (or an equal one).
	Older int `json:"older"`
	// Conflicts could not be applied: another local object holds the name.
	Conflicts []string `json:"conflicts,omitempty"`
	// Unsupported records are of a family the changeset does not carry yet.
	Unsupported int `json:"unsupported"`
	// Clamped records carried a date later than this library's clock: they
	// were taken as made now, so they cannot outrank every later local edit.
	Clamped int `json:"clamped,omitempty"`
}

// codec reads and writes one family's content.
type codec struct {
	trashKind domain.TrashKind
	read      func(ctx context.Context, s storage.Stores, scope string, id int64) (any, error)
	create    func(ctx context.Context, s storage.Stores, scope string, payload json.RawMessage) (int64, error)
	update    func(ctx context.Context, s storage.Stores, scope string, id int64, payload json.RawMessage) error
	remove    func(ctx context.Context, s storage.Stores, scope string, id int64, toTrash bool) error
}

var codecs = map[string]codec{"collection": collectionCodec}

func families() []string {
	out := make([]string, 0, len(codecs))
	for f := range codecs {
		out = append(out, f)
	}
	slices.Sort(out)
	return out
}

func providers(s storage.Stores) (storage.ChangeJournalStore, storage.ObjectUUIDStore, error) {
	j, ok1 := s.(storage.ChangeJournalProvider)
	u, ok2 := s.(storage.ObjectUUIDProvider)
	if !ok1 || !ok2 {
		return nil, nil, errors.New("changeset: the backend keeps no change journal")
	}
	return j.ChangeJournal(), u.ObjectUUIDs(), nil
}

// normalize renders a date as the journal's layout: UTC, milliseconds. Text
// of that layout compares as time; the text a peer sends may not (an offset,
// no fraction).
func normalize(at string) (string, error) {
	t, err := time.Parse(time.RFC3339Nano, at)
	if err != nil {
		return "", fmt.Errorf("changeset: date %q: %w", at, storage.ErrInvalid)
	}
	return t.UTC().Truncate(time.Millisecond).Format(storage.ChangedAtLayout), nil
}

// Export reads up to limit journal entries of the carried families after the
// cursor and loads the content of the creates and updates. An object that
// vanished between the journal read and the content read is left out: its
// delete is a later entry.
func Export(ctx context.Context, s storage.Stores, scope string, after int64, limit int) (Set, error) {
	journal, uuids, err := providers(s)
	if err != nil {
		return Set{}, err
	}
	if limit <= 0 {
		limit = 1000
	}
	set := Set{Cursor: after}
	// The rows that predate the journal first: an object created while they
	// are read has an entry, so the journal read below finds it.
	if after == 0 {
		for _, f := range families() {
			us, err := journal.Unjournaled(ctx, scope, f)
			if err != nil {
				return Set{}, err
			}
			for _, u := range us {
				rec := Record{Change: storage.Change{UUID: u, Family: f, Op: storage.ChangeCreate, ChangedAt: floor}}
				ok, err := loadPayload(ctx, s, uuids, scope, &rec)
				if err != nil {
					return Set{}, err
				}
				if ok {
					set.Records = append(set.Records, rec)
				}
			}
		}
	}
	changes, err := journal.ChangesOf(ctx, scope, families(), after, limit+1)
	if err != nil {
		return Set{}, err
	}
	if len(changes) > limit {
		changes, set.More = changes[:limit], true
	}
	for _, c := range changes {
		set.Cursor = c.Seq
		rec := Record{Change: c}
		if c.Op == storage.ChangeCreate || c.Op == storage.ChangeUpdate {
			ok, err := loadPayload(ctx, s, uuids, scope, &rec)
			if err != nil {
				return Set{}, err
			}
			if !ok {
				continue
			}
		}
		set.Records = append(set.Records, rec)
	}
	return set, nil
}

// loadPayload fills the record's content; false when the object is gone.
func loadPayload(ctx context.Context, s storage.Stores, uuids storage.ObjectUUIDStore, scope string, rec *Record) (bool, error) {
	id, err := uuids.ObjectID(ctx, scope, rec.Family, rec.UUID)
	if errors.Is(err, storage.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	content, err := codecs[rec.Family].read(ctx, s, scope, id)
	if err != nil {
		return false, err
	}
	rec.Payload, err = json.Marshal(content)
	return err == nil, err
}

// tieKey orders two changes made at the same millisecond the same way on both
// sides: by what they did (a write over a trash over a delete), then by a
// digest of the content. Equal keys are the same change.
func tieKey(op string, payload []byte) string {
	class := "u"
	switch op {
	case storage.ChangeTrash:
		class = "t"
	case storage.ChangeDelete:
		class = "d"
	}
	if class != "u" || len(payload) == 0 {
		return class
	}
	var v any
	if json.Unmarshal(payload, &v) != nil {
		return class + string(payload)
	}
	canon, _ := json.Marshal(v) // maps marshal with sorted keys
	sum := sha256.Sum256(canon)
	return class + hex.EncodeToString(sum[:])
}

// Import merges a set into the library: per object, the later change wins.
// Dates are normalised (UTC, milliseconds) and compared as dates; one later
// than this library's clock is taken as made now (Report.Clamped). A tie to
// the millisecond goes to the greater tieKey, so both sides pick the same
// one; an identical change is a no-op, which makes a record that comes back
// around harmless. A restore therefore beats an older delete and a delete an
// older update. An object that exists with no journal entry predates the
// journal and counts as dated at the floor. What is applied keeps the time it
// had where it was made, so it is not re-sent as newer. A delete of an object
// this library never held is still recorded, so an older create that arrives
// later loses. Each record is applied, and its entry written, in one
// transaction.
func Import(ctx context.Context, s storage.Stores, scope string, set Set) (Report, error) {
	var rep Report
	for _, r := range set.Records {
		cd, ok := codecs[r.Family]
		if !ok {
			rep.Unsupported++
			continue
		}
		switch r.Op {
		case storage.ChangeCreate, storage.ChangeUpdate, storage.ChangeDelete, storage.ChangeTrash:
		default:
			return rep, fmt.Errorf("changeset: operation %q of %s: %w", r.Op, r.UUID, storage.ErrInvalid)
		}
		at, err := normalize(r.ChangedAt)
		if err != nil {
			return rep, err
		}
		if now := time.Now().UTC().Format(storage.ChangedAtLayout); at > now {
			at = now
			rep.Clamped++
		}
		r.ChangedAt = at
		var applied, conflict bool
		err = trash.InTx(ctx, s, scope, func(s storage.Stores) error {
			var err error
			applied, conflict, err = apply(ctx, s, scope, cd, r)
			return err
		})
		switch {
		case err != nil:
			return rep, err
		case conflict:
			rep.Conflicts = append(rep.Conflicts, r.UUID)
		case applied:
			rep.Applied++
		default:
			rep.Older++
		}
	}
	return rep, nil
}

// apply merges one record inside the caller's transaction.
func apply(ctx context.Context, s storage.Stores, scope string, cd codec, r Record) (applied, conflict bool, err error) {
	journal, uuids, err := providers(s)
	if err != nil {
		return false, false, err
	}
	local, err := journal.Entry(ctx, scope, r.UUID)
	hasEntry := err == nil
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return false, false, err
	}
	id, err := uuids.ObjectID(ctx, scope, r.Family, r.UUID)
	held := err == nil
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return false, false, err
	}
	switch {
	case hasEntry:
		localAt, err := normalize(local.ChangedAt)
		if err != nil {
			return false, false, err
		}
		switch {
		case r.ChangedAt < localAt:
			return false, false, nil
		case r.ChangedAt == localAt:
			var payload []byte
			if held && local.Op != storage.ChangeTrash && local.Op != storage.ChangeDelete {
				content, err := cd.read(ctx, s, scope, id)
				if err != nil {
					return false, false, err
				}
				if payload, err = json.Marshal(content); err != nil {
					return false, false, err
				}
			}
			if tieKey(r.Op, r.Payload) <= tieKey(local.Op, payload) {
				return false, false, nil
			}
		}
	case held && r.ChangedAt <= floor:
		return false, false, nil
	}

	switch r.Op {
	case storage.ChangeDelete, storage.ChangeTrash:
		if held {
			if err := cd.remove(ctx, s, scope, id, r.Op == storage.ChangeTrash); err != nil {
				return false, false, err
			}
		}
		if r.Op == storage.ChangeDelete {
			// Gone for good: the entry the trash kept can no longer be restored.
			if err := trash.DiscardByUUID(ctx, s, scope, cd.trashKind, r.UUID); err != nil {
				return false, false, err
			}
		}
	default:
		if held {
			err = cd.update(ctx, s, scope, id, r.Payload)
		} else if id, err = cd.create(ctx, s, scope, r.Payload); err == nil {
			if err = uuids.SetObjectUUID(ctx, scope, r.Family, id, r.UUID); err == nil {
				// It is back by this way: restoring the trash entry that held
				// it would make a second one.
				err = trash.DiscardByUUID(ctx, s, scope, cd.trashKind, r.UUID)
			}
		}
		if errors.Is(err, storage.ErrConflict) {
			return false, true, nil
		}
		if err != nil {
			return false, false, err
		}
	}
	if err := journal.Mark(ctx, scope, r.Family, r.UUID, r.Op, r.ChangedAt); err != nil {
		return false, false, err
	}
	return true, false, nil
}

// --- collection ---

// collectionPayload is a collection with its membership, by position uuid, in
// order. A living collection (FilterQuery) has no membership rows. A member
// whose position this library does not hold is left out on import.
type collectionPayload struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	FilterQuery string   `json:"filterQuery,omitempty"`
	SortOrder   int      `json:"sortOrder"`
	Members     []string `json:"members,omitempty"`
}

var collectionCodec = codec{
	trashKind: domain.TrashCollection,
	read: func(ctx context.Context, s storage.Stores, scope string, id int64) (any, error) {
		c, err := s.Collections().Get(ctx, scope, id)
		if err != nil {
			return nil, err
		}
		p := collectionPayload{Name: c.Name, Description: c.Description, FilterQuery: c.FilterQuery, SortOrder: c.SortOrder}
		if c.FilterQuery != "" {
			return p, nil
		}
		ids, err := s.Collections().PositionIDs(ctx, scope, id, storage.ListOpts{})
		if err != nil {
			return nil, err
		}
		uu := s.(storage.ObjectUUIDProvider).ObjectUUIDs()
		for _, pid := range ids {
			u, err := uu.ObjectUUID(ctx, scope, "position", pid)
			if err != nil {
				return nil, err
			}
			if u != "" {
				p.Members = append(p.Members, u)
			}
		}
		return p, nil
	},
	create: func(ctx context.Context, s storage.Stores, scope string, raw json.RawMessage) (int64, error) {
		var p collectionPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			return 0, fmt.Errorf("collection payload: %w", err)
		}
		id, err := s.Collections().Create(ctx, scope, p.Name, p.Description)
		if err != nil {
			return 0, err
		}
		return id, fill(ctx, s, scope, id, p, true)
	},
	update: func(ctx context.Context, s storage.Stores, scope string, id int64, raw json.RawMessage) error {
		var p collectionPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			return fmt.Errorf("collection payload: %w", err)
		}
		if err := s.Collections().Update(ctx, scope, id, p.Name, p.Description); err != nil {
			return err
		}
		return fill(ctx, s, scope, id, p, false)
	},
	remove: func(ctx context.Context, s storage.Stores, scope string, id int64, toTrash bool) error {
		if toTrash {
			_, err := trash.Collection(ctx, s, scope, id)
			return err
		}
		return s.Collections().Delete(ctx, scope, id)
	},
}

// fill brings a collection's query, membership and rank to the payload's.
func fill(ctx context.Context, s storage.Stores, scope string, id int64, p collectionPayload, created bool) error {
	cs := s.Collections()
	if p.FilterQuery != "" || !created {
		if err := cs.SetFilterQuery(ctx, scope, id, p.FilterQuery); err != nil {
			return err
		}
	}
	if p.FilterQuery == "" {
		uu := s.(storage.ObjectUUIDProvider).ObjectUUIDs()
		var want []int64
		for _, u := range p.Members {
			pid, err := uu.ObjectID(ctx, scope, "position", u)
			if errors.Is(err, storage.ErrNotFound) {
				continue
			}
			if err != nil {
				return err
			}
			if !slices.Contains(want, pid) {
				want = append(want, pid)
			}
		}
		have, err := cs.PositionIDs(ctx, scope, id, storage.ListOpts{})
		if err != nil {
			return err
		}
		if !slices.Equal(have, want) {
			var drop, add []int64
			for _, h := range have {
				if !slices.Contains(want, h) {
					drop = append(drop, h)
				}
			}
			for _, w := range want {
				if !slices.Contains(have, w) {
					add = append(add, w)
				}
			}
			if len(drop) > 0 {
				if err := cs.RemovePositions(ctx, scope, id, drop); err != nil {
					return err
				}
			}
			if len(add) > 0 {
				if err := cs.AddPositions(ctx, scope, id, add); err != nil {
					return err
				}
			}
			if len(want) > 1 {
				if err := cs.ReorderPositions(ctx, scope, id, want); err != nil {
					return err
				}
			}
		}
	}
	return placeCollection(ctx, s, scope, id, p.SortOrder)
}

// placeCollection moves the collection to rank among the library's, when it
// is not there. The others keep their journal dates: reordering rewrites their
// rows, which is no edit of theirs and must not look newer than a peer's.
func placeCollection(ctx context.Context, s storage.Stores, scope string, id int64, rank int) error {
	cs := s.Collections()
	var order []int64
	journal, uuids, err := providers(s)
	if err != nil {
		return err
	}
	for c, err := range cs.List(ctx, scope) {
		if err != nil {
			return err
		}
		order = append(order, c.ID)
	}
	cur := slices.Index(order, id)
	target := min(max(rank, 0), len(order)-1)
	if cur < 0 || cur == target {
		return nil
	}
	order = slices.Delete(order, cur, cur+1)
	order = slices.Insert(order, target, id)
	type stamp struct{ uuid, op, at string }
	var stamps []stamp
	for _, o := range order {
		if o == id {
			continue
		}
		u, err := uuids.ObjectUUID(ctx, scope, "collection", o)
		if err != nil || u == "" {
			continue
		}
		if e, err := journal.Entry(ctx, scope, u); err == nil {
			stamps = append(stamps, stamp{u, e.Op, e.ChangedAt})
		}
	}
	if err := cs.Reorder(ctx, scope, order); err != nil {
		return err
	}
	for _, st := range stamps {
		if err := journal.Mark(ctx, scope, "collection", st.uuid, st.op, st.at); err != nil {
			return err
		}
	}
	return nil
}

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
// An export is a run of calls, one page each. Within a run the families come in
// rank order, each read from the cursor the run started at, so a page boundary
// never falls between an object and what it points at, whatever order the
// journal holds them in. A run ends on the journal's horizon at its start: what
// changed during the run comes again in the next one, and with it whatever
// points at it. An import absorbs the repetition.
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

// Set is the changes after a cursor. Cursor is opaque: the next export starts
// there, and an entry of a family not carried is never behind it. More says the
// limit cut the run short: ask again with Cursor.
//
// An export from cursor 0 first gives the objects that predate the journal
// (they have no entry) an entry dated at the floor; they then come, a page at
// a time, with the rest.
type Set struct {
	Cursor  storage.Cursor `json:"cursor"`
	More    bool           `json:"more"`
	Records []Record       `json:"records"`
}

// Report counts what an import did with each record.
type Report struct {
	// Applied is Added, Modified and Removed, and a delete or trash of an object
	// this library never held, which removes nothing but is recorded.
	Applied  int `json:"applied"`
	Added    int `json:"added"`
	Modified int `json:"modified"`
	Removed  int `json:"removed"`
	// Older records lost against a newer local change (or an equal one).
	Older int `json:"older"`
	// Conflicts could not be applied: another local object holds the name.
	Conflicts []string `json:"conflicts,omitempty"`
	// Missing are references of applied records this library could not place.
	Missing []MissingRef `json:"missing,omitempty"`
	// Unsupported records are of a family the changeset does not carry yet.
	Unsupported int `json:"unsupported"`
	// Clamped records carried a date later than this library's clock: they
	// were taken as made now, so they cannot outrank every later local edit.
	Clamped int `json:"clamped,omitempty"`
}

// MissingRef is a reference an applied record holds that this library cannot
// place.
type MissingRef struct {
	Family string `json:"family"`
	UUID   string `json:"uuid"`
	Ref    string `json:"ref"`
}

// codec reads and writes one family's content.
type codec struct {
	// rank orders the families of one set on import: what others point at
	// (positions, matches, collections) before what points at them.
	rank      int
	trashKind domain.TrashKind
	// locate finds the local object a record names when its uuid is unknown
	// here, and gives it the record's uuid when that is the older one; nil for
	// the families whose uuid is the only name.
	locate func(ctx context.Context, s storage.Stores, scope string, r Record) (id int64, uuid string, found bool, err error)
	read   func(ctx context.Context, s storage.Stores, scope string, id int64) (any, error)
	create func(ctx context.Context, s storage.Stores, scope string, payload json.RawMessage) (int64, error)
	update func(ctx context.Context, s storage.Stores, scope string, id int64, payload json.RawMessage) error
	remove func(ctx context.Context, s storage.Stores, scope string, id int64, toTrash bool) error
	// merge settles a tie between two contents of the same object, nil when the
	// greater tieKey simply wins. It keeps what the winner would lose for no
	// reason; a same result on both sides is what lets them converge.
	merge func(winner, loser json.RawMessage) (json.RawMessage, error)
}

var codecs = map[string]codec{
	"position":   positionCodec,
	"match":      matchCodec,
	"collection": collectionCodec,
	"comment":    commentCodec,
	"lesson":     lessonCodec,
}

// families lists the carried families in rank order.
func families() []string {
	out := make([]string, 0, len(codecs))
	for f := range codecs {
		out = append(out, f)
	}
	slices.SortFunc(out, func(a, b string) int { return codecs[a].rank - codecs[b].rank })
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
// delete is a later entry. ErrCursorStale when the cursor belongs to a journal
// this one is not: export from the zero Cursor instead.
func Export(ctx context.Context, s storage.Stores, scope string, after storage.Cursor, limit int) (Set, error) {
	journal, uuids, err := providers(s)
	if err != nil {
		return Set{}, err
	}
	if limit <= 0 {
		limit = 1000
	}
	carried := families()
	run := after
	if run.Family == "" {
		// The rows that predate the journal have no entry for a cursor to reach:
		// from the start they are given one, dated at the floor. An object created
		// or edited meanwhile keeps its own entry.
		if after.IsStart() {
			for _, f := range carried {
				var id int64
				for {
					us, last, err := journal.Unjournaled(ctx, scope, f, id, limit)
					if err != nil {
						return Set{}, err
					}
					if len(us) == 0 {
						break
					}
					if err := journal.Adopt(ctx, scope, f, us, floor); err != nil {
						return Set{}, err
					}
					id = last
				}
			}
		}
		end, err := journal.Horizon(ctx, scope, carried, after)
		if err != nil {
			return Set{}, err
		}
		run.Epoch = end.Epoch
		run.Family, run.ReadXid, run.ReadSeq = carried[0], after.Xid, after.Seq
		run.EndXid, run.EndSeq = end.Xid, end.Seq
		// An entry past the horizon can be gone (a uuid given back drops it).
		if end.Xid < after.Xid || (end.Xid == after.Xid && end.Seq < after.Seq) {
			run.EndXid, run.EndSeq = after.Xid, after.Seq
		}
	}
	family := slices.Index(carried, run.Family)
	if family < 0 {
		return Set{}, fmt.Errorf("changeset: cursor names family %q: %w", run.Family, storage.ErrCursorStale)
	}
	set := Set{}
	read := 0
	for {
		room := limit - read
		changes, err := journal.ChangesOf(ctx, scope, carried[family:family+1],
			storage.Cursor{Xid: run.ReadXid, Seq: run.ReadSeq, Epoch: run.Epoch}, room+1)
		if err != nil {
			return Set{}, err
		}
		if len(changes) > room {
			changes, set.More = changes[:room], true
		}
		for _, c := range changes {
			read++
			run.ReadXid, run.ReadSeq = c.Xid, c.Seq
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
		if set.More {
			set.Cursor = run
			return set, nil
		}
		if family++; family == len(carried) {
			set.Cursor = storage.Cursor{Xid: run.EndXid, Seq: run.EndSeq, Epoch: run.Epoch}
			return set, nil
		}
		run.Family, run.ReadXid, run.ReadSeq = carried[family], run.Xid, run.Seq
	}
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

// clockSlack is how far ahead of this library's clock a record may be dated and
// still keep its date. A journal dates two writes of an object in the same
// millisecond a millisecond apart, so an honest date can lead the clock a little;
// taken as made now, it would tie with, or lose to, the write it follows. A
// local edit made after it still comes later: the journal dates it past what the
// object holds.
const clockSlack = time.Second

// Import merges a set into the library: per object, the later change wins.
// Dates are normalised (UTC, milliseconds) and compared as dates; one later
// than this library's clock by more than clockSlack is taken as made now
// (Report.Clamped). A tie to
// the millisecond goes to the greater tieKey, so both sides pick the same
// one; an identical change is a no-op, which makes a record that comes back
// around harmless. A restore therefore beats an older delete and a delete an
// older update. An object that exists with no journal entry predates the
// journal and counts as dated at the floor. What is applied keeps the time it
// had where it was made, so it is not re-sent as newer. A delete of an object
// this library never held is still recorded, so an older create that arrives
// later loses. Each record is applied, and its entry written, in one
// transaction, in which nothing it writes is dated now (see Applying).
func Import(ctx context.Context, s storage.Stores, scope string, set Set) (Report, error) {
	var rep Report
	var waiting []waitingLink
	records := slices.Clone(set.Records)
	slices.SortStableFunc(records, func(a, b Record) int { return codecs[a.Family].rank - codecs[b.Family].rank })
	for _, r := range records {
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
		future := ""
		now := time.Now().UTC()
		if at > now.Add(clockSlack).Format(storage.ChangedAtLayout) {
			future, at = at, now.Format(storage.ChangedAtLayout)
			rep.Clamped++
		}
		r.ChangedAt = at
		var res outcome
		var missing []string
		rctx := context.WithValue(ctx, missingKey{}, &missing)
		var links []waitingLink
		rctx = context.WithValue(rctx, waitingKey{}, &links)
		err = trash.InTx(rctx, s, scope, func(s storage.Stores) error {
			journal, _, err := providers(s)
			if err != nil {
				return err
			}
			if err := journal.Applying(rctx, scope, r.ChangedAt); err != nil {
				return err
			}
			if res, err = apply(rctx, s, scope, cd, r, future); err != nil {
				return err
			}
			return journal.Applying(rctx, scope, "")
		})
		if err != nil {
			return rep, err
		}
		switch res {
		case conflicted:
			rep.Conflicts = append(rep.Conflicts, r.UUID)
			continue
		case lost:
			rep.Older++
			continue
		case added:
			rep.Added++
		case modified:
			rep.Modified++
		case removed:
			rep.Removed++
		}
		rep.Applied++
		for _, ref := range missing {
			rep.Missing = append(rep.Missing, MissingRef{Family: r.Family, UUID: r.UUID, Ref: ref})
		}
		for _, l := range links {
			l.Family, l.UUID = r.Family, r.UUID
			waiting = append(waiting, l)
		}
	}
	// Every record is in: the origins that arrived after their variant, in this
	// set or an earlier one, are here now. What is still missing is the peer's
	// to send.
	still, err := settleLinks(ctx, s, scope, &rep)
	if err != nil {
		return rep, err
	}
	for _, l := range waiting {
		if still[l.Variant] {
			rep.Missing = append(rep.Missing, MissingRef{Family: l.Family, UUID: l.UUID, Ref: l.Ref})
		}
	}
	return rep, nil
}

// outcome is what apply did with a record.
type outcome int

const (
	lost outcome = iota
	conflicted
	added
	modified
	removed
	recorded // a delete of an object this library never held
)

type missingKey struct{}

// noteMissing records a reference the record being applied holds and this
// library cannot place.
func noteMissing(ctx context.Context, ref string) {
	if list, ok := ctx.Value(missingKey{}).(*[]string); ok {
		*list = append(*list, ref)
	}
}

// apply merges one record inside the caller's transaction.
func apply(ctx context.Context, s storage.Stores, scope string, cd codec, r Record, future string) (outcome, error) {
	journal, uuids, err := providers(s)
	if err != nil {
		return lost, err
	}
	// A record dated in the future was taken as made at its first receipt. The
	// peer sends it again as long as its clock is ahead: that is the same
	// change, not a newer one, and must not beat what was edited here since.
	seen := ""
	if future != "" {
		seen = futureMarker(r.UUID)
		switch m, err := journal.Entry(ctx, scope, seen); {
		case err == nil && m.ChangedAt == future:
			return lost, nil
		case err != nil && !errors.Is(err, storage.ErrNotFound):
			return lost, err
		}
	}
	id, err := uuids.ObjectID(ctx, scope, r.Family, r.UUID)
	held := err == nil
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return lost, err
	}
	if !held && cd.locate != nil {
		// The same object under another name: the older uuid is kept on both
		// sides, whichever library meets the other first.
		var uuid string
		if id, uuid, held, err = cd.locate(ctx, s, scope, r); err != nil {
			return lost, err
		}
		if held {
			r.UUID = uuid
		}
	}
	local, err := journal.Entry(ctx, scope, r.UUID)
	hasEntry := err == nil
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return lost, err
	}
	switch {
	case hasEntry:
		localAt, err := normalize(local.ChangedAt)
		if err != nil {
			return lost, err
		}
		switch {
		case r.ChangedAt < localAt:
			return lost, nil
		case r.ChangedAt == localAt:
			var payload []byte
			if held && local.Op != storage.ChangeTrash && local.Op != storage.ChangeDelete {
				content, err := cd.read(ctx, s, scope, id)
				if err != nil {
					return lost, err
				}
				if payload, err = json.Marshal(content); err != nil {
					return lost, err
				}
			}
			remoteKey, localKey := tieKey(r.Op, r.Payload), tieKey(local.Op, payload)
			if cd.merge != nil && payload != nil && len(r.Payload) > 0 {
				winner, loser := []byte(r.Payload), payload
				if remoteKey < localKey {
					winner, loser = loser, winner
				}
				merged, err := cd.merge(winner, loser)
				if err != nil {
					return lost, err
				}
				if tieKey(storage.ChangeUpdate, merged) == localKey {
					return lost, nil
				}
				r.Payload = merged
			} else if remoteKey <= localKey {
				return lost, nil
			}
		}
	case held && r.ChangedAt <= floor:
		return lost, nil
	}

	res := modified
	switch r.Op {
	case storage.ChangeDelete, storage.ChangeTrash:
		res = recorded
		if held {
			res = removed
			if err := cd.remove(ctx, s, scope, id, r.Op == storage.ChangeTrash); err != nil {
				return lost, err
			}
		}
		if r.Op == storage.ChangeDelete && cd.trashKind != "" {
			// Gone for good: the entry the trash kept can no longer be restored.
			if err := trash.DiscardByUUID(ctx, s, scope, cd.trashKind, r.UUID); err != nil {
				return lost, err
			}
		}
	default:
		if held {
			err = cd.update(ctx, s, scope, id, r.Payload)
		} else {
			res = added
			if id, err = cd.create(ctx, s, scope, r.Payload); err == nil {
				if err = uuids.SetObjectUUID(ctx, scope, r.Family, id, r.UUID); err == nil {
					// It is back by this way: restoring the trash entry that held
					// it would make a second one.
					err = trash.DiscardByUUID(ctx, s, scope, cd.trashKind, r.UUID)
				}
			}
		}
		if errors.Is(err, storage.ErrConflict) {
			return conflicted, nil
		}
		if err != nil {
			return lost, err
		}
	}
	if err := journal.Mark(ctx, scope, r.Family, r.UUID, r.Op, r.ChangedAt); err != nil {
		return lost, err
	}
	if seen != "" {
		if err := journal.Mark(ctx, scope, r.Family, seen, storage.ChangeUpdate, future); err != nil {
			return lost, err
		}
	}
	return res, nil
}

// --- collection ---

// collectionPayload is a collection with its membership, by board hash, in
// order. A living collection (FilterQuery) has no membership rows. A member
// whose position this library does not hold is left out on import, and named in
// the report.
type collectionPayload struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	FilterQuery string   `json:"filterQuery,omitempty"`
	SortOrder   int      `json:"sortOrder"`
	Members     []string `json:"members,omitempty"`
}

var collectionCodec = codec{
	rank:      2,
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
		for _, pid := range ids {
			ref, err := refOf(ctx, s, scope, pid)
			if err != nil {
				return nil, err
			}
			p.Members = append(p.Members, ref)
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
	// A tie says neither side knows of the other's edit (two libraries adopted
	// at the floor, a page lost): no member leaves for that.
	merge: func(winner, loser json.RawMessage) (json.RawMessage, error) {
		var w, l collectionPayload
		if err := json.Unmarshal(winner, &w); err != nil {
			return nil, fmt.Errorf("collection payload: %w", err)
		}
		if err := json.Unmarshal(loser, &l); err != nil {
			return nil, fmt.Errorf("collection payload: %w", err)
		}
		if w.FilterQuery != "" {
			return winner, nil
		}
		for _, m := range l.Members {
			if !slices.Contains(w.Members, m) {
				w.Members = append(w.Members, m)
			}
		}
		return json.Marshal(w)
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
		var want []int64
		for _, ref := range p.Members {
			pid, found, err := findPosition(ctx, s, scope, ref)
			if err != nil {
				return err
			}
			if !found {
				noteMissing(ctx, ref)
				continue
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
// is not there.
func placeCollection(ctx context.Context, s storage.Stores, scope string, id int64, rank int) error {
	cs := s.Collections()
	var order []int64
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
	if err := cs.Reorder(ctx, scope, order); err != nil {
		return err
	}
	return nil
}

// markerFamily seeds the key of the journal entry that remembers a future-dated
// record (see apply). It is an entry of no object: an export finds no row for
// it and leaves it out.
const markerFamily = "changeset-future"

// futureMarker is the journal key remembering the future date of uuid's record.
func futureMarker(uuid string) string {
	return deriveMarker(uuid)
}

func deriveMarker(uuid string) string {
	sum := sha256.Sum256([]byte("blunderdb/changeset/1|" + markerFamily + "|" + uuid))
	h := hex.EncodeToString(sum[:])
	return h[0:8] + "-" + h[8:12] + "-7" + h[13:16] + "-8" + h[17:20] + "-" + h[20:32]
}

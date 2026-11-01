// Package changeset moves the change journal between two libraries: an export
// of what changed after a cursor, and an import that merges it, the last
// write winning (ADR to come with I3).
//
// A Set carries, per object, the last thing that happened to it
// (storage.Change) and, for a create or an update, the object's content. The
// families it speaks are the ones in codecs; the journal's other entries are
// passed over, the cursor moving past them. Everything is written against
// storage.Stores, so the desktop, the CLI and the daemon share it.
package changeset

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/trash"
)

// Record is one object's last change and, for a create or an update, its content.
type Record struct {
	storage.Change
	Payload json.RawMessage `json:"payload,omitempty"`
}

// Set is the changes after a cursor. Cursor is the seq of the last journal
// entry read (supported or not): the next export starts there. More says
// the limit cut the read short.
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
}

// codec reads and writes one family's content.
type codec struct {
	read   func(ctx context.Context, s storage.Stores, scope string, id int64) (any, error)
	create func(ctx context.Context, s storage.Stores, scope string, payload json.RawMessage) (int64, error)
	update func(ctx context.Context, s storage.Stores, scope string, id int64, payload json.RawMessage) error
	remove func(ctx context.Context, s storage.Stores, scope string, id int64, toTrash bool) error
}

var codecs = map[string]codec{"collection": collectionCodec}

type collectionPayload struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

var collectionCodec = codec{
	read: func(ctx context.Context, s storage.Stores, scope string, id int64) (any, error) {
		c, err := s.Collections().Get(ctx, scope, id)
		if err != nil {
			return nil, err
		}
		return collectionPayload{Name: c.Name, Description: c.Description}, nil
	},
	create: func(ctx context.Context, s storage.Stores, scope string, raw json.RawMessage) (int64, error) {
		var p collectionPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			return 0, fmt.Errorf("collection payload: %w", err)
		}
		return s.Collections().Create(ctx, scope, p.Name, p.Description)
	},
	update: func(ctx context.Context, s storage.Stores, scope string, id int64, raw json.RawMessage) error {
		var p collectionPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			return fmt.Errorf("collection payload: %w", err)
		}
		return s.Collections().Update(ctx, scope, id, p.Name, p.Description)
	},
	remove: func(ctx context.Context, s storage.Stores, scope string, id int64, toTrash bool) error {
		if toTrash {
			_, err := trash.Collection(ctx, s, scope, id)
			return err
		}
		return s.Collections().Delete(ctx, scope, id)
	},
}

func providers(s storage.Stores) (storage.ChangeJournalStore, storage.ObjectUUIDStore, error) {
	j, ok1 := s.(storage.ChangeJournalProvider)
	u, ok2 := s.(storage.ObjectUUIDProvider)
	if !ok1 || !ok2 {
		return nil, nil, errors.New("changeset: the backend keeps no change journal")
	}
	return j.ChangeJournal(), u.ObjectUUIDs(), nil
}

// Export reads up to limit journal entries after the cursor and loads the
// content of the creates and updates. An object that vanished between the
// journal read and the content read is left out: its delete is a later entry.
func Export(ctx context.Context, s storage.Stores, scope string, after int64, limit int) (Set, error) {
	journal, uuids, err := providers(s)
	if err != nil {
		return Set{}, err
	}
	if limit <= 0 {
		limit = 1000
	}
	changes, err := journal.Changes(ctx, scope, after, limit+1)
	if err != nil {
		return Set{}, err
	}
	set := Set{Cursor: after}
	if len(changes) > limit {
		changes, set.More = changes[:limit], true
	}
	for _, c := range changes {
		set.Cursor = c.Seq
		cd, ok := codecs[c.Family]
		if !ok {
			continue
		}
		rec := Record{Change: c}
		if c.Op == storage.ChangeCreate || c.Op == storage.ChangeUpdate {
			id, err := uuids.ObjectID(ctx, scope, c.Family, c.UUID)
			if errors.Is(err, storage.ErrNotFound) {
				continue
			}
			if err != nil {
				return Set{}, err
			}
			content, err := cd.read(ctx, s, scope, id)
			if err != nil {
				return Set{}, err
			}
			if rec.Payload, err = json.Marshal(content); err != nil {
				return Set{}, err
			}
		}
		set.Records = append(set.Records, rec)
	}
	return set, nil
}

// Import merges a set into the library: per object, the later change wins
// (ChangedAt is UTC RFC 3339, so it compares as text); an equal time keeps
// the local one, which makes a record that comes back around a no-op. A
// restore therefore beats an older delete and a delete an older update. What
// is applied keeps the time it had where it was made, so it is not re-sent
// as newer. A delete of an object this library never held is still recorded,
// so an older create that arrives later loses.
func Import(ctx context.Context, s storage.Stores, scope string, set Set) (Report, error) {
	journal, uuids, err := providers(s)
	if err != nil {
		return Report{}, err
	}
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
		local, err := journal.Entry(ctx, scope, r.UUID)
		if err != nil && !errors.Is(err, storage.ErrNotFound) {
			return rep, err
		}
		if err == nil && local.ChangedAt >= r.ChangedAt {
			rep.Older++
			continue
		}
		id, err := uuids.ObjectID(ctx, scope, r.Family, r.UUID)
		held := err == nil
		if err != nil && !errors.Is(err, storage.ErrNotFound) {
			return rep, err
		}
		switch r.Op {
		case storage.ChangeDelete, storage.ChangeTrash:
			if held {
				if err := cd.remove(ctx, s, scope, id, r.Op == storage.ChangeTrash); err != nil {
					return rep, err
				}
			}
		default:
			if held {
				err = cd.update(ctx, s, scope, id, r.Payload)
			} else if id, err = cd.create(ctx, s, scope, r.Payload); err == nil {
				err = uuids.SetObjectUUID(ctx, scope, r.Family, id, r.UUID)
			}
			if errors.Is(err, storage.ErrConflict) {
				rep.Conflicts = append(rep.Conflicts, r.UUID)
				continue
			}
			if err != nil {
				return rep, err
			}
		}
		if err := journal.Mark(ctx, scope, r.Family, r.UUID, r.Op, r.ChangedAt); err != nil {
			return rep, err
		}
		rep.Applied++
	}
	return rep, nil
}

package storage

import (
	"context"
	"fmt"
)

// Presence of a match, a tournament, a collection or a lesson in other
// libraries: whether each holds what a copy would land on. The keys are the
// copy's own (ingest.Transfer, through the native import): a match is the
// same match by its fingerprint (MatchStore.FindByHash, match_hash or
// canonical_hash), a tournament, a collection and a lesson by their name.
// Like Elsewhere, it reads every source and writes nothing.

// Kinds of object a presence read names.
const (
	ObjectMatch      = "match"
	ObjectTournament = "tournament"
	ObjectCollection = "collection"
	ObjectLesson     = "lesson"
)

// ObjectQuery names an object by the key a copy merges it on: Hash and
// CanonicalHash for a match, Name for the others.
type ObjectQuery struct {
	Kind          string `json:"kind"`
	Hash          string `json:"hash,omitempty"`
	CanonicalHash string `json:"canonicalHash,omitempty"`
	Name          string `json:"name,omitempty"`
}

// ObjectPresence is what one source holds under the key: ID 0 when nothing.
// Count is what the held object holds — a tournament's matches, a
// collection's positions, a lesson's steps, a match's games — so that a
// namesake with other contents reads as such.
type ObjectPresence struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// ObjectKey reads the key of object id of kind in scope of s.
func ObjectKey(ctx context.Context, s Stores, scope, kind string, id int64) (ObjectQuery, error) {
	q := ObjectQuery{Kind: kind}
	switch kind {
	case ObjectMatch:
		m, err := s.Matches().Get(ctx, scope, id)
		if err != nil {
			return q, err
		}
		q.Hash, q.CanonicalHash = m.MatchHash, m.CanonicalHash
	case ObjectTournament:
		t, err := s.Tournaments().Get(ctx, scope, id)
		if err != nil {
			return q, err
		}
		q.Name = t.Name
	case ObjectCollection:
		c, err := s.Collections().Get(ctx, scope, id)
		if err != nil {
			return q, err
		}
		q.Name = c.Name
	case ObjectLesson:
		l, err := s.Lessons().Get(ctx, scope, id)
		if err != nil {
			return q, err
		}
		q.Name = l.Name
	default:
		return q, fmt.Errorf("%w: unknown object kind %q", ErrInvalid, kind)
	}
	return q, nil
}

// ElsewhereObject reads, from every source in order, whether it holds the
// object q names.
func ElsewhereObject(ctx context.Context, sources []Source, q ObjectQuery) ([]Tagged[ObjectPresence], error) {
	switch q.Kind {
	case ObjectMatch:
		if q.Hash == "" && q.CanonicalHash == "" {
			return nil, fmt.Errorf("%w: a match is named by its hash or canonical hash", ErrInvalid)
		}
	case ObjectTournament, ObjectCollection, ObjectLesson:
		if q.Name == "" {
			return nil, fmt.Errorf("%w: a %s is named by its name", ErrInvalid, q.Kind)
		}
	default:
		return nil, fmt.Errorf("%w: unknown object kind %q", ErrInvalid, q.Kind)
	}
	return ReadSources(ctx, sources, func(ctx context.Context, src Source) (ObjectPresence, error) {
		return objectIn(ctx, src.Storage, src.Scope, q)
	})
}

func objectIn(ctx context.Context, s Storage, scope string, q ObjectQuery) (ObjectPresence, error) {
	var out ObjectPresence
	switch q.Kind {
	case ObjectMatch:
		id, found, err := s.Matches().FindByHash(ctx, scope, q.Hash, q.CanonicalHash)
		if err != nil || !found {
			return out, err
		}
		m, err := s.Matches().Get(ctx, scope, id)
		if err != nil {
			return out, err
		}
		out.ID, out.Name, out.Count = id, m.Player1Name+" – "+m.Player2Name, int(m.GameCount)
	case ObjectTournament:
		for t, err := range s.Tournaments().List(ctx, scope, ListOpts{}) {
			if err != nil {
				return out, err
			}
			if t.Name == q.Name {
				out.ID, out.Name = t.ID, t.Name
				break
			}
		}
		if out.ID == 0 {
			return out, nil
		}
		for _, err := range s.Tournaments().Matches(ctx, scope, out.ID) {
			if err != nil {
				return out, err
			}
			out.Count++
		}
	case ObjectCollection:
		for c, err := range s.Collections().List(ctx, scope) {
			if err != nil {
				return out, err
			}
			if c.Name == q.Name {
				out.ID, out.Name = c.ID, c.Name
				break
			}
		}
		if out.ID == 0 {
			return out, nil
		}
		ids, err := s.Collections().PositionIDs(ctx, scope, out.ID, ListOpts{})
		if err != nil {
			return out, err
		}
		out.Count = len(ids)
	case ObjectLesson:
		lessons, err := s.Lessons().List(ctx, scope)
		if err != nil {
			return out, err
		}
		for _, l := range lessons {
			if l.Name != q.Name {
				continue
			}
			full, err := s.Lessons().Get(ctx, scope, l.ID)
			if err != nil {
				return out, err
			}
			out.ID, out.Name, out.Count = l.ID, l.Name, len(full.Steps)
			break
		}
	default:
		return out, fmt.Errorf("%w: unknown object kind %q", ErrInvalid, q.Kind)
	}
	return out, nil
}

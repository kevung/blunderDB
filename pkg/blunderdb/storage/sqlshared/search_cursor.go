package sqlshared

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// searchCursor is where a page read by cursor resumes: after the row (key, id) of
// the order "key [DESC] NULLS LAST, p.id". The next page is a WHERE condition, not
// an OFFSET, so its cost does not grow with its depth, and a row saved meanwhile
// lands on its side of the anchor without shifting the others.
type searchCursor struct {
	col  string // the sort key, "" for the id order
	desc bool
	null bool // the anchor has no key: only the NULL tail is left
	key  any
	id   int64
}

// wireCursor is the cursor as the client holds it, base64url JSON. Fingerprint is
// a digest of the scope, filters and sort that issued it: it tells a cursor replayed
// on another search from a mistake, and protects nothing — the client can read and
// rewrite every field. Tenant isolation does not rest on it but on the tenant filter
// every statement of the page carries; the anchor is therefore checked for its shape
// and its type only, as any client input.
type wireCursor struct {
	Fingerprint string `json:"f"`
	ID          int64  `json:"id"`
	Null        bool   `json:"n,omitempty"`
	Int         *int64 `json:"i,omitempty"`
	Time        *int64 `json:"t,omitempty"` // Unix nanoseconds: 1678 to 2262, the range of time.UnixNano
	Bool        *bool  `json:"b,omitempty"`
}

func cursorFingerprint(scope string, f domain.SearchFilters) string {
	raw, _ := json.Marshal(struct {
		Scope   string
		Filters domain.SearchFilters
	}{scope, f})
	sum := sha256.Sum256(raw)
	return base64.RawURLEncoding.EncodeToString(sum[:9])
}

// keyKind is the type a sort column is read back and bound as.
type keyKind uint8

const (
	kindNone keyKind = iota // the id order: the anchor is an id alone
	kindInt
	kindTime
	kindBool
)

// cursorKeyKind is the kind of col on backend. SQLite stores every key as an integer;
// PostgreSQL binds a key only to a column of its own type, and fails the statement
// otherwise.
func cursorKeyKind(backend, col string) keyKind {
	switch {
	case col == "":
		return kindNone
	case backend != "postgres":
		return kindInt
	case col == "a.is_close_cube":
		return kindBool
	case col == "p.match_date":
		return kindTime
	}
	return kindInt
}

func invalidCursor(why string) error {
	return fmt.Errorf("%w: search cursor %s", storage.ErrInvalid, why)
}

func decodeSearchCursor(raw, backend, scope string, f domain.SearchFilters) (*searchCursor, error) {
	col, desc, ok := sortKey(f.Sort)
	if !ok {
		return nil, invalidCursor("for an unknown sort")
	}
	blob, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, invalidCursor("unreadable")
	}
	dec := json.NewDecoder(bytes.NewReader(blob))
	dec.DisallowUnknownFields()
	var w wireCursor
	if err := dec.Decode(&w); err != nil || dec.More() {
		return nil, invalidCursor("unreadable")
	}
	if w.Fingerprint != cursorFingerprint(scope, f) {
		return nil, invalidCursor("issued by another search")
	}
	if w.ID < 1 {
		return nil, invalidCursor("anchored on no row")
	}
	c := &searchCursor{col: col, desc: desc, null: w.Null, id: w.ID}
	var kinds []keyKind
	if w.Int != nil {
		kinds, c.key = append(kinds, kindInt), *w.Int
	}
	if w.Time != nil {
		kinds, c.key = append(kinds, kindTime), time.Unix(0, *w.Time).UTC()
	}
	if w.Bool != nil {
		kinds, c.key = append(kinds, kindBool), *w.Bool
	}
	want := cursorKeyKind(backend, col)
	switch {
	case want == kindNone && (w.Null || len(kinds) > 0):
		return nil, invalidCursor("with a key the id order has not")
	case want == kindNone:
	case w.Null && len(kinds) > 0:
		return nil, invalidCursor("both with and without a key")
	case w.Null:
	case len(kinds) != 1:
		return nil, invalidCursor("without one key")
	case kinds[0] != want:
		return nil, invalidCursor("with a key of another type than its sort")
	}
	return c, nil
}

func encodeSearchCursor(backend, scope string, f domain.SearchFilters, id int64, key any) (string, error) {
	col, _, _ := sortKey(f.Sort)
	want := cursorKeyKind(backend, col)
	w := wireCursor{Fingerprint: cursorFingerprint(scope, f), ID: id}
	var got keyKind
	switch v := key.(type) {
	case nil:
		w.Null, got = want != kindNone, want
	case int64:
		w.Int, got = &v, kindInt
	case int32:
		n := int64(v)
		w.Int, got = &n, kindInt
	case int16:
		n := int64(v)
		w.Int, got = &n, kindInt
	case int:
		n := int64(v)
		w.Int, got = &n, kindInt
	case time.Time:
		n := v.UnixNano()
		w.Time, got = &n, kindTime
	case bool:
		w.Bool, got = &v, kindBool
	}
	if got != want {
		return "", fmt.Errorf("%w: search cursor key of type %T on %q", storage.ErrInternal, key, col)
	}
	blob, _ := json.Marshal(w)
	return base64.RawURLEncoding.EncodeToString(blob), nil
}

// keyedCond restricts the rows that have a key to those after the anchor; skip says
// none is left (the anchor is already in the NULL tail). `key >= v` bounds an index
// range on the key; the OR only sorts out the anchor's ties.
func (c *searchCursor) keyedCond() (cond string, args []any, skip bool) {
	if c.null {
		return "", nil, true
	}
	if c.desc {
		return " AND " + c.col + " <= ? AND (" + c.col + " < ? OR p.id > ?)", []any{c.key, c.key, c.id}, false
	}
	return " AND " + c.col + " >= ? AND (" + c.col + " > ? OR p.id > ?)", []any{c.key, c.key, c.id}, false
}

// tailCond restricts the rows without a key: all of them follow a keyed anchor.
func (c *searchCursor) tailCond() (string, []any) {
	if c.null {
		return " AND p.id > ?", []any{c.id}
	}
	return "", nil
}

// cond is the whole condition, for a scan that does not split keyed rows from the tail.
func (c *searchCursor) cond() (string, []any) {
	if c.col == "" {
		return " AND p.id > ?", []any{c.id}
	}
	if c.null {
		return " AND " + c.col + " IS NULL AND p.id > ?", []any{c.id}
	}
	keyed, args, _ := c.keyedCond()
	return " AND ((" + keyed[len(" AND "):] + ") OR " + c.col + " IS NULL)", args
}

// FindPage implements storage.SearchStore.
func (s *SearchStore) FindPage(ctx context.Context, scope string, f domain.SearchFilters, opts storage.PageOpts) ([]domain.Position, string, error) {
	wc, err := s.pageWhere(ctx, scope, f, opts)
	if err != nil {
		return nil, "", err
	}
	positions, err := s.findWith(ctx, f, wc, storage.ListOpts{Limit: opts.Limit + 1, Offset: opts.Offset})
	if err != nil || len(positions) <= opts.Limit {
		return positions, "", err
	}
	positions = positions[:opts.Limit]
	pageRead(positions[len(positions)-1].ID)
	next, err := s.nextCursor(scope, f, wc, positions[len(positions)-1].ID)
	return positions, next, err
}

// FindIDsPage implements storage.SearchStore.
func (s *SearchStore) FindIDsPage(ctx context.Context, scope string, f domain.SearchFilters, opts storage.PageOpts) ([]int64, string, error) {
	wc, err := s.pageWhere(ctx, scope, f, opts)
	if err != nil {
		return nil, "", err
	}
	ids, err := s.findIDsWith(ctx, f, wc, storage.ListOpts{Limit: opts.Limit + 1, Offset: opts.Offset})
	if err != nil || len(ids) <= opts.Limit {
		return ids, "", err
	}
	ids = ids[:opts.Limit]
	pageRead(ids[len(ids)-1])
	next, err := s.nextCursor(scope, f, wc, ids[len(ids)-1])
	return ids, next, err
}

func (s *SearchStore) pageWhere(ctx context.Context, scope string, f domain.SearchFilters, opts storage.PageOpts) (searchWhereClause, error) {
	if opts.Limit <= 0 || opts.Limit >= maxWindowInline {
		return searchWhereClause{}, fmt.Errorf("%w: a page needs a limit between 1 and %d", storage.ErrInvalid, maxWindowInline-1)
	}
	if opts.Cursor != "" && opts.Offset != 0 {
		return searchWhereClause{}, fmt.Errorf("%w: a page resumes from a cursor or an offset, not both", storage.ErrInvalid)
	}
	var after *searchCursor
	if opts.Cursor != "" {
		var err error
		if after, err = decodeSearchCursor(opts.Cursor, s.DB.Name(), scope, f); err != nil {
			return searchWhereClause{}, err
		}
	}
	wc, err := s.buildWhere(ctx, scope, f)
	wc.after, wc.keys = after, map[int64]any{}
	return wc, err
}

// nextCursor anchors the next page on the last row of this one, at the sort key the
// window read with it.
func (s *SearchStore) nextCursor(scope string, f domain.SearchFilters, wc searchWhereClause, id int64) (string, error) {
	col, _, _ := sortKey(f.Sort)
	key, read := wc.keys[id]
	if col != "" && !read {
		return "", fmt.Errorf("%w: search cursor on row %d, which the window did not read", storage.ErrInternal, id)
	}
	return encodeSearchCursor(s.DB.Name(), scope, f, id, key)
}

// pageReadHook runs between the read of a page and the issue of its cursor.
var pageReadHook atomic.Pointer[func(lastID int64)]

func pageRead(lastID int64) {
	if h := pageReadHook.Load(); h != nil {
		(*h)(lastID)
	}
}

// SetPageReadHook makes fn run on the last id of every page read by cursor, before
// its next cursor is issued, and returns a func removing it. For tests only: it
// places a write between the two exactly.
func SetPageReadHook(fn func(lastID int64)) (restore func()) {
	pageReadHook.Store(&fn)
	return func() { pageReadHook.Store(nil) }
}

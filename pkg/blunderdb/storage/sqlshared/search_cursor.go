package sqlshared

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
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

// wireCursor is the cursor as the client holds it, base64url JSON. Seal ties it to
// the scope and filters that issued it, so that it cannot be replayed on another
// search, another sort or another tenant.
type wireCursor struct {
	Seal string `json:"s"`
	ID   int64  `json:"id"`
	Null bool   `json:"n,omitempty"`
	Int  *int64 `json:"i,omitempty"`
	Time *int64 `json:"t,omitempty"` // Unix nanoseconds
	Bool *bool  `json:"b,omitempty"`
}

func cursorSeal(scope string, f domain.SearchFilters) string {
	raw, _ := json.Marshal(struct {
		Scope   string
		Filters domain.SearchFilters
	}{scope, f})
	sum := sha256.Sum256(raw)
	return base64.RawURLEncoding.EncodeToString(sum[:9])
}

func invalidCursor(why string) error {
	return fmt.Errorf("%w: search cursor %s", storage.ErrInvalid, why)
}

func decodeSearchCursor(raw, scope string, f domain.SearchFilters) (*searchCursor, error) {
	col, desc, ok := sortKey(f.Sort)
	if !ok {
		return nil, invalidCursor("for an unknown sort")
	}
	blob, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, invalidCursor("unreadable")
	}
	var w wireCursor
	if err := json.Unmarshal(blob, &w); err != nil {
		return nil, invalidCursor("unreadable")
	}
	if w.Seal != cursorSeal(scope, f) {
		return nil, invalidCursor("issued by another search")
	}
	c := &searchCursor{col: col, desc: desc, null: w.Null, id: w.ID}
	switch {
	case col == "" || w.Null:
	case w.Int != nil:
		c.key = *w.Int
	case w.Time != nil:
		c.key = time.Unix(0, *w.Time).UTC()
	case w.Bool != nil:
		c.key = *w.Bool
	default:
		return nil, invalidCursor("without its key")
	}
	return c, nil
}

func encodeSearchCursor(scope string, f domain.SearchFilters, id int64, key any) (string, error) {
	w := wireCursor{Seal: cursorSeal(scope, f), ID: id}
	switch v := key.(type) {
	case nil:
		w.Null = true
	case int64:
		w.Int = &v
	case int32:
		n := int64(v)
		w.Int = &n
	case int16:
		n := int64(v)
		w.Int = &n
	case int:
		n := int64(v)
		w.Int = &n
	case time.Time:
		n := v.UnixNano()
		w.Time = &n
	case bool:
		w.Bool = &v
	default:
		return "", fmt.Errorf("%w: search cursor key of type %T", storage.ErrInternal, key)
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
	next, err := s.nextCursor(ctx, scope, f, positions[len(positions)-1].ID)
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
	next, err := s.nextCursor(ctx, scope, f, ids[len(ids)-1])
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
		if after, err = decodeSearchCursor(opts.Cursor, scope, f); err != nil {
			return searchWhereClause{}, err
		}
	}
	wc, err := s.buildWhere(ctx, scope, f)
	wc.after = after
	return wc, err
}

// nextCursor reads the sort key of the last row of a page.
func (s *SearchStore) nextCursor(ctx context.Context, scope string, f domain.SearchFilters, id int64) (string, error) {
	col, _, _ := sortKey(f.Sort)
	var key any
	if col != "" {
		tenant, tenantArgs := s.DB.TenantFilter("p", scope)
		err := s.DB.QueryRow(ctx, `SELECT `+col+` FROM position p
	LEFT JOIN analysis a ON a.position_id = p.id`+andSameTenant(s.DB, "a", "p")+`
	WHERE `+tenant+` AND p.id = ?`, append(tenantArgs, id)...).Scan(&key)
		if err != nil && !errors.Is(err, ErrNoRows) {
			return "", errf(s.DB, "search cursor key", err)
		}
	}
	return encodeSearchCursor(scope, f, id, key)
}

package document

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// ErrUnverified refuses an import that holds an entry neither sure nor
// verified. It matches storage.ErrInvalid, so the daemon answers it with a
// 400.
var ErrUnverified = fmt.Errorf("document: entries neither sure nor verified: %w", storage.ErrInvalid)

// ImportOptions are a caller's choices for Import.
type ImportOptions struct {
	// Collection names the Collection; the document's title when empty. An
	// existing Collection of that name receives the positions it lacks.
	Collection string `json:"collection"`
	// Force imports entries neither sure nor verified.
	Force bool `json:"force"`
}

// ImportResult reports what Import wrote.
type ImportResult struct {
	CollectionID int64  `json:"collectionId"`
	Collection   string `json:"collection"`
	// CreatedCollection is false when the Collection already existed.
	CreatedCollection bool `json:"createdCollection"`
	Entries           int  `json:"entries"`
	// Positions counts the distinct positions of the document.
	Positions int `json:"positions"`
	// NewPositions counts the positions the library did not hold yet.
	NewPositions int `json:"newPositions"`
	// Added counts the positions put into the Collection by this import.
	Added int `json:"added"`
	// Repeated counts the entries whose position the document showed
	// earlier: their reference joins the first appearance.
	Repeated int `json:"repeated"`
	// Forced counts the entries imported only because Force was set.
	Forced int `json:"forced"`
}

// Unverified returns the ranks of the entries neither sure nor verified.
func Unverified(r *Releve) []int {
	var ranks []int
	for _, e := range r.Entries {
		if !e.Importable() {
			ranks = append(ranks, e.Rank)
		}
	}
	return ranks
}

// Description is the Collection's description for the document: its title,
// author, edition and URL, one per line.
func (s Source) Description() string {
	var lines []string
	for _, l := range []string{s.Title, s.Author, s.Edition, s.URL} {
		if l = strings.TrimSpace(l); l != "" {
			lines = append(lines, l)
		}
	}
	return strings.Join(lines, "\n")
}

// commentPrefix opens the comment that carries a position's references in
// this document; Import finds its own comment again by it.
func (s Source) commentPrefix() string { return strings.TrimSpace(s.Title) + " — " }

// Import writes the Relevé into one Collection, in one transaction: every
// position as a position brought in on its own (ADR-0001), in the order of
// the document, a position the document shows twice belonging once, at its
// first appearance, with all its references in one comment. Importing the
// same Relevé again writes nothing new. An entry neither sure nor verified
// refuses the import, before anything is written, unless opts.Force.
func Import(ctx context.Context, st storage.Storage, scope string, r *Releve, opts ImportOptions) (ImportResult, error) {
	if err := r.Validate(); err != nil {
		return ImportResult{}, err
	}
	unverified := Unverified(r)
	if len(unverified) > 0 && !opts.Force {
		return ImportResult{}, fmt.Errorf("%w: rank %s; verify them, or force the import", ErrUnverified, joinInts(unverified))
	}
	name := strings.TrimSpace(opts.Collection)
	if name == "" {
		name = strings.TrimSpace(r.Document.Title)
	}
	tx, err := st.BeginTx(ctx)
	if err != nil {
		return ImportResult{}, err
	}
	defer func() { _ = tx.Rollback() }()
	res, err := importTx(ctx, tx, scope, r, name)
	if err != nil {
		return ImportResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return ImportResult{}, err
	}
	res.Forced = len(unverified)
	return res, nil
}

func importTx(ctx context.Context, st storage.Stores, scope string, r *Releve, name string) (ImportResult, error) {
	res := ImportResult{Collection: name, Entries: len(r.Entries)}
	cid, created, err := documentCollection(ctx, st, scope, name, r.Document.Description())
	if err != nil {
		return res, err
	}
	res.CollectionID, res.CreatedCollection = cid, created

	var order []int64
	refs := map[int64][]string{}
	for _, e := range r.Entries {
		pos, err := ParsePosition(e.Position)
		if err != nil {
			return res, fmt.Errorf("%w: rank %d: %w", ErrInvalid, e.Rank, err)
		}
		pos.IndividuallyImported = true
		id, isNew, err := st.Positions().SaveCreated(ctx, scope, &pos)
		if err != nil {
			return res, fmt.Errorf("document: rank %d: %w", e.Rank, err)
		}
		if isNew {
			res.NewPositions++
		}
		if _, seen := refs[id]; seen {
			res.Repeated++
		} else {
			order = append(order, id)
		}
		refs[id] = append(refs[id], e.Reference())
	}
	res.Positions = len(order)

	members, err := st.Collections().PositionIDs(ctx, scope, cid, storage.ListOpts{})
	if err != nil {
		return res, err
	}
	held := make(map[int64]bool, len(members))
	for _, id := range members {
		held[id] = true
	}
	var add []int64
	for _, id := range order {
		if !held[id] {
			add = append(add, id)
		}
	}
	if len(add) > 0 {
		if err := st.Collections().AddPositions(ctx, scope, cid, add); err != nil {
			return res, err
		}
	}
	res.Added = len(add)

	prefix := r.Document.commentPrefix()
	for _, id := range order {
		if err := writeReference(ctx, st, scope, id, prefix, prefix+strings.Join(refs[id], "; ")); err != nil {
			return res, err
		}
	}
	return res, nil
}

// documentCollection finds the Collection called name or creates it with the
// document's description. An existing Collection keeps a description it has.
func documentCollection(ctx context.Context, st storage.Stores, scope, name, description string) (int64, bool, error) {
	var found *storage.Collection
	for c, err := range st.Collections().List(ctx, scope) {
		if err != nil {
			return 0, false, err
		}
		if found == nil && c.Name == name {
			found = c
		}
	}
	if found == nil {
		id, err := st.Collections().Create(ctx, scope, name, description)
		return id, true, err
	}
	if strings.TrimSpace(found.FilterQuery) != "" {
		return 0, false, fmt.Errorf("%w: collection %q is living, it holds no hand-picked positions", storage.ErrInvalid, name)
	}
	if strings.TrimSpace(found.Description) == "" {
		if err := st.Collections().Update(ctx, scope, found.ID, found.Name, description); err != nil {
			return 0, false, err
		}
	}
	return found.ID, false, nil
}

// writeReference keeps one comment per position and document: the comment
// that opens with prefix is rewritten when its references changed, added when
// there is none.
func writeReference(ctx context.Context, st storage.Stores, scope string, positionID int64, prefix, text string) error {
	var mine *domain.CommentEntry
	for c, err := range st.Comments().ByPosition(ctx, scope, positionID) {
		if err != nil {
			return err
		}
		if mine == nil && strings.HasPrefix(c.Text, prefix) {
			mine = c
		}
	}
	switch {
	case mine == nil:
		_, err := st.Comments().Add(ctx, scope, positionID, text)
		return err
	case mine.Text != text:
		return st.Comments().Update(ctx, scope, mine.ID, text)
	}
	return nil
}

func joinInts(ns []int) string {
	s := make([]string, len(ns))
	for i, n := range ns {
		s[i] = strconv.Itoa(n)
	}
	return strings.Join(s, ", ")
}

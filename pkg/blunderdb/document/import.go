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
	// AcceptComplete imports, as a whole, the entries whose fifteen checkers
	// a side were read with no doubt and whose context is certified
	// (Releve.CompleteCertified), neither sure nor verified.
	AcceptComplete bool `json:"acceptComplete"`
	// Fallback imports, rather than refuse, the entries whose checkers are
	// certified and whose context is to verify, under the context their
	// position carries (Releve.FallbackCertified); each one's reference
	// names the fields to check against the diagram.
	Fallback bool `json:"fallback"`
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
	// Complete counts the entries imported because AcceptComplete was set.
	Complete int `json:"complete"`
	// Fallback counts the entries imported under their fallback context
	// because Fallback was set.
	Fallback int `json:"fallback"`
	// Forced counts the entries imported only because Force was set.
	Forced int `json:"forced"`
}

// Unverified returns the ranks of the entries an import refuses unless it is
// forced: neither sure nor verified, nor complete when acceptComplete.
func Unverified(r *Releve, acceptComplete bool) []int {
	r = r.Upgrade()
	var ranks []int
	for _, e := range r.Entries {
		if !r.Importable(e, acceptComplete) {
			ranks = append(ranks, e.Rank)
		}
	}
	return ranks
}

// FallbackRanks returns the ranks of the entries an import refuses but
// accepts under their fallback context (ImportOptions.Fallback).
func FallbackRanks(r *Releve, acceptComplete bool) []int {
	r = r.Upgrade()
	var ranks []int
	for _, e := range r.Entries {
		if r.FallbackCertified(e, acceptComplete) {
			ranks = append(ranks, e.Rank)
		}
	}
	return ranks
}

// refusalHint says what a person does about the refused ranks.
func refusalHint(r *Releve, ranks []int, acceptComplete, fallback bool) string {
	want := map[int]bool{}
	for _, k := range ranks {
		want[k] = true
	}
	var defaults, direction, complete, context, repli bool
	for _, e := range r.Entries {
		if !want[e.Rank] {
			continue
		}
		if e.Context == ContextDefault && !r.ContextCertified(e) {
			defaults = true
		}
		if e.Complete && r.ContextCertified(e) && !r.DirectionCertified(e) {
			direction = true
		}
		if !acceptComplete && e.Complete && r.ContextCertified(e) && r.DirectionCertified(e) {
			complete = true
		}
		if !r.ContextCertified(e) {
			context = true
		}
		if !fallback && r.FallbackCertified(e, acceptComplete) {
			repli = true
		}
	}
	var hints []string
	if defaults {
		hints = append(hints, "confirm the document's defaults once if its diagrams follow them (\"defaults\": {\"confirmed\": true})")
	}
	if direction {
		hints = append(hints, "check the direction of play the reader assumed on a few diagrams, then confirm it once (\"defaults\": {\"direction\": {\"confirmed\": true}})")
	}
	if complete {
		hints = append(hints, "accept the entries of fifteen checkers a side read with no doubt as a whole (accept complete)")
	}
	if context {
		hints = append(hints, "check the cube, dice, player on roll and score of an entry whose context is not certified against its diagram")
	}
	if repli {
		hints = append(hints, "or import the entries whose checkers are certified under the context their position carries, each reference naming the fields to check (fallback)")
	}
	hints = append(hints, "verify the others (\"verified\": true)")
	return strings.Join(hints, "; ")
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
// refuses the import, before anything is written, unless opts.Force; an
// entry complete with its context certified is accepted when
// opts.AcceptComplete, and an entry whose checkers are certified and whose
// context is to verify is accepted under its fallback context when
// opts.Fallback, its reference naming what to check.
func Import(ctx context.Context, st storage.Storage, scope string, r *Releve, opts ImportOptions) (ImportResult, error) {
	if err := r.Validate(); err != nil {
		return ImportResult{}, err
	}
	r = r.Upgrade()
	var unread []int
	for _, e := range r.Entries {
		if e.Position == "" {
			unread = append(unread, e.Rank)
		}
	}
	if len(unread) > 0 {
		return ImportResult{}, fmt.Errorf("%w: rank %s has no position; fill it in or remove the entry", ErrInvalid, joinInts(unread))
	}
	fallback := map[int]bool{}
	if opts.Fallback {
		for _, k := range FallbackRanks(r, opts.AcceptComplete) {
			fallback[k] = true
		}
	}
	var unverified []int
	for _, k := range Unverified(r, opts.AcceptComplete) {
		if !fallback[k] {
			unverified = append(unverified, k)
		}
	}
	if len(unverified) > 0 && !opts.Force {
		return ImportResult{}, fmt.Errorf("%w: rank %s; %s, or force the import", ErrUnverified, joinInts(unverified), refusalHint(r, unverified, opts.AcceptComplete, opts.Fallback))
	}
	complete := 0
	if opts.AcceptComplete {
		for _, e := range r.Entries {
			if r.CompleteCertified(e) {
				complete++
			}
		}
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
	res, err := importTx(ctx, tx, scope, r, name, fallback)
	if err != nil {
		return ImportResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return ImportResult{}, err
	}
	res.Complete = complete
	res.Fallback = len(fallback)
	res.Forced = len(unverified)
	return res, nil
}

func importTx(ctx context.Context, st storage.Stores, scope string, r *Releve, name string, fallback map[int]bool) (ImportResult, error) {
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
		ref := e.Reference()
		if fallback[e.Rank] {
			ref += " (" + e.FallbackNote() + ")"
		}
		refs[id] = append(refs[id], ref)
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

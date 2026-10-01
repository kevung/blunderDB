package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// What a read depends on, condensed into a token that changes whenever one of its stored
// sources does — without replaying any Direction. A reader that already holds the answer for a
// token need not ask again (the serve daemon's ETag, ADR-0057 rule 4).
//
// The token covers every stored fact a read is built from: each Direction's record, the length
// and head of its journal, its filled Slots, its Tournament's row (the name the pages show),
// the Rencontre's row, and — for the Slots view — the drafts started from a Slot. What it does
// not cover is the wall clock: proposals, pages and the clock also depend on the time of the
// reading, and a caller that caches on this token bounds that drift itself.

// ErrStale refuses a gesture whose caller stated a version that is no longer the current one:
// someone else wrote in between. The caller reads the fresh state and decides again.
var ErrStale = fmt.Errorf("%w: the version moved since it was read", storage.ErrConflict)

type expectKey struct{}

// expectation is what a caller states before a gesture, and what the gesture leaves it.
type expectation struct {
	want   string
	result string
}

// ExpectVersion returns ctx carrying the version its caller last read (DirectionVersion, or
// RencontreVersion for a gesture on a room). A gesture run under it compares that version
// with the current one under its own lock, before it writes anything, and refuses with
// ErrStale when they differ — so of two gestures sent on one reading, one only applies.
func ExpectVersion(ctx context.Context, version string) context.Context {
	return context.WithValue(ctx, expectKey{}, &expectation{want: version})
}

// ResultVersion is the version of the view a gesture run under ExpectVersion handed back
// (ViewAfter). Empty when the gesture returned no view — it removed what it named — or when
// writers kept that view and its version from agreeing.
func ResultVersion(ctx context.Context) string {
	if e, ok := expectedVersion(ctx); ok {
		return e.result
	}
	return ""
}

func expectedVersion(ctx context.Context) (*expectation, bool) {
	e, ok := ctx.Value(expectKey{}).(*expectation)
	return e, ok
}

// versionOf is the version of the Rencontre rencontreID, or else of the Direction tournamentID.
func (d *Service) versionOf(ctx context.Context, tournamentID, rencontreID int64) (string, error) {
	if rencontreID != 0 {
		return d.RencontreVersion(ctx, rencontreID)
	}
	return d.DirectionVersion(ctx, tournamentID)
}

// checkVersion compares the caller's version, if it stated one, with the current version of
// what the gesture names. Called in the gesture's guarded transaction (lockGesture), so no
// writer of any process moves it between the comparison and the commit.
func (d *Service) checkVersion(ctx context.Context, tournamentID, rencontreID int64) error {
	e, ok := expectedVersion(ctx)
	if !ok {
		return nil
	}
	cur, err := d.versionOf(ctx, tournamentID, rencontreID)
	if err != nil {
		return err
	}
	if cur != e.want {
		return ErrStale
	}
	return nil
}

// stableReads bounds the rereads of viewAfter and roomAfter while writers keep moving the state.
const stableReads = 5

// ViewAfter is the Direction as a gesture left it, read once the gesture's locks are released,
// and — for a caller that stated a version (ExpectVersion) — the version that view was built
// from, as ResultVersion: read before and after the view, again until both agree, so the view
// and the version handed back describe one state, whoever wrote meanwhile. When writers never
// let them agree, no version is handed back and the caller reads again.
func (d *Service) ViewAfter(ctx context.Context, tournamentID int64) (*DirectionView, error) {
	return afterGesture(ctx, func() (string, error) { return d.DirectionVersion(ctx, tournamentID) },
		func() (*DirectionView, error) { return d.GetDirection(ctx, tournamentID) })
}

func (d *Service) viewAfter(ctx context.Context, tournamentID int64) (*DirectionView, error) {
	return d.ViewAfter(ctx, tournamentID)
}

// roomAfter is ViewAfter for a Rencontre.
func (d *Service) roomAfter(ctx context.Context, id int64) (*RencontreView, error) {
	return afterGesture(ctx, func() (string, error) { return d.RencontreVersion(ctx, id) },
		func() (*RencontreView, error) { return d.GetRencontre(ctx, id) })
}

func afterGesture[V any](ctx context.Context, version func() (string, error), view func() (V, error)) (V, error) {
	e, versioned := expectedVersion(ctx)
	if !versioned {
		return view()
	}
	e.result = ""
	var out V
	for range stableReads {
		before, err := version()
		if err != nil {
			return out, err
		}
		if out, err = view(); err != nil {
			return out, err
		}
		after, err := version()
		if err != nil {
			return out, err
		}
		if before == after {
			e.result = before
			return out, nil
		}
	}
	return out, nil
}

// DirectionVersion is the token of a read about one Direction. A Direction that plays in a
// Rencontre reads its sister events too (busy tables, players seated elsewhere), so its token
// is the Rencontre's.
func (d *Service) DirectionVersion(ctx context.Context, tournamentID int64) (string, error) {
	rid, err := d.st.Rencontres().Of(ctx, d.scope, tournamentID)
	if err != nil {
		return "", err
	}
	if rid != 0 {
		return d.RencontreVersion(ctx, rid)
	}
	h := sha256.New()
	if err := d.writeDirectionToken(ctx, h, tournamentID, false); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// SlotsVersion is DirectionVersion plus the drafts started from the Direction's Slots, which
// the Slots view leads to.
func (d *Service) SlotsVersion(ctx context.Context, tournamentID int64) (string, error) {
	v, err := d.DirectionVersion(ctx, tournamentID)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	fmt.Fprintf(h, "%s|", v)
	if err := d.writeDraftsToken(ctx, h, tournamentID); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// RencontreVersion is the token of a read about a Rencontre: its row and every member's. A
// member that is not directed counts by its Tournament row alone, as the wall page shows it.
func (d *Service) RencontreVersion(ctx context.Context, id int64) (string, error) {
	r, err := d.st.Rencontres().Get(ctx, d.scope, id)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	if err := json.NewEncoder(h).Encode(r); err != nil {
		return "", err
	}
	for _, tid := range r.TournamentIDs {
		if err := d.writeDirectionToken(ctx, h, tid, true); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// ScopeVersion is the token of a read over the whole scope — the list of Directions, the
// directory, the list of Rencontres.
func (d *Service) ScopeVersion(ctx context.Context) (string, error) {
	h := sha256.New()
	recs, err := d.dirStore().ListDirections(ctx)
	if err != nil {
		return "", err
	}
	for _, r := range recs {
		if err := d.writeDirectionHead(ctx, h, r.TournamentID); err != nil {
			return "", err
		}
	}
	rs, err := d.st.Rencontres().List(ctx, d.scope)
	if err != nil {
		return "", err
	}
	if err := json.NewEncoder(h).Encode(rs); err != nil {
		return "", err
	}
	// Every Tournament row, not a count and a latest timestamp: a timestamp has the second
	// for unit, and two edits within one second would leave both unchanged.
	for t, err := range d.st.Tournaments().List(ctx, d.scope, storage.ListOpts{}) {
		if err != nil {
			return "", err
		}
		writeTournamentRow(h, t)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// writeDirectionToken feeds h with one member's sources: its Tournament row, then its
// Direction's. tolerateUndirected lets a Rencontre member with no Direction count by its row.
func (d *Service) writeDirectionToken(ctx context.Context, h hash.Hash, tournamentID int64, tolerateUndirected bool) error {
	t, err := d.st.Tournaments().Get(ctx, d.scope, tournamentID)
	if err != nil {
		return err
	}
	writeTournamentRow(h, t)
	err = d.writeDirectionHead(ctx, h, tournamentID)
	if tolerateUndirected && errors.Is(err, direction.ErrNoDirection) {
		fmt.Fprintf(h, "undirected %d|", tournamentID)
		return nil
	}
	return err
}

// writeDirectionHead feeds h with a Direction's own sources: its record, the length and head of
// its append-only journal, and its filled Slots.
func (d *Service) writeDirectionHead(ctx context.Context, h hash.Hash, tournamentID int64) error {
	ds := d.st.Directions()
	rec, err := ds.Get(ctx, d.scope, tournamentID)
	if err != nil {
		return err
	}
	n, last, err := ds.EventsHead(ctx, d.scope, tournamentID)
	if err != nil {
		return err
	}
	slots, err := ds.FilledSlots(ctx, d.scope, tournamentID)
	if err != nil {
		return err
	}
	fmt.Fprintf(h, "%d|%d|%s|%s|%s|%s|%d|%d|", rec.TournamentID, rec.FormatVersion, rec.EngineVersion,
		rec.State, rec.Config, rec.UpdatedAt.UTC().Format("2006-01-02T15:04:05.999999999"), n, last)
	return json.NewEncoder(h).Encode(slots)
}

// writeDraftsToken feeds h with the unsaved drafts started from a Slot of the Tournament — the
// ones slotDrafts finds.
func (d *Service) writeDraftsToken(ctx context.Context, h hash.Hash, tournamentID int64) error {
	for t, err := range d.st.Transcriptions().List(ctx, d.scope) {
		if err != nil {
			return err
		}
		if t.MatchID != 0 {
			continue
		}
		if slot, tid := draftSlot(t.Document); slot != "" && tid == tournamentID {
			fmt.Fprintf(h, "draft %d %s %s|", t.ID, t.UpdatedAt, slot)
		}
	}
	return nil
}

// writeTournamentRow feeds h with what a page shows of a Tournament.
func writeTournamentRow(h hash.Hash, t *domain.Tournament) {
	fmt.Fprintf(h, "t %d %q %q %q %s %d|", t.ID, t.Name, t.Date, t.Location, t.UpdatedAt, t.MatchCount)
}

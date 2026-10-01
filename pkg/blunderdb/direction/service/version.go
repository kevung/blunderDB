package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
)

// What a read depends on, condensed into a token that changes whenever one of its sources
// does — without replaying any Direction. A reader that already holds the answer for a token
// need not ask again (the serve daemon's ETag, ADR-0057 rule 4).
//
// The token covers the journals, the records, the Slots and the Rencontre rows a read is
// built from. It does not cover the wall clock: proposals, pages and the clock depend on the
// time of the reading too, and a caller that caches on this token bounds that drift itself.
// Nor does it cover a Tournament's own row (its name): renaming one is rare enough that the
// same bound is accepted.

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
	if err := d.writeDirectionToken(ctx, h, tournamentID); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// RencontreVersion is the token of a read about a Rencontre: its row and every member's.
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
		if err := d.writeDirectionToken(ctx, h, tid); err != nil {
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
		if err := d.writeDirectionToken(ctx, h, r.TournamentID); err != nil {
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
	return hex.EncodeToString(h.Sum(nil)), nil
}

// writeDirectionToken feeds h with what one Direction's reads are built from: its record, the
// length and head of its append-only journal, and its filled Slots.
func (d *Service) writeDirectionToken(ctx context.Context, h hash.Hash, tournamentID int64) error {
	ds := d.st.Directions()
	rec, err := ds.Get(ctx, d.scope, tournamentID)
	if err != nil {
		return err
	}
	evs, err := ds.LoadEvents(ctx, d.scope, tournamentID)
	if err != nil {
		return err
	}
	slots, err := ds.FilledSlots(ctx, d.scope, tournamentID)
	if err != nil {
		return err
	}
	last := -1
	if n := len(evs); n > 0 {
		last = evs[n-1].Seq
	}
	fmt.Fprintf(h, "%d|%d|%s|%s|%s|%s|%d|%d|", rec.TournamentID, rec.FormatVersion, rec.EngineVersion,
		rec.State, rec.Config, rec.UpdatedAt.UTC().Format("2006-01-02T15:04:05.999999999"), len(evs), last)
	return json.NewEncoder(h).Encode(slots)
}

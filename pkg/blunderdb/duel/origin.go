package duel

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// Origin is the origin of a Match played here, read back for anyone to check
// (ADR-0072 rules 8 and 10): what was recorded, plus the Fingerprint
// recomputed from the revealed seed — the one published when the Duel was
// created, so the rolls can be recomputed without trusting the Arbiter — and
// the Cadence decoded.
type Origin struct {
	storage.MatchOrigin
	// Fingerprint is the SHA-256 of the revealed seed; "" when the recorded
	// seed is not hexadecimal, which no Duel writes.
	Fingerprint string `json:"fingerprint"`
	// CadenceSettings is Cadence decoded, nil when the Duel had none.
	CadenceSettings *Cadence `json:"cadence_settings,omitempty"`
}

// OriginReader is the part of a storage the origin is read from.
type OriginReader interface {
	Duels() storage.DuelStore
}

// ReadOrigin returns the origin of a Match, or nil and no error when the
// Match was not played here — imported or transcribed.
func ReadOrigin(ctx context.Context, store OriginReader, scope string, matchID int64) (*Origin, error) {
	o, err := store.Duels().Origin(ctx, scope, matchID)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	out := &Origin{MatchOrigin: *o}
	if fp, ferr := Fingerprint(o.DiceSeed); ferr == nil {
		out.Fingerprint = fp
	}
	if o.Cadence != "" {
		var c Cadence
		if json.Unmarshal([]byte(o.Cadence), &c) == nil {
			out.CadenceSettings = &c
		}
	}
	return out, nil
}

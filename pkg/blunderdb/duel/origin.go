package duel

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// Origin is the origin of a Match played here, read back for anyone to check
// (ADR-0072 rules 8 and 10): what was recorded, the SHA-256 of the revealed
// seed — to compare with the fingerprint the Duel published at its creation,
// which is not kept — and the Cadence decoded.
type Origin struct {
	storage.MatchOrigin
	// Fingerprint is the SHA-256 of the revealed seed, computed here; "" when
	// the recorded seed is not hexadecimal, which no Duel writes.
	Fingerprint string `json:"fingerprint"`
	// CadenceSettings is Cadence decoded, nil when the Duel had none.
	CadenceSettings *Cadence `json:"cadence_settings,omitempty"`
	// LostOnTime: the match ended because the reserve of player OverTime ran
	// out under TimeLoseMatch — the test the Arbiter ends a Duel on. Such a
	// match is also StoppedEarly, though nobody stopped it.
	LostOnTime bool `json:"lost_on_time"`
}

// OriginReader is the part of a storage the origin is read from.
type OriginReader interface {
	Duels() storage.DuelStore
	Matches() storage.MatchStore
}

// ReadOrigin returns the origin of a Match, nil and no error when the Match
// exists but was not played here, and ErrNotFound when there is no such Match.
func ReadOrigin(ctx context.Context, store OriginReader, scope string, matchID int64) (*Origin, error) {
	o, err := store.Duels().Origin(ctx, scope, matchID)
	if errors.Is(err, storage.ErrNotFound) {
		if _, merr := store.Matches().Get(ctx, scope, matchID); merr != nil {
			return nil, merr
		}
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
	out.LostOnTime = out.CadenceSettings != nil && out.CadenceSettings.TimeOut == TimeLoseMatch && o.OverTime != 0
	return out, nil
}

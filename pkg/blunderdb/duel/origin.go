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
	// BindingDeclaredBot carries nothing and is never encoded. Wails walks no
	// embedded field when it collects the structs to generate, so without a
	// field naming storage.DeclaredBot directly the TypeScript binding of
	// MatchOrigin.DeclaredBots would name a class that does not exist.
	BindingDeclaredBot *storage.DeclaredBot `json:"-"`
	// Fingerprint is the SHA-256 of the revealed seed, computed here; "" when
	// the recorded seed is not hexadecimal, which no Duel writes.
	Fingerprint string `json:"fingerprint"`
	// CadenceSettings is Cadence decoded, nil when the Duel had none.
	CadenceSettings *Cadence `json:"cadence_settings,omitempty"`
	// LostOnTime: the match ended because the reserve of player OverTime ran
	// out under TimeLoseMatch — the test the Arbiter ends a Duel on. In
	// points, that player forfeits as the Duel ends, so the Match is won by
	// the other; a Match whose last game has no winner is also StoppedEarly.
	LostOnTime bool `json:"lost_on_time"`
	// Clock is the Match's clock replayed from its durations, nil without a
	// Cadence.
	Clock *MatchClock `json:"clock,omitempty"`
	// RollSeed is the seed the rolls were computed from when the Duel had a
	// combined seed — CombinedSeed over DiceSeed and Contributions, computed
	// here — and "" when they came from DiceSeed itself.
	RollSeed string `json:"roll_seed,omitempty"`
}

// OriginReader is the part of a storage the origin is read from.
type OriginReader interface {
	Duels() storage.DuelStore
	Matches() storage.MatchStore
	Stats() storage.StatsStore
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
	if len(o.Contributions) == 2 {
		if rs, rerr := CombinedSeed(o.DiceSeed, [2]string(o.Contributions)); rerr == nil {
			out.RollSeed = rs
		}
	}
	if o.Cadence != "" {
		var c Cadence
		if json.Unmarshal([]byte(o.Cadence), &c) == nil {
			out.CadenceSettings = &c
		}
	}
	out.LostOnTime = out.CadenceSettings != nil && out.CadenceSettings.TimeOut == TimeLoseMatch && o.OverTime != 0
	if c := out.CadenceSettings; c != nil {
		if clock, ok := replayClock(ctx, store, scope, matchID, *c); ok {
			out.Clock = &clock
		}
	}
	return out, nil
}

// replayClock replays the Match under its Cadence; false when the Match, its
// length or its Moves cannot be read, which leaves the origin without a clock.
func replayClock(ctx context.Context, store OriginReader, scope string, matchID int64, c Cadence) (MatchClock, bool) {
	m, err := store.Matches().Get(ctx, scope, matchID)
	if err != nil || m == nil {
		return MatchClock{}, false
	}
	turns, err := store.Stats().MatchTurns(ctx, scope, matchID)
	if err != nil {
		return MatchClock{}, false
	}
	return c.Replay(int(m.MatchLength), turns), true
}

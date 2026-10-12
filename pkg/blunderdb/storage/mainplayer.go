package storage

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// The Main player is the person a library belongs to: the one whose matches
// it holds, wanted at seat 1 of each so that every match reads from their
// side. Designating them swaps, in one transaction, every match where they
// sit at seat 2 (MatchStore.SwapPlayers, which keeps every analysis, error,
// comment, video mark, transcription and tournament), and every later import
// places them at seat 1. They are recognised by their aliases: a name merged
// into theirs is theirs. Forgetting them swaps nothing back.

// MainPlayerPlan is what designating a Main player does: the matches it
// swaps, and those where the player sits at seat 2 but whose swap is refused
// (a Transcription draft open on it, a match played here), which it leaves.
type MainPlayerPlan struct {
	// Name is the canonical name the plan was drawn for.
	Name    string  `json:"name"`
	Swap    []int64 `json:"swap"`
	Refused []int64 `json:"refused"`
}

// MainPlayerAtSeat2 reports whether the match whose seats are p1 and p2 has
// the Main player main at seat 2 and somebody else at seat 1, the names read
// through aliases. A match between two spellings of the Main player is left
// as it is: swapping it would put nobody else first.
func MainPlayerAtSeat2(aliases AliasMap, main, p1, p2 string) bool {
	main = aliases.Canonical(main)
	if main == "" {
		return false
	}
	return aliases.Canonical(p2) == main && aliases.Canonical(p1) != main
}

// MainPlayer returns the scope's Main player under their canonical name, ""
// when none is designated: an alias recorded after the designation still
// names the same person.
func MainPlayer(ctx context.Context, st Stores, scope string) (string, error) {
	name, err := st.LibrarySettings().MainPlayer(ctx, scope)
	if err != nil || name == "" {
		return "", err
	}
	aliases, err := st.Aliases().List(ctx, scope, AliasPlayer)
	if err != nil {
		return "", err
	}
	return NewAliasMap(aliases).Canonical(name), nil
}

// PlanMainPlayer draws the plan of designating name, writing nothing.
func PlanMainPlayer(ctx context.Context, st Stores, scope, name string) (MainPlayerPlan, error) {
	plan := MainPlayerPlan{Swap: []int64{}, Refused: []int64{}}
	if strings.TrimSpace(name) == "" {
		return plan, fmt.Errorf("main player: blank name: %w", ErrInvalid)
	}
	list, err := st.Aliases().List(ctx, scope, AliasPlayer)
	if err != nil {
		return plan, err
	}
	aliases := NewAliasMap(list)
	plan.Name = aliases.Canonical(name)

	// Collected before any refusal is asked: a cursor still open on the
	// connection would hold it across the next query.
	var candidates []int64
	for m, err := range st.Matches().List(ctx, scope, MatchListOpts{PlayerName: plan.Name}) {
		if err != nil {
			return plan, err
		}
		if MainPlayerAtSeat2(aliases, plan.Name, m.Player1Name, m.Player2Name) {
			candidates = append(candidates, m.ID)
		}
	}
	for _, id := range candidates {
		err := st.Matches().SwapRefusal(ctx, scope, id)
		switch {
		case err == nil:
			plan.Swap = append(plan.Swap, id)
		case errors.Is(err, ErrSwapDraftOpen), errors.Is(err, ErrSwapPlayedHere):
			plan.Refused = append(plan.Refused, id)
		default:
			return plan, err
		}
	}
	return plan, nil
}

// DesignateMainPlayer records name as the scope's Main player and swaps
// every match of the plan, all in one transaction: on any failure no match
// is swapped and the setting is unchanged. It returns the plan it applied.
func DesignateMainPlayer(ctx context.Context, st Storage, scope, name string) (MainPlayerPlan, error) {
	tx, err := st.BeginTx(ctx)
	if err != nil {
		return MainPlayerPlan{}, err
	}
	defer func() { _ = tx.Rollback() }()
	plan, err := PlanMainPlayer(ctx, tx, scope, name)
	if err != nil {
		return plan, err
	}
	if err := tx.LibrarySettings().SetMainPlayer(ctx, scope, plan.Name); err != nil {
		return plan, err
	}
	for _, id := range plan.Swap {
		if err := tx.Matches().SwapPlayers(ctx, scope, id); err != nil {
			return plan, fmt.Errorf("main player %q: %w", plan.Name, err)
		}
	}
	// The swap only invalidates the per-seat tallies; they are rebuilt in the
	// same transaction, as an import does, so no reader waits for them.
	if len(plan.Swap) > 0 {
		if err := tx.Stats().RefreshMatchStats(ctx, scope, plan.Swap); err != nil {
			return plan, err
		}
	}
	if err := tx.Commit(); err != nil {
		return plan, err
	}
	return plan, nil
}

// ClearMainPlayer forgets the scope's Main player. No match is swapped back.
func ClearMainPlayer(ctx context.Context, st Stores, scope string) error {
	return st.LibrarySettings().SetMainPlayer(ctx, scope, "")
}

// PlaceMainPlayerFirst swaps the match when the scope's Main player sits at
// its seat 2, and reports whether it did. A match whose swap is refused (a
// draft open on it, played here) is left as it is. The caller refreshes the
// match's stats.
func PlaceMainPlayerFirst(ctx context.Context, st Stores, scope string, matchID int64) (bool, error) {
	main, err := st.LibrarySettings().MainPlayer(ctx, scope)
	if err != nil || main == "" {
		return false, err
	}
	list, err := st.Aliases().List(ctx, scope, AliasPlayer)
	if err != nil {
		return false, err
	}
	m, err := st.Matches().Get(ctx, scope, matchID)
	if err != nil {
		return false, err
	}
	if !MainPlayerAtSeat2(NewAliasMap(list), main, m.Player1Name, m.Player2Name) {
		return false, nil
	}
	switch err := st.Matches().SwapPlayers(ctx, scope, matchID); {
	case err == nil:
		return true, nil
	case errors.Is(err, ErrSwapDraftOpen), errors.Is(err, ErrSwapPlayedHere):
		return false, nil
	default:
		return false, err
	}
}

// PlayerMatchIDs returns the ids of the matches name plays, at either seat
// and under any of their spellings: what a player's match count opens.
func PlayerMatchIDs(ctx context.Context, st Stores, scope, name string) ([]int64, error) {
	ids := []int64{}
	for m, err := range st.Matches().List(ctx, scope, MatchListOpts{PlayerName: name}) {
		if err != nil {
			return nil, err
		}
		ids = append(ids, m.ID)
	}
	return ids, nil
}

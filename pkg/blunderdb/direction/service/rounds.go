package service

import (
	"context"
	"strings"
	"time"

	tournoi "github.com/kevung/nicomaque"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
)

// The gestures a round-based phase adds to the director's console: the byes of a manual phase,
// the half-byes a player announces in a Swiss by rounds, and the two corrections of an entry
// list made before anyone has played — a replacement and an exchange of places. The engine
// judges each one; this layer only names the current phase and writes the event.

// byeAct is what PlaceBye and CancelBye do.
type byeAct int

const (
	// byePlace exempts the player for the round.
	byePlace byeAct = iota
	// byeCancel withdraws an exemption placed by mistake.
	byeCancel
)

// PlaceBye exempts the Participant id from round of the current manual phase — a point, as a
// bye of a Swiss by rounds. round 0 is the phase's latest round.
func (d *Service) PlaceBye(ctx context.Context, tournamentID int64, id string, round int) (*DirectionView, error) {
	return d.byeGesture(ctx, tournamentID, id, round, byePlace)
}

// CancelBye withdraws the exemption PlaceBye gave id in round of the current manual phase.
func (d *Service) CancelBye(ctx context.Context, tournamentID int64, id string, round int) (*DirectionView, error) {
	return d.byeGesture(ctx, tournamentID, id, round, byeCancel)
}

func (d *Service) byeGesture(ctx context.Context, tournamentID int64, id string, round int, g byeAct) (*DirectionView, error) {
	err := d.applyToCurrentPhase(ctx, tournamentID, func(st *tournoi.State, ph *tournoi.PhaseState, now time.Time) (tournoi.Event, error) {
		if ph.Cfg.Kind != tournoi.KindManual {
			return tournoi.Event{}, direction.Refusef("direction: only a manual phase takes a bye from the director")
		}
		if round == 0 {
			round = max(ph.Round, 1)
		}
		if g == byeCancel {
			return st.ManualByeCancelledEvent(round, tournoi.PlayerID(id), now), nil
		}
		return st.ManualByeEvent(round, tournoi.PlayerID(id), now), nil
	})
	if err != nil {
		return nil, err
	}
	return d.viewAfter(ctx, tournamentID)
}

// GiveHalfBye records that the Participant id will not play round of the current phase, a Swiss
// by rounds: they are not paired in it and take half a point. The engine refuses a round
// already open; how many a player may take is the organiser's rule, not checked here.
func (d *Service) GiveHalfBye(ctx context.Context, tournamentID int64, id string, round int) (*DirectionView, error) {
	return d.halfBye(ctx, tournamentID, id, round, false)
}

// CancelHalfBye withdraws a half-bye of round, while that round is not open.
func (d *Service) CancelHalfBye(ctx context.Context, tournamentID int64, id string, round int) (*DirectionView, error) {
	return d.halfBye(ctx, tournamentID, id, round, true)
}

func (d *Service) halfBye(ctx context.Context, tournamentID int64, id string, round int, cancel bool) (*DirectionView, error) {
	if round <= 0 {
		return nil, direction.Refusef("direction: a half-bye names its round")
	}
	err := d.applyToCurrentPhase(ctx, tournamentID, func(st *tournoi.State, _ *tournoi.PhaseState, now time.Time) (tournoi.Event, error) {
		if cancel {
			return tournoi.HalfByeCancelledEvent(tournoi.PlayerID(id), st.Current, round, now), nil
		}
		return tournoi.HalfByeEvent(tournoi.PlayerID(id), st.Current, round, now), nil
	})
	if err != nil {
		return nil, err
	}
	return d.viewAfter(ctx, tournamentID)
}

// ReplaceParticipant puts a new player in the place of id, who has not played a match yet: the
// newcomer takes their seed, their slot in a draw and their pairing. newID, when given, is the
// identifier the newcomer keeps; without one it is derived from the name.
func (d *Service) ReplaceParticipant(ctx context.Context, tournamentID int64, id, newID, name, club string, rating float64) (*DirectionView, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, direction.Refusef("direction: an entry needs a name")
	}
	err := d.applyToDirection(ctx, tournamentID, func(st *tournoi.State, now time.Time) (tournoi.Event, error) {
		if _, ok := st.Players[tournoi.PlayerID(id)]; !ok {
			return tournoi.Event{}, direction.Refusef("direction: no entry %q", id)
		}
		pid, err := entryID(st, newID, name)
		if err != nil {
			return tournoi.Event{}, err
		}
		p := tournoi.Player{ID: pid, Name: name, Club: club, Rating: rating}
		return tournoi.PlayerReplacedEvent(tournoi.PlayerID(id), p, now), nil
	})
	if err != nil {
		return nil, err
	}
	return d.viewAfter(ctx, tournamentID)
}

// SwapParticipants exchanges the places of a and b, neither of whom has played yet — two
// names entered in each other's line of a draw.
func (d *Service) SwapParticipants(ctx context.Context, tournamentID int64, a, b string) (*DirectionView, error) {
	err := d.applyToDirection(ctx, tournamentID, func(_ *tournoi.State, now time.Time) (tournoi.Event, error) {
		return tournoi.PlayersSwappedEvent(tournoi.PlayerID(a), tournoi.PlayerID(b), now), nil
	})
	if err != nil {
		return nil, err
	}
	return d.viewAfter(ctx, tournamentID)
}

// applyToDirection writes the event build makes from the Direction's state, under its lock.
func (d *Service) applyToDirection(ctx context.Context, tournamentID int64, build func(*tournoi.State, time.Time) (tournoi.Event, error)) (err error) {
	d, release, err := d.lockDirection(ctx, tournamentID)
	if err != nil {
		return err
	}
	defer release(&err)
	dir, err := direction.Open(ctx, d.dirStore(), tournamentID)
	if err != nil {
		return err
	}
	st := dir.State()
	if st == nil {
		return direction.ErrNoDirection
	}
	ev, err := build(st, time.Now())
	if err != nil {
		return err
	}
	return dir.Apply(ctx, ev)
}

// applyToCurrentPhase is applyToDirection for a gesture about the phase being played.
func (d *Service) applyToCurrentPhase(ctx context.Context, tournamentID int64, build func(*tournoi.State, *tournoi.PhaseState, time.Time) (tournoi.Event, error)) error {
	return d.applyToDirection(ctx, tournamentID, func(st *tournoi.State, now time.Time) (tournoi.Event, error) {
		if st.Current < 0 || st.Current >= len(st.Phases) || st.Phases[st.Current] == nil {
			return tournoi.Event{}, direction.Refusef("direction: no phase is being played")
		}
		return build(st, st.Phases[st.Current], now)
	})
}

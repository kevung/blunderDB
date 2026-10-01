package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	tournoi "github.com/PileOfCells/backgammon-tournoi"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// A doubles event enters pairs (ADR-0056 §4). The engine knows ONE Participant per pair, named
// "A / B" — a pair decides together, and the Match carries "A / B" as the Player of each side.
// The two persons behind it live in direction_pair_member: the Directory lists them one per
// line, and the entry form shows them. The label is derived from the members, and the entry
// rating is their mean unless the director corrects it.

// PairMember is one of the two persons of a pair.
type PairMember = storage.PairMember

// PairLabel is the Participant's name for a pair: "A / B", in the order the pair was entered.
func PairLabel(members []PairMember) string {
	names := make([]string, len(members))
	for i, m := range members {
		names[i] = m.Name
	}
	return strings.Join(names, " / ")
}

// PairRating is the entry rating of a pair: the mean of the ratings known. A member with no
// rating (0) does not pull the mean to zero.
func PairRating(members []PairMember) float64 {
	sum, n := 0.0, 0
	for _, m := range members {
		if m.Rating > 0 {
			sum += m.Rating
			n++
		}
	}
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}

// pairClub is one club when both play for it, both otherwise.
func pairClub(members []PairMember) string {
	if members[0].Club == members[1].Club {
		return members[0].Club
	}
	if members[0].Club == "" || members[1].Club == "" {
		return members[0].Club + members[1].Club
	}
	return members[0].Club + " / " + members[1].Club
}

func parsePair(membersJSON string) ([]PairMember, error) {
	var members []PairMember
	if err := json.Unmarshal([]byte(membersJSON), &members); err != nil {
		return nil, fmt.Errorf("direction: pair: %w", err)
	}
	if len(members) != 2 {
		return nil, fmt.Errorf("direction: a pair is two persons, got %d", len(members))
	}
	for i := range members {
		members[i].Name = strings.TrimSpace(members[i].Name)
		members[i].Club = strings.TrimSpace(members[i].Club)
		if members[i].Name == "" {
			return nil, fmt.Errorf("direction: each person of a pair needs a name")
		}
	}
	return members, nil
}

// AddPair enters a pair: one Participant "A / B" for the engine, two persons for the Directory,
// written in one transaction. rating 0 means "the mean of the two"; any other value is the
// director's correction.
func (d *Service) AddPair(ctx context.Context, tournamentID int64, membersJSON string, rating float64) (_ *DirectionView, err error) {
	d, release, err := d.lockDirection(ctx, tournamentID)
	if err != nil {
		return nil, err
	}
	defer release(&err)
	members, err := parsePair(membersJSON)
	if err != nil {
		return nil, err
	}
	err = d.directionTx(ctx, func(ctx context.Context, tx storage.Tx, store direction.Store) error {
		dir, err := direction.Open(ctx, store, tournamentID)
		if err != nil {
			return err
		}
		st := dir.State()
		if st == nil {
			return direction.ErrNoDirection
		}
		taken := map[string]bool{}
		for id := range st.Players {
			taken[string(id)] = true
		}
		label := PairLabel(members)
		if rating <= 0 {
			rating = PairRating(members)
		}
		p := tournoi.Player{ID: tournoi.PlayerID(participantID(label, taken)), Name: label, Club: pairClub(members), Rating: rating}
		if err := dir.Apply(ctx, tournoi.PlayerAddedEvent(p, time.Now())); err != nil {
			return err
		}
		return writePairMembers(ctx, tx, d.scope, tournamentID, string(p.ID), members)
	})
	if err != nil {
		return nil, err
	}
	return d.GetDirection(ctx, tournamentID)
}

// UpdatePair corrects a pair — a member's name, club or rating, or the entry rating — without
// changing its identifier, which is what its Slots and Matches point at.
func (d *Service) UpdatePair(ctx context.Context, tournamentID int64, id, membersJSON string, rating float64) (_ *DirectionView, err error) {
	d, release, err := d.lockDirection(ctx, tournamentID)
	if err != nil {
		return nil, err
	}
	defer release(&err)
	members, err := parsePair(membersJSON)
	if err != nil {
		return nil, err
	}
	err = d.directionTx(ctx, func(ctx context.Context, tx storage.Tx, store direction.Store) error {
		dir, err := direction.Open(ctx, store, tournamentID)
		if err != nil {
			return err
		}
		st := dir.State()
		if st == nil {
			return direction.ErrNoDirection
		}
		cur, ok := st.Players[tournoi.PlayerID(id)]
		if !ok {
			return fmt.Errorf("direction: no entry %q", id)
		}
		if rating <= 0 {
			rating = PairRating(members)
		}
		p := *cur
		p.Name, p.Club, p.Rating = PairLabel(members), pairClub(members), rating
		if err := dir.Apply(ctx, tournoi.PlayerUpdatedEvent(p, time.Now())); err != nil {
			return err
		}
		return writePairMembers(ctx, tx, d.scope, tournamentID, id, members)
	})
	if err != nil {
		return nil, err
	}
	return d.GetDirection(ctx, tournamentID)
}

func writePairMembers(ctx context.Context, tx storage.Tx, scope string, tournamentID int64, id string, members []PairMember) error {
	if err := tx.Directions().SetPair(ctx, scope, tournamentID, id, members); err != nil {
		return fmt.Errorf("direction: pair %s: %w", id, err)
	}
	return nil
}

// Pairs gives the persons of every pair of a Direction, by Participant id. Empty for a singles
// event.
func (d *Service) Pairs(ctx context.Context, tournamentID int64) (map[string][]PairMember, error) {
	return d.st.Directions().Pairs(ctx, d.scope, tournamentID)
}

// personsOf is a Direction's entrants as persons: a pair counts as its two members, so the
// Directory never holds an "A / B" line.
func (d *Service) personsOf(ctx context.Context, tournamentID int64) ([]tournoi.Player, error) {
	players, err := d.entrantsOf(ctx, tournamentID)
	if err != nil {
		return nil, err
	}
	pairs, err := d.Pairs(ctx, tournamentID)
	if err != nil {
		return nil, err
	}
	if len(pairs) == 0 {
		return players, nil
	}
	out := make([]tournoi.Player, 0, len(players)+len(pairs))
	for _, p := range players {
		members, ok := pairs[string(p.ID)]
		if !ok {
			out = append(out, p)
			continue
		}
		for _, m := range members {
			out = append(out, tournoi.Player{Name: m.Name, Club: m.Club, Rating: m.Rating})
		}
	}
	return out, nil
}

// directionTx runs fn in one transaction with a Direction store bound to it: an event and the
// rows that go with it are written together or not at all.
func (d *Service) directionTx(ctx context.Context, fn func(context.Context, storage.Tx, direction.Store) error) error {
	return d.inTx(ctx, func(tx storage.Tx, store direction.Store) error { return fn(ctx, tx, store) })
}

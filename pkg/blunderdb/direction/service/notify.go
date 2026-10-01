package service

import (
	"context"
	"errors"
	"slices"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
	"github.com/kevung/blunderdb/pkg/blunderdb/events"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// SetPublisher installs the publisher told of every gesture this Memory's services commit —
// the serve daemon's event bus. nil (the default) tells no one and reads nothing more; neither
// does a publisher that wants nothing of the scope. The desktop installs none.
func (m *Memory) SetPublisher(p events.Publisher) {
	m.directionMu.Lock()
	defer m.directionMu.Unlock()
	m.publisher = p
}

// listener is the publisher when someone listens to the service's scope, nil otherwise: the
// versions an event carries cost reads that no one would hear.
func (d *Service) listener() events.Publisher {
	d.directionMu.RLock()
	p := d.publisher
	d.directionMu.RUnlock()
	if p == nil || !p.Wants(d.scope) {
		return nil
	}
	return p
}

// roomMembers is the members of room before a gesture, so that a gesture that removes one —
// a detach, a trashed room — still reaches the subscribers of the Directions it moved. Read
// only when someone listens.
func (d *Service) roomMembers(ctx context.Context, st storage.Stores, room int64) []int64 {
	if room == 0 || d.listener() == nil {
		return nil
	}
	r, err := st.Rencontres().Get(ctx, d.scope, room)
	if err != nil {
		return nil
	}
	return slices.Clone(r.TournamentIDs)
}

// publishGesture tells the publisher what a committed gesture moved: the room it locked, with
// its members before and after, and the Direction it named when that one plays in no room
// after the gesture. The versions are read after the commit, outside the gesture's locks: a
// later writer may already have moved them, and then publishes its own event.
func (d *Service) publishGesture(ctx context.Context, tournamentID, room int64, before []int64) {
	p := d.listener()
	if p == nil {
		return
	}
	ctx = context.WithoutCancel(ctx)
	var after []int64
	if room != 0 {
		ev := events.Event{Scope: d.scope, Kind: events.KindRencontre, RencontreID: room}
		r, err := d.st.Rencontres().Get(ctx, d.scope, room)
		switch {
		case err == nil:
			after = r.TournamentIDs
			ev.Version, _ = d.RencontreVersion(ctx, room)
		case errors.Is(err, storage.ErrNotFound):
			ev.Removed = true
		}
		ev.TournamentIDs = union(before, after)
		p.Publish(ev)
	}
	if tournamentID != 0 && !slices.Contains(after, tournamentID) {
		d.publishDirection(ctx, p, tournamentID)
	}
}

// publishDirection publishes the version a Direction has now, or its removal.
func (d *Service) publishDirection(ctx context.Context, p events.Publisher, tournamentID int64) {
	ev := events.Event{Scope: d.scope, Kind: events.KindDirection, TournamentID: tournamentID}
	v, err := d.DirectionVersion(ctx, tournamentID)
	switch {
	case err == nil:
		ev.Version = v
	case errors.Is(err, direction.ErrNoDirection), errors.Is(err, storage.ErrNotFound):
		ev.Removed = true
	}
	p.Publish(ev)
}

// publishRoom publishes a Rencontre written outside a gesture's locks (its creation). Inside a
// gesture it waits for the gesture's own commit, which publishes what it locked.
func (d *Service) publishRoom(ctx context.Context, id int64) {
	if d.g != nil || d.listener() == nil {
		return
	}
	d.publishGesture(ctx, 0, id, nil)
}

func union(a, b []int64) []int64 {
	out := slices.Clone(a)
	for _, id := range b {
		if !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	slices.Sort(out)
	return out
}

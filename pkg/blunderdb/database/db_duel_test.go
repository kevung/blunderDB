package database

import (
	"context"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/duel"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func TestDuelFacade_CreateConflictStop(t *testing.T) {
	db := newTestDB(t)

	sides := [2]duel.SideSpec{{Kind: duel.SideExternal, Name: "Kévin"}, {Kind: duel.SideExternal, Name: "Alice"}}
	st, err := db.CreateDuel(duel.Settings{MatchLength: 3, Sides: sides})
	if err != nil {
		t.Fatalf("CreateDuel: %v", err)
	}
	if st.State.ID == 0 || st.State.Awaiting == nil {
		t.Fatalf("a new Duel awaits a decision: %+v", st.State)
	}
	if st.Sheet.Document.Header.MatchLength != 3 {
		t.Fatalf("the sheet carries the session: %+v", st.Sheet.Document.Header)
	}

	list, err := db.ListDuels()
	if err != nil || len(list) != 1 || !list[0].Open {
		t.Fatalf("ListDuels = %+v, %v; want the one Duel, open", list, err)
	}

	// A stale revision is answered with the Duel as it stands, not an error.
	stale, err := db.PlayDuel(st.State.ID, st.State.Revision+7, duel.Play{Side: st.State.Awaiting.Side, Kind: duel.PlayResign, Level: 1})
	if err != nil || !stale.Conflict || stale.State.Revision != st.State.Revision {
		t.Fatalf("stale PlayDuel = %+v, %v; want a flagged conflict at revision %d", stale, err, st.State.Revision)
	}

	stopped, err := db.StopDuel(st.State.ID, st.State.Revision, false)
	if err != nil || stopped.State.Ended == nil || !stopped.State.Ended.Discarded {
		t.Fatalf("StopDuel discard = %+v, %v; want nothing written", stopped, err)
	}
	if list, _ := db.ListDuels(); len(list) != 0 {
		t.Fatalf("an ended Duel leaves the list: %+v", list)
	}
}

// Closing the library puts the open Duel in suspense: its clocks stop with it.
func TestDuelFacade_CloseSuspends(t *testing.T) {
	db := newTestDB(t)
	sides := [2]duel.SideSpec{{Kind: duel.SideExternal, Name: "A"}, {Kind: duel.SideExternal, Name: "B"}}
	if _, err := db.CreateDuel(duel.Settings{MatchLength: 1, Sides: sides}); err != nil {
		t.Fatalf("CreateDuel: %v", err)
	}
	db.forgetDuels()
	list, err := db.ListDuels()
	if err != nil || len(list) != 1 || list[0].Open {
		t.Fatalf("after forgetDuels: %+v, %v; want the Duel in suspense", list, err)
	}
}

// The form's « score seul » Start: the opening position at a chosen away score, no roll on it.
func TestDuelFacade_StartAtScore(t *testing.T) {
	db := newTestDB(t)
	start := domain.InitializePosition()
	start.Dice = [2]int{}
	start.Score = [2]int{2, 5}
	sides := [2]duel.SideSpec{{Kind: duel.SideExternal, Name: "A"}, {Kind: duel.SideExternal, Name: "B"}}
	st, err := db.CreateDuel(duel.Settings{MatchLength: 5, Start: &start, Sides: sides})
	if err != nil {
		t.Fatalf("CreateDuel at a score: %v", err)
	}
	if st.State.Score != [2]int{3, 0} || st.State.Awaiting == nil {
		t.Fatalf("score %v, awaiting %+v; want 3-0 and a decision", st.State.Score, st.State.Awaiting)
	}
}

// The origin of a Match played here is read back with its revealed seed, which
// gives the fingerprint published at creation; an imported Match has none.
func TestDuelFacade_MatchOrigin(t *testing.T) {
	db := newTestDB(t)

	sides := [2]duel.SideSpec{{Kind: duel.SideExternal, Name: "Kévin"}, {Kind: duel.SideExternal, Name: "Alice"}}
	st, err := db.CreateDuel(duel.Settings{MatchLength: 1, Sides: sides})
	if err != nil {
		t.Fatalf("CreateDuel: %v", err)
	}
	published := st.State.Fingerprint
	for range 3 {
		a := st.State.Awaiting
		p := duel.Play{Side: a.Side, Kind: duel.PlayRoll}
		if a.Kind == duel.DecideAnswer {
			p.Kind = duel.PlayTake
		} else if a.Kind != duel.DecideCube {
			p = duel.Play{Side: a.Side, Kind: duel.PlayMove, Steps: domain.LegalMoves(&a.Position)[0].Steps}
		}
		if st, err = db.PlayDuel(st.State.ID, st.State.Revision, p); err != nil || st.Conflict {
			t.Fatalf("PlayDuel: %+v, %v", st, err)
		}
	}
	if _, err := db.StopDuel(st.State.ID, st.State.Revision, true); !errors.Is(err, storage.ErrInvalid) {
		t.Fatalf("keeping an unfinished match in points: %v, want ErrInvalid", err)
	}
	ended, err := db.ForfeitDuel(st.State.ID, st.State.Revision, 1)
	if err != nil || ended.State.Ended == nil || ended.State.Ended.MatchID == 0 {
		t.Fatalf("forfeiting the match = %+v, %v", ended, err)
	}

	o, err := db.GetMatchOrigin(ended.State.Ended.MatchID)
	if err != nil || o == nil {
		t.Fatalf("GetMatchOrigin = %+v, %v", o, err)
	}
	if fp, _ := duel.Fingerprint(o.DiceSeed); fp != published || o.Fingerprint != published {
		t.Errorf("seed read back gives %q, origin says %q, published %q", fp, o.Fingerprint, published)
	}
	if o.StoppedEarly {
		t.Errorf("a match forfeited is won, not stopped early: %+v", o.MatchOrigin)
	}

	imported, err := db.store.Matches().Save(context.Background(), "", &domain.Match{Player1Name: "A", Player2Name: "B", MatchLength: 3})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if o, err := db.GetMatchOrigin(imported); err != nil || o != nil {
		t.Errorf("a match not played here = %+v, %v; want no origin, no error", o, err)
	}
	if _, err := db.GetMatchOrigin(imported + 100); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("no such match: %v, want ErrNotFound", err)
	}
}

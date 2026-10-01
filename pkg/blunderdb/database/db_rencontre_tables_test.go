package database

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
)

// roomOfTwo puts two directed events of n players in one Rencontre of the given tables.
func roomOfTwo(t *testing.T, d *Database, n, tables int) (int64, int64) {
	t.Helper()
	_, a := directedTournament(t, d, n)
	_, b := directedTournament(t, d, n)
	r, err := d.CreateRencontre("Festival", "", "", tables)
	if err != nil {
		t.Fatal(err)
	}
	for _, tID := range []int64{a, b} {
		if _, err := d.AttachToRencontre(tID, r.ID); err != nil {
			t.Fatal(err)
		}
	}
	return a, b
}

// within fails the test instead of hanging it when fn does not return in time.
func within(t *testing.T, d time.Duration, what string, fn func() error) {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- fn() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("%s: %v", what, err)
		}
	case <-time.After(d):
		t.Fatalf("%s did not return within %v", what, d)
	}
}

// The GUI runs on an in-memory database, whose pool holds a single connection: a gesture that
// read the room outside its own transaction would wait for that connection forever.
func TestStartMatchManuallyInARoomOnAnInMemoryDatabase(t *testing.T) {
	d := NewDatabase()
	if err := d.SetupDatabase(":memory:"); err != nil {
		t.Fatal(err)
	}
	// Closed by hand: a gesture left waiting on the connection would hold the cleanup too.
	a, b := roomOfTwo(t, d, 8, 8)
	within(t, 5*time.Second, "StartMatchManually in A", func() error {
		_, err := d.StartMatchManually(a, "a", "b", 0, 0)
		return err
	})
	within(t, 5*time.Second, "StartMatchManually in B", func() error {
		v, err := d.StartMatchManually(b, "c", "d", 0, 0)
		if err == nil && (len(v.Running) != 1 || v.Running[0].Table != 2) {
			return fmt.Errorf("B's match runs at %+v, want table 2", v.Running)
		}
		return err
	})
	_ = d.Close()
}

// A table number typed by hand is refused when a sister event of the room plays on it: the
// room never holds two matches on one table.
func TestStartMatchManuallyRefusesASistersTable(t *testing.T) {
	d := newTestDB(t)
	a, b := roomOfTwo(t, d, 8, 8)
	if _, err := d.StartMatchManually(a, "a", "b", 0, 3); err != nil {
		t.Fatal(err)
	}
	if _, err := d.StartMatchManually(b, "c", "d", 0, 3); err == nil {
		t.Fatal("B started a match on table 3, where A plays")
	}
	if _, err := d.StartMatchManually(b, "c", "d", 0, 4); err != nil {
		t.Fatalf("a free table of the room is refused: %v", err)
	}
}

// Sister events launching at once never land on one table: the free table is chosen and taken
// under the room's lock.
func TestSistersStartingAtOnceNeverShareATable(t *testing.T) {
	d := newTestDB(t)
	a, b := roomOfTwo(t, d, 16, 16)
	var wg sync.WaitGroup
	errs := make(chan error, 16)
	for _, tID := range []int64{a, b} {
		for i := range 8 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				p := string(rune('a' + 2*i))
				q := string(rune('a' + 2*i + 1))
				if _, err := d.StartMatchManually(tID, p, q, 0, 0); err != nil {
					errs <- err
				}
			}()
		}
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("a simultaneous start failed: %v", err)
	}
	seen := map[int]int64{}
	for _, tID := range []int64{a, b} {
		v, err := d.GetDirection(tID)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range v.Running {
			if m.Table <= 0 {
				t.Errorf("event %d: %s runs at no table", tID, m.ID)
				continue
			}
			if other, ok := seen[m.Table]; ok {
				t.Errorf("table %d holds a match of %d and one of %d", m.Table, other, tID)
			}
			seen[m.Table] = tID
		}
	}
}

// A proposal shown before a sister took its table is stale: confirming it is refused rather
// than stacking two matches on one table.
func TestConfirmingAStaleProposalOnASistersTableIsRefused(t *testing.T) {
	d := newTestDB(t)
	a, b := roomOfTwo(t, d, 16, 8)
	firstStart := func(tID int64) tournoi.Action {
		t.Helper()
		v, err := d.GetDirection(tID)
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range v.Proposals {
			if p.Kind == tournoi.ActStartMatch && p.Table > 0 {
				return p
			}
		}
		t.Fatalf("event %d proposes no match on a table", tID)
		return tournoi.Action{}
	}
	pa, pb := firstStart(a), firstStart(b)
	if pa.Table != pb.Table {
		t.Fatalf("both events should see table %d free, B proposes %d", pa.Table, pb.Table)
	}
	confirmJSON := func(tID int64, p tournoi.Action) error {
		blob, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		_, err = d.ConfirmProposal(tID, string(blob))
		return err
	}
	if err := confirmJSON(a, pa); err != nil {
		t.Fatal(err)
	}
	if err := confirmJSON(b, pb); err == nil {
		t.Fatalf("B confirmed a match on table %d, where A now plays", pb.Table)
	}
}

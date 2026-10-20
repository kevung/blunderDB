package database

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"
)

func openTemp(t *testing.T, name string) (*Database, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	d := NewDatabase()
	if err := d.SetupDatabase(path); err != nil {
		t.Fatalf("setup %s: %v", name, err)
	}
	t.Cleanup(func() { _ = d.Close() })
	return d, path
}

func TestExchange_SwapsTheOpenLibrary(t *testing.T) {
	bound, pathA := openTemp(t, "a.db")
	parked, pathB := openTemp(t, "b.db")
	pos := initialPosition()
	if _, err := bound.SavePosition(&pos); err != nil {
		t.Fatal(err)
	}
	genBound, genParked := bound.currentGeneration(), parked.currentGeneration()
	hooked := 0
	bound.SetBeforeSwitch(func() { hooked++ })

	if err := Exchange(bound, parked); err != nil {
		t.Fatal(err)
	}
	if OpenPath(bound) != pathB || OpenPath(parked) != pathA {
		t.Fatalf("paths after exchange: bound %q parked %q", OpenPath(bound), OpenPath(parked))
	}
	if n, _ := bound.CountPositions(); n != 0 {
		t.Errorf("bound handle reads %d positions, want B's 0", n)
	}
	if n, _ := parked.CountPositions(); n != 1 {
		t.Errorf("parked handle reads %d positions, want A's 1", n)
	}
	if hooked != 1 {
		t.Errorf("before-switch hook ran %d times, want 1", hooked)
	}
	if bound.currentGeneration() == genBound || parked.currentGeneration() == genParked {
		t.Error("an exchange must move both generations so no job writes across it")
	}

	// And back: nothing was opened or closed on the way.
	if err := Exchange(bound, parked); err != nil {
		t.Fatal(err)
	}
	if n, _ := bound.CountPositions(); n != 1 || OpenPath(bound) != pathA {
		t.Errorf("after the return trip the bound handle reads %d positions at %q", n, OpenPath(bound))
	}
}

func TestExchange_RefusesWhileAnImportHoldsEitherSide(t *testing.T) {
	bound, _ := openTemp(t, "a.db")
	parked, _ := openTemp(t, "b.db")
	_, done := parked.beginCancellableImport()
	err := Exchange(bound, parked)
	done()
	var busy *BusyError
	if !errors.As(err, &busy) {
		t.Fatalf("exchange during an import: %v, want *BusyError", err)
	}
	if err := Exchange(bound, parked); err != nil {
		t.Fatalf("exchange once idle: %v", err)
	}
}

func TestOpenPath_FollowsOpenAndClose(t *testing.T) {
	d, path := openTemp(t, "a.db")
	if OpenPath(d) != path {
		t.Fatalf("OpenPath = %q, want %q", OpenPath(d), path)
	}
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
	if OpenPath(d) != "" {
		t.Errorf("OpenPath after Close = %q", OpenPath(d))
	}
	if err := d.OpenDatabase(path); err != nil {
		t.Fatal(err)
	}
	if OpenPath(d) != path {
		t.Errorf("OpenPath after reopen = %q", OpenPath(d))
	}
}

// Every field of Database is either part of the open library, and so moved by
// Exchange, or part of the handle. A new field fails here until it is placed:
// left out of Exchange by mistake, it would follow the wrong file.
func TestExchange_EveryFieldIsPlaced(t *testing.T) {
	library := map[string]bool{
		"db": true, "store": true, "lock": true, "readOnly": true, "path": true,
		"transcriptSvc": true, "transcriptOn": true, "duelSvc": true, "duelOn": true, "duelBoard": true,
		"positionsSinceStats": true, "pendingPhaseBackfill": true, "pendingAnkiCardKinds": true,
		// Moved by bumping, not swapping: both sides change.
		"generation": true, "storeEpoch": true,
	}
	handle := map[string]bool{
		"mu": true, "cancelMu": true, "switchMu": true, "beforeSwitch": true,
		"transcriptMu": true, "duelMu": true,
		// Refused while set (idleForExchange) or only alive during one import.
		"importCancel": true, "job": true, "searchCancels": true, "searchSeq": true,
		"importBatchID": true, "importBatchCounts": true, "importJournal": true,
		"migrationProgress": true, "convertBatchSize": true, "afterActionColumnDrop": true,
		"skipDuplicates": true, "commentAuthor": true, "directionMem": true,
	}
	typ := reflect.TypeOf(Database{})
	for i := 0; i < typ.NumField(); i++ {
		name := typ.Field(i).Name
		if !library[name] && !handle[name] {
			t.Errorf("Database.%s is neither library nor handle state: place it (and swap it in Exchange if it follows the file)", name)
		}
	}
}

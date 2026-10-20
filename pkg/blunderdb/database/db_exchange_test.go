package database

import (
	"context"
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
		"generation": true, "storeEpoch": true, "openPath": true,
	}
	handle := map[string]bool{
		"mu": true, "cancelMu": true, "switchMu": true, "beforeSwitch": true, "openGuard": true,
		"transcriptMu": true, "duelMu": true,
		// Refused while set (idleForExchange) or only alive during one import.
		"importCancel": true, "job": true, "exchanging": true, "searchCancels": true, "searchSeq": true,
		"importBatchID": true, "importBatchCounts": true, "importJournal": true,
		"migrationProgress": true, "convertBatchSize": true, "afterActionColumnDrop": true,
		"skipDuplicates": true, "commentAuthor": true, "directionMem": true,
		// A running recount is cancelled by the generation bump.
		"recount": true,
	}
	typ := reflect.TypeOf(Database{})
	for i := 0; i < typ.NumField(); i++ {
		name := typ.Field(i).Name
		if !library[name] && !handle[name] {
			t.Errorf("Database.%s is neither library nor handle state: place it (and swap it in Exchange if it follows the file)", name)
		}
	}
}

// Between the idle check and the swap, neither a job nor an import may start
// on either handle: it would begin on one library and write into the next.
// The before-switch hook runs inside that window.
func TestExchange_NoJobStartsDuringTheSwap(t *testing.T) {
	bound, pathA := openTemp(t, "a.db")
	parked, _ := openTemp(t, "b.db")
	var jobErr, parkedJobErr error
	importCancelled := false
	bound.SetBeforeSwitch(func() {
		_, jobErr = bound.acquireJob(context.Background(), JobRollout, nil)
		_, parkedJobErr = parked.acquireJob(context.Background(), JobRollout, nil)
		ctx, done := bound.beginCancellableImport()
		importCancelled = ctx.Err() != nil
		done()
	})
	if err := Exchange(bound, parked); err != nil {
		t.Fatal(err)
	}
	var busy *BusyError
	if !errors.As(jobErr, &busy) || busy.Job != JobExchange {
		t.Errorf("job on the bound handle during the swap: %v, want busy %q", jobErr, JobExchange)
	}
	if !errors.As(parkedJobErr, &busy) || busy.Job != JobExchange {
		t.Errorf("job on the parked handle during the swap: %v, want busy %q", parkedJobErr, JobExchange)
	}
	if !importCancelled {
		t.Error("an import begun during the swap got a live context")
	}
	bound.SetBeforeSwitch(nil)
	release, err := bound.acquireJob(context.Background(), JobRollout, nil)
	if err != nil {
		t.Fatalf("job after the swap: %v", err)
	}
	release()
	if !parked.IsOpenDatabase(pathA) {
		t.Error("the parked handle does not report the file it now carries")
	}
}

func TestOpenGuard_RefusesBeforeTouchingTheOpenFile(t *testing.T) {
	d, path := openTemp(t, "a.db")
	refused := errors.New("held elsewhere")
	other := filepath.Join(t.TempDir(), "b.db")
	SetOpenGuard(d, func(p string) error {
		if p == other {
			return refused
		}
		return nil
	})
	if err := d.OpenDatabase(other); !errors.Is(err, refused) {
		t.Fatalf("OpenDatabase = %v, want the guard's refusal", err)
	}
	if err := d.SetupDatabase(other); !errors.Is(err, refused) {
		t.Fatalf("SetupDatabase = %v, want the guard's refusal", err)
	}
	if OpenPath(d) != path {
		t.Errorf("open file after a refusal = %q, want %q", OpenPath(d), path)
	}
}

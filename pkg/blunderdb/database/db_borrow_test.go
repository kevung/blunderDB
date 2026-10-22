package database

import (
	"context"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// A borrowed handle stays on its library across an exchange: what it writes
// lands in the library it was lent, never in the one the bound handle carries
// next.
func TestBorrow_KeepsItsLibraryAcrossAnExchange(t *testing.T) {
	t.Parallel()
	bound, pathA := openTemp(t, "a.db")
	parked, _ := openTemp(t, "b.db")
	lent, err := Borrow(bound)
	if err != nil {
		t.Fatal(err)
	}
	if err := Exchange(bound, parked); err != nil {
		t.Fatalf("exchange with a borrowed handle out: %v", err)
	}
	pos := initialPosition()
	if _, err := lent.SavePosition(&pos); err != nil {
		t.Fatal(err)
	}
	if n, _ := bound.CountPositions(); n != 0 {
		t.Errorf("the bound handle (now B) reads %d positions, want 0", n)
	}
	if n, _ := parked.CountPositions(); n != 1 || OpenPath(parked) != pathA {
		t.Errorf("A, now parked, reads %d positions at %q, want 1 at %q", n, OpenPath(parked), pathA)
	}
	// Letting it go closes nothing.
	if err := lent.Close(); err != nil {
		t.Fatal(err)
	}
	if n, err := parked.CountPositions(); err != nil || n != 1 {
		t.Errorf("A after the borrower let go: %d positions, %v", n, err)
	}
}

// A job on a borrowed handle holds the library's slot wherever the library
// goes, and does not stop the exchange; a job on the bound handle still does.
func TestBorrow_SharesTheJobSlotAndLetsTheExchangeThrough(t *testing.T) {
	t.Parallel()
	bound, _ := openTemp(t, "a.db")
	parked, _ := openTemp(t, "b.db")
	lent, err := Borrow(bound)
	if err != nil {
		t.Fatal(err)
	}
	release, err := lent.acquireJob(context.Background(), JobGammonNet, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	var busy *BusyError
	if _, err := bound.acquireJob(context.Background(), JobMaintenance, nil); !errors.As(err, &busy) || busy.Job != JobGammonNet {
		t.Fatalf("a second job on the lender: %v, want busy: gammonnet", err)
	}
	if err := Exchange(bound, parked); err != nil {
		t.Fatalf("exchange while a borrowed handle runs a job: %v", err)
	}
	// The slot went with A: B is free, A (now parked) is not.
	rel, err := bound.acquireJob(context.Background(), JobMaintenance, nil)
	if err != nil {
		t.Fatalf("a job on B: %v", err)
	}
	rel()
	if _, err := parked.acquireJob(context.Background(), JobMaintenance, nil); !errors.As(err, &busy) {
		t.Fatalf("a job on A, parked, while its borrower runs one: %v, want busy", err)
	}

	own, err := bound.acquireJob(context.Background(), JobRollout, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer own()
	if err := Exchange(bound, parked); !errors.As(err, &busy) {
		t.Fatalf("exchange while the bound handle runs a job: %v, want busy", err)
	}
}

// The import batch opened on the bound handle moves to the borrower, which
// finishes it; the lender is left with none.
func TestHandOverImportBatch_MovesTheOpenBatch(t *testing.T) {
	t.Parallel()
	bound, _ := openTemp(t, "a.db")
	id, err := bound.BeginImportBatch("x", "mixed")
	if err != nil {
		t.Fatal(err)
	}
	lent, err := Borrow(bound)
	if err != nil {
		t.Fatal(err)
	}
	HandOverImportBatch(bound, lent)
	if bound.importBatchID != 0 || lent.importBatchID != id {
		t.Fatalf("batch after hand-over: lender %d, borrower %d, want 0 and %d", bound.importBatchID, lent.importBatchID, id)
	}
	if err := lent.FinishImportBatch(id, domain.ImportReport{}); err != nil {
		t.Fatal(err)
	}
}

// A vacuum replaces the file under every handle of the library: refused while
// a borrowed handle runs a job or an import, and no import starts during it.
func TestVacuum_RefusedWhileABorrowedHandleWrites(t *testing.T) {
	t.Parallel()
	bound, _ := openTemp(t, "a.db")
	lent, err := Borrow(bound)
	if err != nil {
		t.Fatal(err)
	}
	var busy *BusyError
	release, err := lent.acquireJob(context.Background(), JobGammonNet, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bound.Vacuum(); !errors.As(err, &busy) || busy.Job != JobGammonNet {
		t.Fatalf("vacuum during a borrowed batch: %v, want busy: gammonnet", err)
	}
	release()
	_, done := lent.beginCancellableImport()
	if _, err := bound.Vacuum(); !errors.As(err, &busy) || busy.Job != "import" {
		t.Fatalf("vacuum during a borrowed import: %v, want busy: import", err)
	}
	done()

	rel, err := bound.acquireVacuum()
	if err != nil {
		t.Fatal(err)
	}
	ctx, end := lent.beginCancellableImport()
	if ctx.Err() == nil {
		t.Error("an import started during a vacuum")
	}
	end()
	rel()
	if _, err := bound.Vacuum(); err != nil {
		t.Fatalf("vacuum once idle: %v", err)
	}
}

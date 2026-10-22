package database

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// copyFixture is a source library holding one position with a comment, and
// an empty destination.
func copyFixture(t *testing.T) (src, dst *Database, id int64, pos Position) {
	t.Helper()
	src, dst = newTestDB(t), newTestDB(t)
	pos = initialPosition()
	id, err := src.SavePosition(&pos)
	if err != nil {
		t.Fatalf("SavePosition: %v", err)
	}
	if err := src.AddComment(id, "carried along"); err != nil {
		t.Fatalf("AddComment: %v", err)
	}
	return src, dst, id, pos
}

func heldIn(t *testing.T, d *Database, p Position) (int64, bool) {
	t.Helper()
	id, found, err := d.store.Positions().Exists(context.Background(), "", storage.ZobristOf(&p))
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	return id, found
}

func TestCopyPosition_CreatesThenTakesBack(t *testing.T) {
	src, dst, id, pos := copyFixture(t)
	r, err := CopyPosition(context.Background(), src, dst, id)
	if err != nil {
		t.Fatalf("CopyPosition: %v", err)
	}
	held, found := heldIn(t, dst, pos)
	if !r.Created || !found || r.ID != held {
		t.Fatalf("receipt = created %v id %d; destination holds %d (%v)", r.Created, r.ID, held, found)
	}
	if err := TakeBackPositionCopy(dst, r); err != nil {
		t.Fatalf("TakeBackPositionCopy: %v", err)
	}
	if _, found := heldIn(t, dst, pos); found {
		t.Error("the copied position outlived its take-back")
	}
	if _, found := heldIn(t, src, pos); !found {
		t.Error("the take-back reached the source")
	}
}

func TestCopyPosition_AlreadyHeldIsNotTakenBack(t *testing.T) {
	src, dst, id, pos := copyFixture(t)
	before := pos
	if _, err := dst.SavePosition(&before); err != nil {
		t.Fatalf("SavePosition: %v", err)
	}
	r, err := CopyPosition(context.Background(), src, dst, id)
	if err != nil {
		t.Fatalf("CopyPosition: %v", err)
	}
	if r.Created {
		t.Fatal("a copy onto a position the destination held reported it created")
	}
	if err := TakeBackPositionCopy(dst, r); err == nil {
		t.Error("taking back a copy that created nothing succeeded")
	}
	if _, found := heldIn(t, dst, pos); !found {
		t.Error("the destination lost the position it held before the copy")
	}
}

func TestTakeBackPositionCopy_RefusesWhatChangedSince(t *testing.T) {
	src, dst, id, pos := copyFixture(t)
	r, err := CopyPosition(context.Background(), src, dst, id)
	if err != nil || !r.Created {
		t.Fatalf("CopyPosition: created %v, %v", r.Created, err)
	}
	if err := dst.AddComment(r.ID, "written after the copy"); err != nil {
		t.Fatalf("AddComment: %v", err)
	}
	if err := TakeBackPositionCopy(dst, r); !errors.Is(err, ErrPositionChanged) {
		t.Fatalf("take-back after a new comment = %v, want ErrPositionChanged", err)
	}
	if _, found := heldIn(t, dst, pos); !found {
		t.Error("a refused take-back removed the position")
	}
	if err := TakeBackPositionCopy(src, r); !errors.Is(err, ErrPositionChanged) {
		t.Errorf("take-back on another library = %v, want ErrPositionChanged", err)
	}
}

// A borrowed handle writes through the destination's storage without its
// lock: when it saves the same position while the copy runs, exactly one of
// the two inserted it, and the receipt says so.
func TestCopyPosition_ConcurrentDuplicateHasOneCreator(t *testing.T) {
	for round := range 8 {
		src, dst, id, pos := copyFixture(t)
		b, err := Borrow(dst)
		if err != nil {
			t.Fatalf("Borrow: %v", err)
		}
		var (
			wg          sync.WaitGroup
			r           PositionCopyReceipt
			copyErr     error
			borrowedNew bool
			borrowedErr error
		)
		wg.Add(2)
		go func() {
			defer wg.Done()
			r, copyErr = CopyPosition(context.Background(), src, dst, id)
		}()
		go func() {
			defer wg.Done()
			p := pos
			_, borrowedNew, borrowedErr = b.store.Positions().SaveCreated(context.Background(), "", &p)
		}()
		wg.Wait()
		if copyErr != nil || borrowedErr != nil {
			t.Fatalf("round %d: copy %v, borrowed save %v", round, copyErr, borrowedErr)
		}
		if r.Created == borrowedNew {
			t.Fatalf("round %d: copy created %v, borrowed save created %v; want exactly one", round, r.Created, borrowedNew)
		}
		if borrowedNew {
			if err := TakeBackPositionCopy(dst, r); err == nil {
				t.Fatalf("round %d: took back a position the borrowed handle created", round)
			}
			if _, found := heldIn(t, dst, pos); !found {
				t.Fatalf("round %d: the borrowed handle's position was removed", round)
			}
		}
	}
}

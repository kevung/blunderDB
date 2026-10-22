package database

import (
	"context"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
)

// A transfer reads its source through the pool a vacuum closes: the vacuum
// is refused while the source is held, and a transfer is refused while a
// vacuum of its source runs.
func TestTransferTo_HoldsTheSourceAgainstAVacuum(t *testing.T) {
	t.Parallel()
	src, dst := newTestDB(t), newTestDB(t)
	release, err := src.holdForRead()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := src.acquireVacuum(); err == nil {
		t.Fatal("a vacuum of a source being read was allowed")
	}
	release()

	stop, err := src.acquireVacuum()
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	_, err = src.TransferTo(context.Background(), dst, ingest.WholeLibraryOptions())
	var busy *BusyError
	if !errors.As(err, &busy) || busy.Job != JobVacuum {
		t.Fatalf("transfer from a library being vacuumed: %v, want BusyError(vacuum)", err)
	}
}

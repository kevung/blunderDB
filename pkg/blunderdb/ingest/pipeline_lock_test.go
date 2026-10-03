package ingest

import (
	"context"
	"errors"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type probeStore struct {
	reads    *atomic.Int64
	atBegin  atomic.Int64
	lockedAt atomic.Int64
	lock     *atomic.Int64
}

func (p *probeStore) BeginTx(context.Context) (storage.Tx, error) {
	p.atBegin.Store(p.reads.Load())
	return nil, errors.New("probe: stop here")
}

// The writer takes the lock and opens the transaction of a group only when
// every file of the group has been read.
func TestImportFiles_GroupIsReadBeforeLockAndTransaction(t *testing.T) {
	src := filepath.Join("..", "..", "..", "testdata", "match_with_comment.xg")
	paths := []string{src, src, src, src}
	var reads, locks atomic.Int64
	store := &probeStore{reads: &reads, lock: &locks}
	_, err := ImportFiles(context.Background(), store, paths, PipelineOptions{
		Workers:    2,
		FilesPerTx: 4,
		OnRead:     func(int64) { reads.Add(1) },
		Lock: func() func() {
			store.lockedAt.Store(reads.Load())
			return func() {}
		},
	})
	if err == nil {
		t.Fatal("the probe store should have stopped the import")
	}
	if got := store.lockedAt.Load(); got != 4 {
		t.Errorf("lock taken with %d of 4 files read", got)
	}
	if got := store.atBegin.Load(); got != 4 {
		t.Errorf("transaction opened with %d of 4 files read", got)
	}
}

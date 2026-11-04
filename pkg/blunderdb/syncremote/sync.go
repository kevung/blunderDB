package syncremote

import (
	"context"
	"errors"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/changeset"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// DefaultPage is the number of records a page carries.
const DefaultPage = 500

// Local is the open library.
type Local interface {
	ChangesetExport(ctx context.Context, after storage.Cursor, limit int) (changeset.Set, error)
	ChangesetImport(ctx context.Context, set changeset.Set) (changeset.Report, error)
}

// Remote is the server.
type Remote interface {
	Export(ctx context.Context, after storage.Cursor, limit int) (changeset.Set, error)
	Import(ctx context.Context, records []changeset.Record) (changeset.Report, error)
}

// Cursors are the two positions a run resumes from, kept per server and library.
type Cursors struct {
	// Sent is in the local journal: what the server already has.
	Sent storage.Cursor `json:"sent"`
	// Received is in the server's journal: what this library already merged.
	Received storage.Cursor `json:"received"`
}

// Totals are what one direction did, summed over its pages.
type Totals struct {
	Records   int `json:"records"`
	Added     int `json:"added"`
	Modified  int `json:"modified"`
	Removed   int `json:"removed"`
	Older     int `json:"older"`
	Conflicts int `json:"conflicts"`
	Missing   int `json:"missing"`
}

func (t *Totals) add(n int, r changeset.Report) {
	t.Records += n
	t.Added += r.Added
	t.Modified += r.Modified
	t.Removed += r.Removed
	t.Older += r.Older
	t.Conflicts += len(r.Conflicts)
	t.Missing += len(r.Missing)
}

// Result is the outcome of a run: Sent is what the server reported on the
// records it received, Received what the local merge reported.
type Result struct {
	Sent     Totals `json:"sent"`
	Received Totals `json:"received"`
}

// Options tune a run.
type Options struct {
	// Page is the records per page, DefaultPage when zero.
	Page int
	// Save is called with the cursors after every settled page, so a run cut
	// short resumes there. Its error stops the run.
	Save func(Cursors) error
}

// Run sends then receives. It returns the cursors reached even on an error, so
// the caller keeps the progress made; the error is the first thing that failed.
func Run(ctx context.Context, local Local, remote Remote, cur Cursors, opt Options) (Cursors, Result, error) {
	if opt.Page <= 0 {
		opt.Page = DefaultPage
	}
	var res Result
	save := func() error {
		if opt.Save == nil {
			return nil
		}
		return opt.Save(cur)
	}

	restarted := false
	for {
		if err := ctx.Err(); err != nil {
			return cur, res, err
		}
		set, err := local.ChangesetExport(ctx, cur.Sent, opt.Page)
		if errors.Is(err, storage.ErrCursorStale) && !restarted && !cur.Sent.IsStart() {
			cur.Sent, restarted = storage.Cursor{}, true
			continue
		}
		if err != nil {
			return cur, res, fmt.Errorf("reading this library's changes: %w", err)
		}
		if len(set.Records) > 0 {
			rep, err := remote.Import(ctx, set.Records)
			if err != nil {
				return cur, res, fmt.Errorf("sending: %w", err)
			}
			res.Sent.add(len(set.Records), rep)
		}
		if set.More && set.Cursor == cur.Sent {
			return cur, res, fmt.Errorf("reading this library's changes: the cursor does not advance")
		}
		cur.Sent = set.Cursor
		if err := save(); err != nil {
			return cur, res, err
		}
		if !set.More {
			break
		}
	}

	restarted = false
	for {
		if err := ctx.Err(); err != nil {
			return cur, res, err
		}
		set, err := remote.Export(ctx, cur.Received, opt.Page)
		if errors.Is(err, storage.ErrCursorStale) && !restarted && !cur.Received.IsStart() {
			cur.Received, restarted = storage.Cursor{}, true
			continue
		}
		if err != nil {
			return cur, res, fmt.Errorf("receiving: %w", err)
		}
		if len(set.Records) > 0 {
			rep, err := local.ChangesetImport(ctx, set)
			if err != nil {
				return cur, res, fmt.Errorf("merging what was received: %w", err)
			}
			res.Received.add(len(set.Records), rep)
		}
		if set.More && set.Cursor == cur.Received {
			return cur, res, fmt.Errorf("receiving: the cursor does not advance")
		}
		cur.Received = set.Cursor
		if err := save(); err != nil {
			return cur, res, err
		}
		if !set.More {
			break
		}
	}
	return cur, res, nil
}

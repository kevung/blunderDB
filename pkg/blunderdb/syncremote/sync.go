package syncremote

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/changeset"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// DefaultPage is the number of records a page carries; no import carries more.
const DefaultPage = 500

// maxRetryAfter bounds the wait a run accepts on a rate-limited answer: longer,
// the run stops and says so rather than hold the window.
const maxRetryAfter = 2 * time.Minute

// maxRetries is how many times one call waits out a rate limit.
const maxRetries = 3

// Local is the open library.
type Local interface {
	Export(ctx context.Context, after storage.Cursor, limit int) (changeset.Set, error)
	Import(ctx context.Context, set changeset.Set) (changeset.Report, error)
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
	// Unsupported records are of a family the receiving side does not carry.
	Unsupported int `json:"unsupported"`
	// Unsynchronizable records are refused by the server for their size alone.
	Unsynchronizable int `json:"unsynchronizable"`
}

func (t *Totals) add(n int, r changeset.Report) {
	t.Records += n
	t.Added += r.Added
	t.Modified += r.Modified
	t.Removed += r.Removed
	t.Older += r.Older
	t.Conflicts += len(r.Conflicts)
	t.Missing += len(r.Missing)
	t.Unsupported += r.Unsupported
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
	// Sleep waits out a Retry-After, time-based when nil.
	Sleep func(ctx context.Context, d time.Duration) error
}

func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// retrying replays a call the server rate-limited, after the wait it asked for.
// Every other error, a refusal on the allowance included, is returned at once.
func retrying[T any](ctx context.Context, opt Options, call func() (T, error)) (T, error) {
	for n := 0; ; n++ {
		v, err := call()
		after, ok := RetryAfter(err)
		if err == nil || !ok || n >= maxRetries || after > maxRetryAfter {
			return v, err
		}
		if err := opt.Sleep(ctx, after); err != nil {
			return v, err
		}
	}
}

// send imports records, halving the batch while the server finds it too large.
// A record that is too large alone cannot be synchronised: it is counted, and
// the run goes on with the next.
func send(ctx context.Context, remote Remote, opt Options, recs []changeset.Record, res *Result) error {
	rep, err := retrying(ctx, opt, func() (changeset.Report, error) { return remote.Import(ctx, recs) })
	switch {
	case err == nil:
		res.Sent.add(len(recs), rep)
		return nil
	case !errors.Is(err, ErrTooLarge):
		return err
	case len(recs) == 1:
		res.Sent.Records++
		res.Sent.Unsynchronizable++
		return nil
	}
	mid := len(recs) / 2
	if err := send(ctx, remote, opt, recs[:mid], res); err != nil {
		return err
	}
	return send(ctx, remote, opt, recs[mid:], res)
}

// Run sends then receives. It returns the cursors reached even on an error, so
// the caller keeps the progress made; the error is the first thing that failed.
func Run(ctx context.Context, local Local, remote Remote, cur Cursors, opt Options) (Cursors, Result, error) {
	if opt.Page <= 0 || opt.Page > DefaultPage {
		opt.Page = DefaultPage
	}
	if opt.Sleep == nil {
		opt.Sleep = sleep
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
		set, err := local.Export(ctx, cur.Sent, opt.Page)
		if errors.Is(err, storage.ErrCursorStale) && !restarted && !cur.Sent.IsStart() {
			cur.Sent, restarted = storage.Cursor{}, true
			continue
		}
		if err != nil {
			return cur, res, fmt.Errorf("reading this library's changes: %w", err)
		}
		if len(set.Records) > 0 {
			if err := send(ctx, remote, opt, set.Records, &res); err != nil {
				return cur, res, fmt.Errorf("sending: %w", err)
			}
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
		set, err := retrying(ctx, opt, func() (changeset.Set, error) { return remote.Export(ctx, cur.Received, opt.Page) })
		if errors.Is(err, storage.ErrCursorStale) && !restarted && !cur.Received.IsStart() {
			cur.Received, restarted = storage.Cursor{}, true
			continue
		}
		if err != nil {
			return cur, res, fmt.Errorf("receiving: %w", err)
		}
		if len(set.Records) > 0 {
			rep, err := local.Import(ctx, set)
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

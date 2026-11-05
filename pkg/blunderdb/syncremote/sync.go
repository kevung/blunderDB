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

// Page is a page of the remote journal. Skipped names the records the server
// declined to carry: its cursor moves past them.
type Page struct {
	changeset.Set
	Skipped []Skipped `json:"skipped,omitempty"`
}

// Skipped is a record the server's page left out, as it says it.
type Skipped struct {
	Family string `json:"family"`
	UUID   string `json:"uuid"`
	Size   int64  `json:"size,omitempty"`
}

// Remote is the server.
type Remote interface {
	Export(ctx context.Context, after storage.Cursor, limit int) (Page, error)
	Import(ctx context.Context, records []changeset.Record) (changeset.Report, error)
}

// Cursors are the two positions a run resumes from, kept per server and library.
type Cursors struct {
	// Sent is in the local journal: what the server already has.
	Sent storage.Cursor `json:"sent"`
	// Received is in the server's journal: what this library already merged.
	Received storage.Cursor `json:"received"`
	// Skipped are the records set aside, kept until they go through: the
	// cursors have moved past them, so they would not come back otherwise.
	Skipped []Skip `json:"skipped,omitempty"`
}

// The reasons a record is set aside.
const (
	// ReasonTooLarge: the server refused it for its size, in its own words
	// (code payload_too_large or record_too_large), or declined to send it.
	ReasonTooLarge = "too_large"
	// ReasonInterrupted: it alone could not be sent before the connection broke
	// or timed out; nothing refused it.
	ReasonInterrupted = "interrupted"
)

// Skip is a record set aside. From and To are the cursors of the page it was
// in, on the side it belongs to, to find it again.
type Skip struct {
	Direction string         `json:"direction"` // DirectionSent or DirectionReceived
	Family    string         `json:"family"`
	UUID      string         `json:"uuid"`
	Size      int64          `json:"size,omitempty"`
	Reason    string         `json:"reason"`
	From      storage.Cursor `json:"from"`
	To        storage.Cursor `json:"to"`
}

// The directions of a Skip.
const (
	DirectionSent     = "sent"
	DirectionReceived = "received"
)

func (c *Cursors) setAside(s Skip) {
	for i, o := range c.Skipped {
		if o.Direction == s.Direction && o.Family == s.Family && o.UUID == s.UUID {
			c.Skipped[i] = s
			return
		}
	}
	c.Skipped = append(c.Skipped, s)
}

func (c *Cursors) release(direction string, keep func(Skip) bool) {
	out := c.Skipped[:0:0]
	for _, o := range c.Skipped {
		if o.Direction != direction || keep(o) {
			out = append(out, o)
		}
	}
	c.Skipped = out
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
	// Unsynchronizable records are set aside in this run, whatever the side.
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
	// Skipped is every record still set aside when the run ends.
	Skipped []Skip `json:"skipped,omitempty"`
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

// send imports records, halving the batch while the server finds it too large
// or the connection breaks under it. A record alone that the server refuses by
// its size, or that cannot get through, is handed to aside, which keeps it for
// the next run; any other failure stops the run with the cursor where it was.
func send(ctx context.Context, remote Remote, opt Options, recs []changeset.Record, res *Result, aside func(rec changeset.Record, reason string)) error {
	rep, err := retrying(ctx, opt, func() (changeset.Report, error) { return remote.Import(ctx, recs) })
	reason := ""
	switch {
	case err == nil:
		res.Sent.add(len(recs), rep)
		return nil
	case errors.Is(err, ErrTooLarge):
		reason = ReasonTooLarge
	case errors.Is(err, ErrInterrupted):
		reason = ReasonInterrupted
	default:
		return err
	}
	if len(recs) == 1 {
		res.Sent.Records++
		res.Sent.Unsynchronizable++
		aside(recs[0], reason)
		return nil
	}
	mid := len(recs) / 2
	if err := send(ctx, remote, opt, recs[:mid], res, aside); err != nil {
		return err
	}
	return send(ctx, remote, opt, recs[mid:], res, aside)
}

// skippedAfter lists the records of a side set aside, to be tried again.
func (c *Cursors) skippedAfter(direction string) []Skip {
	var out []Skip
	for _, o := range c.Skipped {
		if o.Direction == direction {
			out = append(out, o)
		}
	}
	return out
}

// retrySent sends again the records set aside on the way out, found by
// exporting the page they were in. One no longer there (changed since, so
// journaled again) is dropped. One still refused stays.
func retrySent(ctx context.Context, local Local, remote Remote, cur *Cursors, opt Options, res *Result, save func() error) error {
	for _, sk := range cur.skippedAfter(DirectionSent) {
		var found []changeset.Record
		after := sk.From
		for {
			set, err := local.Export(ctx, after, opt.Page)
			if err != nil {
				if errors.Is(err, storage.ErrCursorStale) {
					break
				}
				return fmt.Errorf("reading this library's changes: %w", err)
			}
			for _, r := range set.Records {
				if r.Family == sk.Family && r.UUID == sk.UUID {
					found = append(found, r)
				}
			}
			if !set.More || set.Cursor == sk.To || set.Cursor == after {
				break
			}
			after = set.Cursor
		}
		cur.release(DirectionSent, func(o Skip) bool { return o.Family != sk.Family || o.UUID != sk.UUID })
		if len(found) == 0 {
			continue
		}
		again := func(rec changeset.Record, reason string) {
			s := sk
			s.Reason, s.Size = reason, int64(len(rec.Payload))
			cur.setAside(s)
		}
		if err := send(ctx, remote, opt, found[len(found)-1:], res, again); err != nil {
			cur.setAside(sk)
			return fmt.Errorf("sending: %w", err)
		}
		if err := save(); err != nil {
			return err
		}
	}
	return nil
}

// receive merges one page of the remote journal and sets aside what it skipped.
func receive(ctx context.Context, local Local, cur *Cursors, from storage.Cursor, page Page, res *Result) error {
	if len(page.Records) > 0 {
		rep, err := local.Import(ctx, page.Set)
		if err != nil {
			return fmt.Errorf("merging what was received: %w", err)
		}
		res.Received.add(len(page.Records), rep)
	}
	for _, k := range page.Skipped {
		res.Received.Records++
		res.Received.Unsynchronizable++
		cur.setAside(Skip{Direction: DirectionReceived, Family: k.Family, UUID: k.UUID, Size: k.Size, Reason: ReasonTooLarge, From: from, To: page.Cursor})
	}
	return nil
}

// retryReceived asks again for the pages whose records the server skipped.
// Merging is idempotent, so a record that came before the skipped one costs
// only its transfer.
func retryReceived(ctx context.Context, local Local, remote Remote, cur *Cursors, opt Options, res *Result, save func() error) error {
	for _, sk := range cur.skippedAfter(DirectionReceived) {
		cur.release(DirectionReceived, func(o Skip) bool { return o.Family != sk.Family || o.UUID != sk.UUID })
		after := sk.From
		for {
			page, err := retrying(ctx, opt, func() (Page, error) { return remote.Export(ctx, after, opt.Page) })
			if errors.Is(err, storage.ErrCursorStale) {
				break // the next full pass brings it back
			}
			if err != nil {
				cur.setAside(sk)
				return fmt.Errorf("receiving: %w", err)
			}
			if err := receive(ctx, local, cur, after, page, res); err != nil {
				cur.setAside(sk)
				return err
			}
			if !page.More || page.Cursor == sk.To || page.Cursor == after {
				break
			}
			after = page.Cursor
		}
		if err := save(); err != nil {
			return err
		}
	}
	return nil
}

// Run sends then receives. It returns the cursors reached even on an error, so
// the caller keeps the progress made; the error is the first thing that failed.
func Run(ctx context.Context, local Local, remote Remote, cur Cursors, opt Options) (Cursors, Result, error) {
	cur, res, err := run(ctx, local, remote, cur, opt)
	res.Skipped = cur.Skipped
	return cur, res, err
}

func run(ctx context.Context, local Local, remote Remote, cur Cursors, opt Options) (Cursors, Result, error) {
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

	if err := retrySent(ctx, local, remote, &cur, opt, &res, save); err != nil {
		return cur, res, err
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
			aside := func(rec changeset.Record, reason string) {
				cur.setAside(Skip{Direction: DirectionSent, Family: rec.Family, UUID: rec.UUID, Size: int64(len(rec.Payload)), Reason: reason, From: cur.Sent, To: set.Cursor})
			}
			if err := send(ctx, remote, opt, set.Records, &res, aside); err != nil {
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

	if err := retryReceived(ctx, local, remote, &cur, opt, &res, save); err != nil {
		return cur, res, err
	}

	restarted = false
	for {
		if err := ctx.Err(); err != nil {
			return cur, res, err
		}
		set, err := retrying(ctx, opt, func() (Page, error) { return remote.Export(ctx, cur.Received, opt.Page) })
		if errors.Is(err, storage.ErrCursorStale) && !restarted && !cur.Received.IsStart() {
			cur.Received, restarted = storage.Cursor{}, true
			cur.release(DirectionReceived, func(Skip) bool { return false })
			continue
		}
		if err != nil {
			return cur, res, fmt.Errorf("receiving: %w", err)
		}
		if err := receive(ctx, local, &cur, cur.Received, set, &res); err != nil {
			return cur, res, err
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

package openingbook

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"sort"
	"sync"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/rollout"
)

// FullSettings are the corrective book's rollout settings (ADR-0094): every
// game played to its end at 2 ply, cubeful, 1296 games (every ordered pair
// of opening rolls once), a candidate stopped at JSD 3 after 324, the 4 best
// plays at 2 ply. Only the generator reads them: the exercise reads lines,
// whatever settings their signature names.
func FullSettings() rollout.Settings {
	return rollout.Settings{Truncation: 0, MinGames: 324, MaxGames: 1296, JSDLimit: 3, Ply: 2, Candidates: 4, Seed: rollout.DefaultSeed}
}

// Filter narrows the jobs to a chosen subset, to check a sample against
// another engine or to roll a few lines first. An empty list allows all.
type Filter struct {
	Openings []Roll
	Replies  []Roll
}

func (f Filter) allows(k Key) bool {
	in := func(rs []Roll, r Roll) bool {
		if len(rs) == 0 {
			return true
		}
		for _, x := range rs {
			if x == r {
				return true
			}
		}
		return false
	}
	return in(f.Openings, k.Opening) && (k.IsOpening() || in(f.Replies, k.Reply))
}

// OpeningJobs is phase 1: every opening roll at every score, openings first.
func OpeningJobs(scores []Score, f Filter) []Key {
	var out []Key
	for _, o := range Openings() {
		for _, s := range scores {
			if k := OpeningKey(o, s); f.allows(k) {
				out = append(out, k)
			}
		}
	}
	return out
}

// ReplyJobs is phase 2, read from the opening lines phase 1 wrote: for each
// opening at each score, the 21 replies after its best play at that score,
// and after each play tied with the best, up to maxAfter plays in all (≤ 1:
// the best only). An opening line not yet rolled gives no reply job: run the
// phases again once it is.
func ReplyJobs(b *Book, scores []Score, maxAfter int, f Filter) []Key {
	var out []Key
	for _, o := range Openings() {
		for _, s := range scores {
			l, ok := b.Lookup(OpeningKey(o, s))
			if !ok {
				continue
			}
			for i, r := range l.Rank() {
				if i >= max(maxAfter, 1) || !r.Tied {
					break
				}
				for _, rep := range Replies() {
					if k := ReplyKey(o, r.Move, rep, s); f.allows(k) {
						out = append(out, k)
					}
				}
			}
		}
	}
	return out
}

// ReadRun reads the lines a run file holds, a torn last line left out and
// the file untouched.
func ReadRun(path string) (*Book, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &Book{lines: map[Key]Line{}}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("openingbook: %w", err)
	}
	data = data[:bytes.LastIndexByte(data, '\n')+1]
	b, err := Compile(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%w (in %s)", err, path)
	}
	return b, nil
}

// Options tune Generate.
type Options struct {
	// Settings are the rollout's; Workers is ignored: each job rolls on one
	// goroutine and Threads jobs run at once, which keeps every core busy
	// to the end of a job and changes no number.
	Settings rollout.Settings
	// Threads is how many jobs run at once; ≤ 0 means one.
	Threads int
	// DryRun counts what is done and left, rolls nothing and writes
	// nothing, not even the cut of a torn line.
	DryRun bool
	// Done, when set, is called after each job is written, from the
	// goroutine that wrote it, under the file's lock.
	Done func(Line, Report)
}

// Report counts a run: Skipped were already in the file, Written finished
// now, Left were neither (an interrupted run).
type Report struct {
	Total, Skipped, Written, Left int
}

// Generate rolls out every job not yet in the file at path and appends one
// Line per finished job, synced before the next is counted. The file is the
// run's only state: run Generate again with the same path and it skips what
// is there. A torn last line (a crash mid-write) is cut off first. A job
// interrupted by ctx is dropped whole, never written short; Generate then
// returns ctx's error with the report of what it wrote.
func Generate(ctx context.Context, path string, jobs []Key, opt Options) (Report, error) {
	rep := Report{Total: len(jobs)}
	done, err := resume(path, opt.DryRun)
	if err != nil {
		return rep, err
	}
	var todo []Key
	for _, k := range jobs {
		if done[k] {
			rep.Skipped++
		} else {
			todo = append(todo, k)
		}
	}
	rep.Left = len(todo)
	if len(todo) == 0 || opt.DryRun {
		return rep, nil
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return rep, fmt.Errorf("openingbook: %w", err)
	}
	defer func() { _ = f.Close() }()

	s := opt.Settings
	s.Workers = 1
	threads := max(opt.Threads, 1)

	var (
		mu       sync.Mutex
		firstErr error
		wg       sync.WaitGroup
	)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	queue := make(chan Key)
	for range threads {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for k := range queue {
				line, err := RollLine(ctx, k, s)
				if err != nil {
					if ctx.Err() == nil {
						mu.Lock()
						if firstErr == nil {
							firstErr = fmt.Errorf("openingbook: %s: %w", k, err)
						}
						mu.Unlock()
						cancel()
					}
					continue
				}
				mu.Lock()
				if err := appendLine(f, line); err != nil {
					if firstErr == nil {
						firstErr = err
					}
					cancel()
				} else {
					rep.Written++
					rep.Left--
					if opt.Done != nil {
						opt.Done(line, rep)
					}
				}
				mu.Unlock()
			}
		}()
	}
feed:
	for _, k := range todo {
		select {
		case queue <- k:
		case <-ctx.Done():
			break feed
		}
	}
	close(queue)
	wg.Wait()
	if firstErr != nil {
		return rep, firstErr
	}
	return rep, ctx.Err()
}

// RollLine rolls one job out. An interrupted rollout is an error: a line is
// whole or absent.
func RollLine(ctx context.Context, k Key, s rollout.Settings) (Line, error) {
	pos, err := k.Position()
	if err != nil {
		return Line{}, err
	}
	// Leaving the exact bearoff table out keeps the book independent of what
	// a machine has generated: two runs of the same settings agree.
	res, err := rollout.Run(ctx, pos, s, rollout.Options{NoBearoffTable: true})
	if err != nil {
		return Line{}, err
	}
	if res.Stop == rollout.StopCancelled {
		return Line{}, context.Canceled
	}
	l := Line{
		Opening:       k.Opening.String(),
		OpeningPlay:   k.Play,
		Score:         k.Score.String(),
		XGID:          "XGID=" + domain.EncodeXGID(&pos),
		EngineVersion: res.EngineVersion,
		Signature:     res.Signature,
		ExactBearoff:  res.ExactBearoff,
		Seed:          res.Settings.Seed,
		Games:         res.Games,
		Stop:          string(res.Stop),
	}
	for _, c := range res.Candidates {
		l.Candidates = append(l.Candidates, Candidate{
			Move: c.Move, Equity: round6(c.Equity), StdErr: round6(c.StdErr),
			Games: c.Games, JSD: math.Round(c.JSD*100) / 100,
		})
	}
	if !k.IsOpening() {
		l.Reply = k.Reply.String()
	}
	sort.SliceStable(l.Candidates, func(i, j int) bool { return l.Candidates[i].Equity > l.Candidates[j].Equity })
	return l, nil
}

// round6 keeps a millionth of a point, far below any σ the book carries.
func round6(x float64) float64 { return math.Round(x*1e6) / 1e6 }

func appendLine(f *os.File, l Line) error {
	var b bytes.Buffer
	if err := lineEncoder(&b).Encode(l); err != nil {
		return err
	}
	if _, err := f.Write(b.Bytes()); err != nil {
		return fmt.Errorf("openingbook: %w", err)
	}
	if err := f.Sync(); err != nil {
		return fmt.Errorf("openingbook: %w", err)
	}
	return nil
}

// resume reads the keys already in the file at path, cutting a torn last
// line off. A missing file is an empty run. A whole line that does not
// parse is not a crash's trace but a damaged file: refused.
func resume(path string, readOnly bool) (map[Key]bool, error) {
	done := map[Key]bool{}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return done, nil
	}
	if err != nil {
		return nil, fmt.Errorf("openingbook: %w", err)
	}
	if end := bytes.LastIndexByte(data, '\n') + 1; end < len(data) {
		if !readOnly {
			if err := os.Truncate(path, int64(end)); err != nil {
				return nil, fmt.Errorf("openingbook: cutting a torn line: %w", err)
			}
		}
		data = data[:end]
	}
	b, err := Compile(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%w (in %s)", err, path)
	}
	for k := range b.lines {
		done[k] = true
	}
	return done, nil
}

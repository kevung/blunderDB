package maintenance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// Kind names one line of the state of health. The spelling is the CLI's
// argument, the daemon's request field and the journal's key suffix.
type Kind string

const (
	// AnalysisMissing counts the positions that carry no analysis at all
	// (ADR-0013's gap rule). Its action is the gammonNet sweep, which needs
	// an engine and stays with the callers that hold one.
	AnalysisMissing Kind = "analysis.missing"
	// StatsMatches counts the matches without a match_stats row and fills
	// them.
	StatsMatches Kind = "stats.matches"
	// AnalysisColumns recomputes the scalar columns every analysis carries
	// beside its blob. Counting would cost the action itself.
	AnalysisColumns Kind = "analysis.columns"
	// MovesErrors scores the played moves that have no error yet.
	MovesErrors Kind = "moves.errors"
	// AnalysisLegacy rewrites the analyses stored in the JSON format of
	// older releases in the binary one (ADR-0070).
	AnalysisLegacy Kind = "analysis.legacy"
	// Trash counts the entries of the trash. Emptying it is the trash's own
	// gesture.
	Trash Kind = "trash"
	// Vacuum counts the bytes of free pages the file holds. Compacting is
	// the vacuum command's (or /ops/maintenance.vacuum's), and only SQLite
	// has a file to compact.
	Vacuum Kind = "vacuum"
)

// Tier is what a count costs on a large library.
type Tier string

const (
	// TierFree is computed whenever the status is read.
	TierFree Tier = "free"
	// TierSeconds takes seconds on a large library: asked for.
	TierSeconds Tier = "seconds"
	// TierScan reads a whole table: asked for.
	TierScan Tier = "scan"
	// TierNone has no counter; the action's result stands in for one.
	TierNone Tier = "none"
)

// Outcome is how a run ended.
const (
	OutcomeDone      = "done"
	OutcomeCancelled = "cancelled"
	OutcomeFailed    = "failed"
)

// Line is one operation of the state of health.
type Line struct {
	Kind Kind `json:"kind"`
	Tier Tier `json:"tier"`
	// Count is set when the line has a counter and it was computed: always
	// for a free one in Status, never for the others there.
	Count *int64 `json:"count,omitempty"`
	// Unit names what Count counts: positions, matches, entries or bytes.
	Unit string `json:"unit,omitempty"`
	// Listable says IDs answers for this line: the ids of what Count counts.
	Listable bool `json:"listable"`
	// Runnable says Run carries the line's action.
	Runnable bool `json:"runnable"`
	// Applicable is false when the backend has nothing to do for the line
	// (Vacuum on PostgreSQL); the line is then neither counted nor run.
	Applicable bool `json:"applicable"`
	// Last is the line's last requested run, nil when none was recorded.
	Last *Run `json:"last,omitempty"`
}

// Run is one requested action, as the journal keeps it.
type Run struct {
	Kind       Kind   `json:"kind"`
	StartedAt  int64  `json:"startedAt"`
	FinishedAt int64  `json:"finishedAt"`
	Changed    int    `json:"changed"`
	Outcome    string `json:"outcome"`
}

// Opts tunes a Run.
type Opts struct {
	// Before, when not nil, is called around each unit of work the pass
	// commits and returns the function that ends it: the desktop wrapper
	// takes its lock there. A pass without batches of its own calls it once.
	Before func() func()
	// Progress, when not nil, is told how far the pass is; total is 0 when
	// the pass does not know it in advance.
	Progress func(done, total int)
}

// ErrUnknownKind is a kind no line carries.
var ErrUnknownKind = fmt.Errorf("unknown maintenance operation: %w", storage.ErrInvalid)

// reclaimer is satisfied by the SQLite backend only; duck-typed like the
// daemon's vacuumer, so PostgreSQL needs no stub.
type reclaimer interface {
	Reclaimable(ctx context.Context) (int64, error)
}

type spec struct {
	kind  Kind
	tier  Tier
	unit  string
	count func(ctx context.Context, st storage.Storage, scope string) (int64, error)
	ids   func(ctx context.Context, st storage.Storage, scope string) ([]int64, error)
	run   func(ctx context.Context, st storage.Storage, scope string, o Opts) (int, error)
	// applies is nil when the line applies to every backend.
	applies func(st storage.Storage) bool
}

// specs is the table of lines, in the order a client shows them.
var specs = []spec{
	{
		kind: AnalysisMissing, tier: TierSeconds, unit: "positions",
		ids: positionsWithoutAnalysis,
	},
	{
		kind: StatsMatches, tier: TierFree, unit: "matches",
		ids: func(ctx context.Context, st storage.Storage, scope string) ([]int64, error) {
			return st.Stats().MissingMatchStats(ctx, scope)
		},
		run: func(ctx context.Context, st storage.Storage, scope string, o Opts) (int, error) {
			defer begin(o)()
			return st.Stats().FillMatchStats(ctx, scope, o.Progress)
		},
	},
	{
		kind: AnalysisColumns, tier: TierNone,
		run: func(ctx context.Context, st storage.Storage, scope string, o Opts) (int, error) {
			defer begin(o)()
			return st.Analyses().RepairDenormalisedColumns(ctx, scope)
		},
	},
	{
		kind: MovesErrors, tier: TierNone,
		run: func(ctx context.Context, st storage.Storage, scope string, o Opts) (int, error) {
			var progress func(int)
			if o.Progress != nil {
				progress = func(scored int) { o.Progress(scored, 0) }
			}
			return storage.ScoreAllMoves(ctx, st.Matches(), scope, o.Before, progress)
		},
	},
	{
		kind: AnalysisLegacy, tier: TierNone,
		run: func(ctx context.Context, st storage.Storage, scope string, o Opts) (int, error) {
			return storage.ReencodeAllAnalyses(ctx, st.Analyses(), scope, o.Before)
		},
	},
	{
		kind: Trash, tier: TierFree, unit: "entries",
		count: func(ctx context.Context, st storage.Storage, scope string) (int64, error) {
			n, err := st.Trash().Count(ctx, scope)
			return int64(n), err
		},
	},
	{
		kind: Vacuum, tier: TierFree, unit: "bytes",
		count: func(ctx context.Context, st storage.Storage, _ string) (int64, error) {
			return st.(reclaimer).Reclaimable(ctx)
		},
		applies: func(st storage.Storage) bool {
			_, ok := st.(reclaimer)
			return ok
		},
	},
}

func lookup(kind Kind) (spec, error) {
	for _, s := range specs {
		if s.kind == kind {
			return s, nil
		}
	}
	return spec{}, fmt.Errorf("%q: %w", kind, ErrUnknownKind)
}

// Kinds lists every line's kind, in display order.
func Kinds() []Kind {
	out := make([]Kind, len(specs))
	for i, s := range specs {
		out[i] = s.kind
	}
	return out
}

func (s spec) applicable(st storage.Storage) bool {
	return s.applies == nil || s.applies(st)
}

func (s spec) counted() bool { return s.count != nil || s.ids != nil }

// countOf computes the line's count: its own counter, else the length of
// its list, so the two cannot disagree.
func (s spec) countOf(ctx context.Context, st storage.Storage, scope string) (int64, error) {
	if s.count != nil {
		return s.count(ctx, st, scope)
	}
	ids, err := s.ids(ctx, st, scope)
	return int64(len(ids)), err
}

// Status returns every line with its free count and its last run. It reads
// only: nothing is written, whatever the journal holds.
func Status(ctx context.Context, st storage.Storage, scope string) ([]Line, error) {
	log, err := st.LibrarySettings().MaintenanceLog(ctx, scope)
	if err != nil {
		return nil, err
	}
	lines := make([]Line, 0, len(specs))
	for _, s := range specs {
		line := Line{
			Kind:       s.kind,
			Tier:       s.tier,
			Unit:       s.unit,
			Listable:   s.ids != nil,
			Runnable:   s.run != nil,
			Applicable: s.applicable(st),
			Last:       parseRun(log[storage.MaintenanceKeyPrefix+string(s.kind)]),
		}
		if line.Applicable && s.tier == TierFree && s.counted() {
			n, err := s.countOf(ctx, st, scope)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", s.kind, err)
			}
			line.Count = &n
		}
		lines = append(lines, line)
	}
	return lines, nil
}

// Count computes one line's count, whatever its tier. A line without a
// counter, or one that does not apply to the backend, is refused.
func Count(ctx context.Context, st storage.Storage, scope string, kind Kind) (int64, error) {
	s, err := lookup(kind)
	if err != nil {
		return 0, err
	}
	if !s.counted() {
		return 0, fmt.Errorf("%s has no counter: %w", kind, storage.ErrInvalid)
	}
	if !s.applicable(st) {
		return 0, fmt.Errorf("%s does not apply to this backend: %w", kind, storage.ErrInvalid)
	}
	return s.countOf(ctx, st, scope)
}

// IDs lists, in ascending order, the ids one line counts: positions or
// matches, as the line's Unit says. Its length is Count's answer.
func IDs(ctx context.Context, st storage.Storage, scope string, kind Kind) ([]int64, error) {
	s, err := lookup(kind)
	if err != nil {
		return nil, err
	}
	if s.ids == nil {
		return nil, fmt.Errorf("%s has no list of ids: %w", kind, storage.ErrInvalid)
	}
	ids, err := s.ids(ctx, st, scope)
	if err != nil {
		return nil, err
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids, nil
}

// now is the journal's clock; a test pins it.
var now = time.Now

// RunKind performs one line's action and records it in the journal, whatever
// its outcome: done, cancelled (the context ended) or failed. The returned
// error is the action's; a journal that cannot be written is reported only
// when the action itself succeeded.
func RunKind(ctx context.Context, st storage.Storage, scope string, kind Kind, o Opts) (Run, error) {
	s, err := lookup(kind)
	if err != nil {
		return Run{}, err
	}
	if s.run == nil {
		return Run{}, fmt.Errorf("%s has no action here: %w", kind, storage.ErrInvalid)
	}
	if !s.applicable(st) {
		return Run{}, fmt.Errorf("%s does not apply to this backend: %w", kind, storage.ErrInvalid)
	}
	r := Run{Kind: kind, StartedAt: now().Unix()}
	changed, runErr := s.run(ctx, st, scope, o)
	r.Changed = changed
	r.FinishedAt = now().Unix()
	switch {
	case runErr == nil:
		r.Outcome = OutcomeDone
	case errors.Is(runErr, context.Canceled) || errors.Is(runErr, context.DeadlineExceeded):
		r.Outcome = OutcomeCancelled
	default:
		r.Outcome = OutcomeFailed
	}
	// The journal outlives the cancellation that ended the run: a cancelled
	// pass is recorded as such.
	recErr := record(context.WithoutCancel(ctx), st, scope, r, o.Before)
	if runErr != nil {
		return r, runErr
	}
	return r, recErr
}

func record(ctx context.Context, st storage.Storage, scope string, r Run, before func() func()) error {
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	defer begin(Opts{Before: before})()
	return st.LibrarySettings().RecordMaintenance(ctx, scope, storage.MaintenanceKeyPrefix+string(r.Kind), string(b))
}

// parseRun reads one journal row; an unreadable row reads as no run, like
// any comfort row of the library settings.
func parseRun(v string) *Run {
	if v == "" {
		return nil
	}
	var r Run
	if json.Unmarshal([]byte(v), &r) != nil || r.Kind == "" {
		return nil
	}
	return &r
}

// begin calls Before once, for a pass that commits in one piece.
func begin(o Opts) func() {
	if o.Before == nil {
		return func() {}
	}
	return o.Before()
}

func positionsWithoutAnalysis(ctx context.Context, st storage.Storage, scope string) ([]int64, error) {
	var ids []int64
	for p, err := range st.Analyses().WithoutAnalysis(ctx, scope, storage.ListOpts{}) {
		if err != nil {
			return nil, err
		}
		ids = append(ids, p.ID)
	}
	return ids, ctx.Err()
}

// Package rollouts writes rollouts down on a library: one position on
// demand, or every position a search selects. It composes the rollout engine
// (pkg/blunderdb/engine/rollout) with the Storage contract, so the GUI and
// the CLI (through database.Database) and the serve daemon run the same
// gather, the same loop and the same write (CLI/GUI/server parity).
//
// A rollout is a second Analysis with its own Configuration (ADR-0060 §8):
// it is attached beside what a position already holds and replaces nothing
// (ADR-0013). A batch writes each position as it is produced and skips the
// positions already carrying a rollout of the same Signature, so stopping
// costs nothing and running again resumes.
package rollouts

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/rollout"
	"github.com/kevung/blunderdb/pkg/blunderdb/searchquery"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// ErrPartial refuses to store a cancelled rollout: its numbers are those of
// the games played so far, not of its Configuration.
var ErrPartial = errors.New("rollouts: a cancelled rollout is not stored")

// loadBatch bounds one LoadMany round trip while gathering.
const loadBatch = 500

// Store attaches res to the analysis of positionID, creating the analysis row
// when the position has none.
func Store(ctx context.Context, st storage.Storage, scope string, positionID int64, res *rollout.Result) error {
	if res == nil || res.Stop == rollout.StopCancelled {
		return ErrPartial
	}
	now := time.Now()
	return update(ctx, st, scope, positionID, func(a *domain.PositionAnalysis) {
		if a.CreationDate.IsZero() {
			a.CreationDate = now
		}
		a.AttachRollout(res.Record(now))
		a.LastModifiedDate = now
	})
}

// SaveAnalysis writes a over positionID's analysis and keeps the rollouts
// stored there: they are analyses of their own, which a caller writing an
// Evaluation knows nothing of (ADR-0060).
func SaveAnalysis(ctx context.Context, st storage.Storage, scope string, positionID int64, a *domain.PositionAnalysis) error {
	return update(ctx, st, scope, positionID, func(existing *domain.PositionAnalysis) {
		rollouts := domain.MergeRollouts(existing.Rollouts, a.Rollouts)
		*existing = *a
		existing.Rollouts = rollouts
	})
}

// update reads positionID's analysis, an empty one when it has none, lets
// change rewrite it and writes it back, in one guarded transaction: two
// writers on the same position cannot both read the row before either writes.
func update(ctx context.Context, st storage.Storage, scope string, positionID int64, change func(*domain.PositionAnalysis)) error {
	tx, err := storage.BeginGuarded(ctx, st, fmt.Sprintf("analysis:%s:%d", scope, positionID))
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	a, err := tx.Analyses().Load(ctx, scope, positionID)
	switch {
	case errors.Is(err, storage.ErrNotFound):
		a = &domain.PositionAnalysis{PositionID: int(positionID)}
	case err != nil:
		return err
	}
	change(a)
	if err := tx.Analyses().Save(ctx, scope, positionID, a); err != nil {
		return err
	}
	return tx.Commit()
}

// List returns the rollouts stored on positionID, newest first; none when it
// has no analysis.
func List(ctx context.Context, st storage.Storage, scope string, positionID int64) ([]domain.RolloutAnalysis, error) {
	a, err := st.Analyses().Load(ctx, scope, positionID)
	if errors.Is(err, storage.ErrNotFound) {
		return []domain.RolloutAnalysis{}, nil
	}
	if err != nil {
		return nil, err
	}
	if a.Rollouts == nil {
		return []domain.RolloutAnalysis{}, nil
	}
	return a.Rollouts, nil
}

// Gather snapshots the positions f selects that do not yet carry a rollout of
// s's Signature, by ascending id as the search returns them. A snapshot, not
// a cursor: the batch writes the rows the search reads.
func Gather(ctx context.Context, st storage.Storage, scope string, f domain.SearchFilters, s rollout.Settings) ([]domain.Position, error) {
	var found []domain.Position
	for p, err := range st.Search().Find(ctx, scope, f, storage.ListOpts{}) {
		if err != nil {
			return nil, err
		}
		found = append(found, *p)
	}
	out := found[:0]
	for start := 0; start < len(found); start += loadBatch {
		chunk := found[start:min(start+loadBatch, len(found))]
		ids := make([]int64, len(chunk))
		for i := range chunk {
			ids[i] = chunk[i].ID
		}
		analyses, err := st.Analyses().LoadMany(ctx, scope, ids)
		if err != nil {
			return nil, err
		}
		for _, p := range chunk {
			if !analyses[p.ID].HasRollout(s.SignatureAt(&p)) {
				out = append(out, p)
			}
		}
	}
	return out, nil
}

// Progress is reported after every batch of games of the position being
// rolled out, and once more when it is written.
type Progress struct {
	// Done positions of Total are behind; PositionID is the one in hand.
	Done       int   `json:"done"`
	Total      int   `json:"total"`
	PositionID int64 `json:"positionId"`
	// Games of MaxGames are behind on PositionID.
	Games    int `json:"games"`
	MaxGames int `json:"maxGames"`
}

// Summary is what a batch did. RolledOut were written; Refused could not be
// rolled out (a position the engine declines: no legal play, a game already
// over, a score beyond the match table); Failed were rolled out but not
// written.
type Summary struct {
	Total     int  `json:"total"`
	RolledOut int  `json:"rolledOut"`
	Refused   int  `json:"refused"`
	Failed    int  `json:"failed"`
	Cancelled bool `json:"cancelled"`
}

// Write stores one finished rollout; the caller supplies it so that it runs
// under the caller's own lock.
type Write func(positionID int64, res *rollout.Result) error

// Batch rolls positions out one after the other — each rollout already plays
// its games on every core — writing each as it finishes. Cancelling ctx drops
// the position in hand whole and returns what was written before it.
func Batch(ctx context.Context, positions []domain.Position, s rollout.Settings, progress func(Progress), write Write) (Summary, error) {
	sum := Summary{Total: len(positions)}
	if err := s.Validate(); err != nil {
		return sum, err
	}
	report := func(p Progress) {
		if progress != nil {
			progress(p)
		}
	}
	for i, pos := range positions {
		if ctx.Err() != nil {
			sum.Cancelled = true
			return sum, nil
		}
		opt := rollout.Options{Progress: func(p rollout.Progress) {
			report(Progress{Done: i, Total: sum.Total, PositionID: pos.ID, Games: p.Games, MaxGames: p.MaxGames})
		}}
		res, err := rollout.Run(ctx, pos, s, opt)
		if ctx.Err() != nil {
			sum.Cancelled = true
			return sum, nil
		}
		switch {
		case err != nil:
			sum.Refused++
			slog.Info("rollout batch: position not rolled out", "position_id", pos.ID, "error", err)
		default:
			if werr := write(pos.ID, res); werr != nil {
				sum.Failed++
				slog.Warn("rollout batch: storing the rollout failed", "position_id", pos.ID, "error", werr)
			} else {
				sum.RolledOut++
			}
		}
		games := 0
		if res != nil {
			games = res.Games
		}
		report(Progress{Done: i + 1, Total: sum.Total, PositionID: pos.ID, Games: games, MaxGames: s.MaxGames})
	}
	return sum, nil
}

// ErrLoad wraps a failure to read the position a rollout was asked on, so a
// caller can tell it from the engine refusing the request.
var ErrLoad = errors.New("rollouts: cannot load position")

// Position loads positionID and rolls it out, writing nothing: the caller
// stores the result with Store when asked to. Cancelled, it returns the games
// finished so far (Stop = cancelled) with ctx's error.
func Position(ctx context.Context, st storage.Storage, scope string, positionID int64, s rollout.Settings, moves []string, progress func(rollout.Progress)) (*rollout.Result, error) {
	pos, err := Load(ctx, st, scope, positionID)
	if err != nil {
		return nil, err
	}
	return Run(ctx, pos, s, moves, progress)
}

// Load reads the position a rollout is asked on, a failure wrapped in
// ErrLoad. Position is Load then Run; a caller that must hold a lock while
// reading, and not while playing, calls the two itself.
func Load(ctx context.Context, st storage.Storage, scope string, positionID int64) (*domain.Position, error) {
	pos, err := st.Positions().Load(ctx, scope, positionID)
	if err != nil {
		return nil, fmt.Errorf("%w %d: %w", ErrLoad, positionID, err)
	}
	return pos, nil
}

// Run rolls pos out: its plays, moves naming them when set, or its cube
// decision.
func Run(ctx context.Context, pos *domain.Position, s rollout.Settings, moves []string, progress func(rollout.Progress)) (*rollout.Result, error) {
	return rollout.Run(ctx, *pos, s, rollout.Options{Moves: moves, Progress: progress})
}

// ParseQuery reads query as the search bar does, refusing a token it does not
// know: a typo must not widen a long rollout to the whole library.
func ParseQuery(query string) (domain.SearchFilters, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return domain.SearchFilters{}, nil
	}
	filters, diags := searchquery.Parse(q)
	var unknown []string
	for _, d := range diags {
		if d.Kind == searchquery.DiagUnknown {
			unknown = append(unknown, d.Token)
		}
	}
	if len(unknown) > 0 {
		return domain.SearchFilters{}, fmt.Errorf("unknown token(s) in query: %s", strings.Join(unknown, ", "))
	}
	return filters, nil
}

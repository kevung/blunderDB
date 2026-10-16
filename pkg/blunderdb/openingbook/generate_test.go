package openingbook

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine/rollout"
)

// tiny is the cheapest rollout that still runs every step: 36 games, two
// half-moves, two candidates.
func tiny() rollout.Settings {
	return rollout.Settings{Truncation: 2, MinGames: 36, MaxGames: 36, Ply: 0, Candidates: 2, Seed: rollout.DefaultSeed}
}

func TestGenerate_ResumesWithoutRedoing(t *testing.T) {
	if testing.Short() {
		t.Skip("rolls three positions out")
	}
	path := filepath.Join(t.TempDir(), "run.jsonl")
	jobs := []Key{
		OpeningKey(Roll{3, 1}, Score{4, 2}),
		ReplyKey(Roll{3, 1}, "8/5 6/5", Roll{5, 2}, Score{1, 4}),
		ReplyKey(Roll{6, 5}, "24/13", Roll{6, 6}, Score{3, 3}),
	}

	ctx, cancel := context.WithCancel(context.Background())
	rep, err := Generate(ctx, path, jobs, Options{Settings: tiny(), Threads: 1, Done: func(Line, Report) { cancel() }})
	if !errors.Is(err, context.Canceled) || rep.Written != 1 || rep.Left != 2 {
		t.Fatalf("interrupted run: %+v, %v; want 1 written, 2 left, cancelled", rep, err)
	}

	// A crash in the middle of a write leaves a torn line behind.
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString(`{"opening":"31","rep`)
	_ = f.Close()

	var written []Key
	rep, err = Generate(context.Background(), path, jobs, Options{Settings: tiny(), Threads: 2, Done: func(l Line, _ Report) {
		k, _ := l.Key()
		written = append(written, k)
	}})
	if err != nil || rep.Skipped != 1 || rep.Written != 2 || rep.Left != 0 {
		t.Fatalf("resumed run: %+v, %v; want 1 skipped, 2 written", rep, err)
	}
	if slices.Contains(written, jobs[0]) {
		t.Fatalf("resumed run redid %s", jobs[0])
	}

	f2, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f2.Close() }()
	book, err := Load(f2)
	if err != nil {
		t.Fatalf("run file is not a clean book: %v", err)
	}
	if book.Len() != 3 {
		t.Fatalf("%d lines, want 3", book.Len())
	}
	for _, k := range jobs {
		l, _ := book.Lookup(k)
		if len(l.Candidates) != 2 || l.Games != 36 || l.Signature == "" || l.ExactBearoff || l.XGID == "" {
			t.Fatalf("%s: %+v", k, l)
		}
		if r := l.Rank(); r[0].Equity < r[1].Equity {
			t.Fatalf("%s: candidates not best first", k)
		}
	}

	rep, err = Generate(context.Background(), path, jobs, Options{Settings: tiny()})
	if err != nil || rep.Skipped != 3 || rep.Written != 0 {
		t.Fatalf("finished run rerun: %+v, %v", rep, err)
	}
}

func TestReplyJobs_FollowTheTiedOpeningPlays(t *testing.T) {
	b := testBook(t)
	s := []Score{{4, 2}}
	if n := len(ReplyJobs(b, s, 2, Filter{})); n != 2*21+21 {
		t.Fatalf("%d jobs after the tied plays, want 63", n)
	}
	if n := len(ReplyJobs(b, s, 1, Filter{})); n != 42 {
		t.Fatalf("%d jobs after the best plays, want 42", n)
	}
	f := Filter{Openings: []Roll{{3, 1}}, Replies: []Roll{{2, 1}}}
	jobs := ReplyJobs(b, s, 2, f)
	if len(jobs) != 2 || jobs[0].Play != "8/5 6/5" || jobs[1].Play != "24/23 13/10" {
		t.Fatalf("filtered jobs %v", jobs)
	}
	if n := len(OpeningJobs(GridScores(), f)); n != 20 {
		t.Fatalf("%d opening jobs for one roll on the grid, want 20", n)
	}
	if n := len(ReplyJobs(&Book{}, s, 2, Filter{})); n != 0 {
		t.Fatalf("no opening line, %d reply jobs", n)
	}
}

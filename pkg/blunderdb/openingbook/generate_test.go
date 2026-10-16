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
		{Roll{3, 1}, Roll{5, 2}, Score{4, 2}},
		{Roll{3, 1}, Roll{5, 2}, Score{1, 4}},
		{Roll{6, 5}, Roll{6, 6}, Score{3, 3}},
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
		l, _ := book.Lookup(k.Opening, k.Reply, k.Score)
		if len(l.Candidates) != 2 || l.Games != 36 || l.Signature == "" || l.XGID == "" {
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

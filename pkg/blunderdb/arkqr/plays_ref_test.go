package arkqr

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// The steps of a decoded play are part of what an archive restores, so the
// fast exploration must find, for every board, the very sequence the plain
// one below finds first. referencePlays is that plain exploration: every
// order of every move, no table kept between rolls.

func (r rel) referenceSingle(d int) []step {
	var srcs []int
	if r.my[25] > 0 {
		srcs = []int{25}
	} else {
		for p := 24; p >= 1; p-- {
			if r.my[p] > 0 {
				srcs = append(srcs, p)
			}
		}
	}
	hi := 0
	if len(srcs) > 0 {
		hi = srcs[0]
	}
	var out []step
	for _, p := range srcs {
		dst := p - d
		if dst >= 1 {
			o := r.opp[25-dst]
			if o >= 2 {
				continue
			}
			out = append(out, step{int8(p), int8(dst), o == 1})
			continue
		}
		if hi > 6 {
			continue
		}
		if dst == 0 || p == hi {
			out = append(out, step{int8(p), 0, false})
		}
	}
	return out
}
func (r rel) referencePlays(d1, d2 int) []play {
	if d1 < d2 {
		d1, d2 = d2, d1
	}
	type term struct {
		used, first int
		steps       []step
	}
	found := map[rel][]term{}
	var order []rel
	best := 0
	var rec func(b rel, dice []int, used, first int, steps []step)
	rec = func(b rel, dice []int, used, first int, steps []step) {
		moved := false
		if len(dice) > 0 {
			for _, s := range b.referenceSingle(dice[0]) {
				nb, _ := b.applyStep(s)
				f := first
				if f == 0 {
					f = dice[0]
				}
				moved = true
				rec(nb, dice[1:], used+1, f, append(append([]step(nil), steps...), s))
			}
		}
		if !moved {
			best = max(best, used)
			if _, ok := found[b]; !ok {
				order = append(order, b)
			}
			found[b] = append(found[b], term{used, first, steps})
		}
	}
	if d1 == d2 {
		rec(r, []int{d1, d1, d1, d1}, 0, 0, nil)
	} else {
		rec(r, []int{d1, d2}, 0, 0, nil)
		rec(r, []int{d2, d1}, 0, 0, nil)
	}
	larger := false
	if d1 != d2 && best == 1 {
		for _, b := range order {
			for _, t := range found[b] {
				if t.used == 1 && t.first == d1 {
					larger = true
				}
			}
		}
	}
	var out []play
	for _, b := range order {
		for _, t := range found[b] {
			if t.used == best && (!larger || t.first == d1) {
				out = append(out, play{b, t.steps})
				break
			}
		}
	}
	sortPlays(out)
	return out
}

// TestPlaysMatchReference compares the two on every position of the corpus,
// for each of the 36 rolls.
func TestPlaysMatchReference(t *testing.T) {
	n := 0
	docs := map[string]transcript.Document{}
	for _, cm := range loadCorpus(t) {
		docs[cm.name] = cm.doc
	}
	if os.Getenv(externalCorpus) != "" {
		for i, m := range loadExternal(t) {
			docs[fmt.Sprintf("external %d", i)] = m.Doc
		}
	}
	for name, doc := range docs {
		info := replayInfo(Match{Doc: doc})
		for i, a := range doc.Actions {
			if !info[i].HasPosition {
				continue
			}
			r := relOf(info[i].Before.Board, a.Side)
			for d1 := 1; d1 <= 6; d1++ {
				for d2 := 1; d2 <= 6; d2++ {
					got, want := r.plays(d1, d2), r.referencePlays(d1, d2)
					if !reflect.DeepEqual(got, want) {
						t.Fatalf("%s: action %d, roll %d%d: plays differ", name, i, d1, d2)
					}
					n++
				}
			}
		}
	}
	t.Logf("%d rolls", n)
}

// TestCanMoveMatchesSingle checks the shortcut the dice law of version 2
// takes against the moves themselves.
func TestCanMoveMatchesSingle(t *testing.T) {
	for _, cm := range loadCorpus(t) {
		info := replayInfo(Match{Doc: cm.doc})
		for i, a := range cm.doc.Actions {
			if !info[i].HasPosition {
				continue
			}
			r := relOf(info[i].Before.Board, a.Side)
			for d := 1; d <= 6; d++ {
				if r.canMove(d) != (len(r.single(d)) > 0) {
					t.Fatalf("%s %d die %d", cm.name, i, d)
				}
			}
		}
	}
}

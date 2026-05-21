package sqlite_test

import (
	"context"
	"reflect"
	"testing"
	"time"
)

// TestCommandHistory covers Save (oldest-first ordering preserved), Load and
// Clear.
func TestCommandHistory(t *testing.T) {
	ctx := context.Background()
	s := openMem(t)

	for _, cmd := range []string{"first", "second", "third"} {
		if err := s.History().Save(ctx, "", cmd); err != nil {
			t.Fatalf("Save %q: %v", cmd, err)
		}
	}

	got, err := s.History().Load(ctx, "")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !reflect.DeepEqual(got, []string{"first", "second", "third"}) {
		t.Errorf("Load: got %v, want [first second third]", got)
	}

	if err := s.History().Clear(ctx, ""); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	got, _ = s.History().Load(ctx, "")
	if len(got) != 0 {
		t.Errorf("after Clear: got %v, want empty", got)
	}
}

// TestSearchHistory covers Save, List (most recent first) and DeleteEntry.
func TestSearchHistory(t *testing.T) {
	ctx := context.Background()
	s := openMem(t)

	// A short pause between saves keeps the millisecond timestamps distinct,
	// so DeleteEntry(timestamp) targets exactly one row.
	for i, c := range []string{"alpha", "beta", "gamma"} {
		if i > 0 {
			time.Sleep(2 * time.Millisecond)
		}
		if err := s.SearchHistory().Save(ctx, "", c, "{}"); err != nil {
			t.Fatalf("Save %q: %v", c, err)
		}
	}

	var cmds []string
	var deleteTS int64
	for e, err := range s.SearchHistory().List(ctx, "") {
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		cmds = append(cmds, e.Command)
		if e.Command == "beta" {
			deleteTS = e.Timestamp
		}
	}
	if len(cmds) != 3 || cmds[0] != "gamma" {
		t.Errorf("List order: got %v, want most-recent-first starting with gamma", cmds)
	}

	if err := s.SearchHistory().DeleteEntry(ctx, "", deleteTS); err != nil {
		t.Fatalf("DeleteEntry: %v", err)
	}
	n := 0
	for e, err := range s.SearchHistory().List(ctx, "") {
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if e.Command == "beta" {
			t.Error("deleted entry still present")
		}
		n++
	}
	if n != 2 {
		t.Errorf("count after DeleteEntry: got %d, want 2", n)
	}
}

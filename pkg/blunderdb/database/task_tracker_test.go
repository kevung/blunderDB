package database

import (
	"errors"
	"testing"
)

func TestTracked_EndsOnceOnEveryPath(t *testing.T) {
	d := NewDatabase()
	var ends []error
	SetTaskTracker(d, func(kind, label string, cancel func()) func(error) {
		if kind != TaskImport || label != "f.xg" {
			t.Errorf("told %q %q", kind, label)
		}
		return func(err error) { ends = append(ends, err) }
	})
	boom := errors.New("boom")
	if _, err := tracked(d, TaskImport, "f.xg", nil, func() (int, error) { return 0, boom }); !errors.Is(err, boom) {
		t.Fatal(err)
	}
	func() {
		defer func() { _ = recover() }()
		_, _ = tracked(d, TaskImport, "f.xg", nil, func() (int, error) { panic("x") })
	}()
	if len(ends) != 2 || !errors.Is(ends[0], boom) || !errors.Is(ends[1], errTaskPanicked) {
		t.Fatalf("ends %v", ends)
	}
	SetTaskTracker(d, nil)
	if v, _ := tracked(d, TaskImport, "", nil, func() (int, error) { return 7, nil }); v != 7 || len(ends) != 2 {
		t.Fatal("an untracked call was told to a removed tracker")
	}
}

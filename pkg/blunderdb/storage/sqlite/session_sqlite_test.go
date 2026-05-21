package sqlite_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// TestSessionSaveLoad round-trips a fully populated session state.
func TestSessionSaveLoad(t *testing.T) {
	ctx := context.Background()
	s := openMem(t)

	want := storage.SessionState{
		LastSearchCommand:  "decision checker",
		LastSearchPosition: `{"board":1}`,
		LastPositionIndex:  3,
		LastPositionIDs:    []int64{10, 20, 30},
		HasActiveSearch:    true,
		ViewsJSON:          `[{"tab":1}]`,
	}
	if err := s.Session().Save(ctx, "", want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := s.Session().Load(ctx, "")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !reflect.DeepEqual(*got, want) {
		t.Errorf("round-trip mismatch:\n got %+v\nwant %+v", *got, want)
	}
}

// TestSessionLoadEmpty checks that a database that never stored a session
// loads a zero-valued SessionState without error.
func TestSessionLoadEmpty(t *testing.T) {
	ctx := context.Background()
	s := openMem(t)

	got, err := s.Session().Load(ctx, "")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !reflect.DeepEqual(*got, storage.SessionState{}) {
		t.Errorf("empty Load: got %+v, want zero value", *got)
	}
}

// TestSessionClear verifies Clear removes a previously saved session.
func TestSessionClear(t *testing.T) {
	ctx := context.Background()
	s := openMem(t)

	if err := s.Session().Save(ctx, "", storage.SessionState{LastSearchCommand: "x", HasActiveSearch: true}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := s.Session().Clear(ctx, ""); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	got, err := s.Session().Load(ctx, "")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !reflect.DeepEqual(*got, storage.SessionState{}) {
		t.Errorf("after Clear: got %+v, want zero value", *got)
	}
}

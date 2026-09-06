// Contract cases for the library's own settings — the error and blunder
// thresholds of ADR-0043. The two backends store them in different tables
// (metadata on SQLite, the tenant-scoped library_settings on PostgreSQL), so
// what is held identical here is the behaviour: the defaults a library that
// never set them reads, the nesting rule, and the round trip.
// The table that runs them lives in contract.go.
package storagetest

import (
	"context"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func testLibrarySettingsDefaults(t *testing.T, s storage.Storage) {
	ctx := context.Background()

	got, err := s.LibrarySettings().Load(ctx, "")
	if err != nil {
		t.Fatalf("Load on a fresh library: %v", err)
	}
	want := storage.DefaultLibrarySettings()
	if got != want {
		t.Fatalf("fresh library: got %+v, want %+v", got, want)
	}
	// The defaults are the values the code used as constants before they
	// became settings; a change here moves every blunder count ever shown.
	if want.ErrorThresholdMP != 50 || want.BlunderThresholdMP != 100 {
		t.Fatalf("defaults drifted: %+v, want {50 100}", want)
	}
}

func testLibrarySettingsRoundTrip(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ls := s.LibrarySettings()

	set := storage.LibrarySettings{ErrorThresholdMP: 20, BlunderThresholdMP: 80}
	if err := ls.Save(ctx, "", set); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := ls.Load(ctx, "")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != set {
		t.Fatalf("round trip: got %+v, want %+v", got, set)
	}

	// Saving again replaces rather than duplicating: the pair is two rows
	// keyed by name, not an append-only log.
	again := storage.LibrarySettings{ErrorThresholdMP: 40, BlunderThresholdMP: 200}
	if err := ls.Save(ctx, "", again); err != nil {
		t.Fatalf("Save again: %v", err)
	}
	got, err = ls.Load(ctx, "")
	if err != nil {
		t.Fatalf("Load again: %v", err)
	}
	if got != again {
		t.Fatalf("second round trip: got %+v, want %+v", got, again)
	}
}

func testLibrarySettingsRejectsInverted(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ls := s.LibrarySettings()

	ok := storage.LibrarySettings{ErrorThresholdMP: 30, BlunderThresholdMP: 90}
	if err := ls.Save(ctx, "", ok); err != nil {
		t.Fatalf("Save valid: %v", err)
	}

	// Every Blunder is an Error, so an error threshold above the blunder
	// threshold names a set that cannot exist.
	bad := storage.LibrarySettings{ErrorThresholdMP: 120, BlunderThresholdMP: 90}
	if err := ls.Save(ctx, "", bad); !errors.Is(err, storage.ErrInvalid) {
		t.Fatalf("Save inverted: got %v, want ErrInvalid", err)
	}
	// Zero is not "no threshold": it would make a perfect decision an error.
	if err := ls.Save(ctx, "", storage.LibrarySettings{ErrorThresholdMP: 0, BlunderThresholdMP: 90}); !errors.Is(err, storage.ErrInvalid) {
		t.Fatalf("Save zero: got %v, want ErrInvalid", err)
	}

	// A refused save leaves the stored pair alone.
	got, err := ls.Load(ctx, "")
	if err != nil {
		t.Fatalf("Load after refusal: %v", err)
	}
	if got != ok {
		t.Fatalf("refused save changed the library: got %+v, want %+v", got, ok)
	}
}

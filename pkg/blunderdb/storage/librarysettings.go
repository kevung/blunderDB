package storage

import (
	"context"
	"fmt"
	"strconv"
)

// LibrarySettings are the reading habits that belong to the library rather
// than to the machine (ADR-0046, CONTEXT.md "Library setting"): where the line
// between a decision worth revisiting and a decision worth being ashamed of is
// drawn. They travel with the file, so the same database counts the same
// blunders wherever it is opened, and a daemon tenant carries its own.
//
// Both thresholds are in the stored millipoint unit of the error columns
// (cube_error, best_move_equity_error): 1000 mp is one unit of the position's
// Referential — a point at money play, the whole current cube at a match
// score. The GUI shows and takes them in that Referential ("0.080"); the
// command line and the CLI speak millipoints, as `E>x` always has.
type LibrarySettings struct {
	// ErrorThresholdMP is the cost at or above which a decision is an Error.
	ErrorThresholdMP int `json:"errorThresholdMP"`
	// BlunderThresholdMP is the cost at or above which an Error is a Blunder.
	BlunderThresholdMP int `json:"blunderThresholdMP"`
}

// The defaults are the two values the code already drew before they became
// settings: 100 is what the statistics have always called a blunder, and 50 is
// what the study queue of an import already retained ("half of what the
// statistics call a blunder, because the decisions worth revisiting start well
// below the ones worth being ashamed of"). Choosing them means the queue does
// not move and the blunder counts do not move; only the "Errors" columns,
// which used to count every non-zero cost, stop counting three-millipoint
// misses.
const (
	DefaultErrorThresholdMP   = 50
	DefaultBlunderThresholdMP = 100

	// MinThresholdMP is 1 rather than 0 so that a threshold always states a
	// cost: at 0 every decision, including a perfect one, would be an error.
	// The old "any non-zero cost" behaviour is still expressible — it is 1.
	MinThresholdMP = 1
	// MaxThresholdMP is past the largest error either Referential can produce
	// (money play tops out around 2000 mp with the cube); it is a guard
	// against a typo, not a statement about backgammon.
	MaxThresholdMP = 10000
)

// The two rows a backend stores. The keys are the storage-layer vocabulary,
// shared by both backends: SQLite keeps them in the metadata table next to the
// Performance Rating objective, PostgreSQL in its own tenant-scoped table.
const (
	LibrarySettingErrorKey   = "error_threshold_mp"
	LibrarySettingBlunderKey = "blunder_threshold_mp"
)

// DefaultLibrarySettings is what a library that has never been configured
// reads as — and what an unreadable or half-written pair falls back to, key by
// key.
func DefaultLibrarySettings() LibrarySettings {
	return LibrarySettings{
		ErrorThresholdMP:   DefaultErrorThresholdMP,
		BlunderThresholdMP: DefaultBlunderThresholdMP,
	}
}

// Validate reports why the pair cannot be stored, or nil. The one rule that is
// not a bound is the nesting: every Blunder is an Error, so a blunder
// threshold below the error threshold would name a set that cannot exist.
func (s LibrarySettings) Validate() error {
	if s.ErrorThresholdMP < MinThresholdMP || s.ErrorThresholdMP > MaxThresholdMP {
		return fmt.Errorf("error threshold %d outside [%d, %d]: %w",
			s.ErrorThresholdMP, MinThresholdMP, MaxThresholdMP, ErrInvalid)
	}
	if s.BlunderThresholdMP < MinThresholdMP || s.BlunderThresholdMP > MaxThresholdMP {
		return fmt.Errorf("blunder threshold %d outside [%d, %d]: %w",
			s.BlunderThresholdMP, MinThresholdMP, MaxThresholdMP, ErrInvalid)
	}
	if s.ErrorThresholdMP > s.BlunderThresholdMP {
		return fmt.Errorf("error threshold %d above blunder threshold %d: every blunder is an error: %w",
			s.ErrorThresholdMP, s.BlunderThresholdMP, ErrInvalid)
	}
	return nil
}

// ParseLibrarySettings reads the pair out of the key/value rows a backend
// holds. A missing, unparseable or out-of-range value falls back to its
// default rather than failing: a settings row is a comfort, and a library
// whose row was hand-edited still opens. A pair that survives parsing but
// inverts the nesting is repaired the same way, by defaulting both.
func ParseLibrarySettings(kv map[string]string) LibrarySettings {
	out := DefaultLibrarySettings()
	read := func(key string, dflt int) int {
		v, ok := kv[key]
		if !ok {
			return dflt
		}
		n, err := strconv.Atoi(v)
		if err != nil || n < MinThresholdMP || n > MaxThresholdMP {
			return dflt
		}
		return n
	}
	out.ErrorThresholdMP = read(LibrarySettingErrorKey, DefaultErrorThresholdMP)
	out.BlunderThresholdMP = read(LibrarySettingBlunderKey, DefaultBlunderThresholdMP)
	if out.ErrorThresholdMP > out.BlunderThresholdMP {
		return DefaultLibrarySettings()
	}
	return out
}

// Rows renders the pair as the key/value rows a backend writes.
func (s LibrarySettings) Rows() map[string]string {
	return map[string]string{
		LibrarySettingErrorKey:   strconv.Itoa(s.ErrorThresholdMP),
		LibrarySettingBlunderKey: strconv.Itoa(s.BlunderThresholdMP),
	}
}

// LibrarySettingsStore persists the library's own settings. Every consumer of
// the two thresholds — the statistics, the players' table, the status bar's
// library counter, the study queue an import proposes — reads them here, so
// the word means one thing per library.
type LibrarySettingsStore interface {
	// Load returns the scope's settings, or the defaults when it has none.
	Load(ctx context.Context, scope string) (LibrarySettings, error)

	// Save records the scope's settings. It rejects a pair that fails
	// Validate rather than storing something no set could match.
	Save(ctx context.Context, scope string, settings LibrarySettings) error
}

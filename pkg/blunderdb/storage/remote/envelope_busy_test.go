package remote

import (
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// TestEnvelopeError_OnlyBusyIsErrBusy: an unavailable answer is a lost write
// to retry as is only when the daemon says so; a daemon stopping or refusing
// a library-wide read stays ErrUnavailable.
func TestEnvelopeError_OnlyBusyIsErrBusy(t *testing.T) {
	busy := envelopeError("stats.compute", "unavailable", "the database is busy", map[string]any{"reason": "busy"})
	if !errors.Is(busy, storage.ErrBusy) {
		t.Errorf("busy reason: %v, want storage.ErrBusy", busy)
	}
	other := envelopeError("stats.compute", "unavailable", "too many library-wide reads in flight", nil)
	if errors.Is(other, storage.ErrBusy) || !errors.Is(other, ErrUnavailable) {
		t.Errorf("no reason: %v, want ErrUnavailable and not storage.ErrBusy", other)
	}
}

package sqlshared

import (
	"errors"
	"fmt"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type referencingDialect struct{ fakeDialect }

func (referencingDialect) Referenced(err error) error {
	return fmt.Errorf("%w: %w", storage.ErrNotFound, err)
}

// A foreign-key refusal that slips between checkTargets and the write is a bad
// request (ErrInvalid), not a raw server error; any other error passes through.
func TestStepTargetGone(t *testing.T) {
	if err := stepTargetGone(referencingDialect{}, errors.New("fk")); !errors.Is(err, storage.ErrInvalid) || errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("FK violation = %v; want ErrInvalid only", err)
	}
	other := errors.New("disk full")
	if err := stepTargetGone(fakeDialect{}, other); err != other {
		t.Fatalf("other error = %v; want it unchanged", err)
	}
}

package storage

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// MinReaderVersionKey is the metadata row recording the oldest schema version
// a binary must speak to open the database. Every backend checks it before
// touching the schema and raises it to domain.MinReaderVersion once open for
// writing, so the check is the same in the GUI, the CLI, serve and migrate.
const MinReaderVersionKey = "min_reader_version"

// ErrDatabaseTooNew — the database demands a newer blunderDB than this one.
// It is a refusal before any write, never a partial open.
var ErrDatabaseTooNew = errors.New("this database requires a newer blunderDB")

// CheckReaderFloor refuses a database whose recorded floor is newer than the
// schema this binary speaks. An absent floor ("") predates the marker and is
// accepted; an unparsable one is refused, since nothing proves it is safe.
func CheckReaderFloor(floor string) error {
	if floor == "" {
		return nil
	}
	cmp, err := CompareSchemaVersions(floor, domain.DatabaseVersion)
	if err != nil {
		return fmt.Errorf("%s %q: %w", MinReaderVersionKey, floor, err)
	}
	if cmp > 0 {
		return fmt.Errorf("%w: it needs a blunderDB that reads schema %s or later, this one reads %s; update blunderDB to open it",
			ErrDatabaseTooNew, floor, domain.DatabaseVersion)
	}
	return nil
}

// RaisedReaderFloor is the floor to record after opening for writing: the
// larger of the stored one and domain.MinReaderVersion, so a floor a newer
// binary set is never lowered.
func RaisedReaderFloor(stored string) string {
	if stored == "" {
		return domain.MinReaderVersion
	}
	if cmp, err := CompareSchemaVersions(stored, domain.MinReaderVersion); err == nil && cmp > 0 {
		return stored
	}
	return domain.MinReaderVersion
}

// ParseSchemaVersion splits a schema version "MAJOR.MINOR.PATCH" into its
// three numeric components.
func ParseSchemaVersion(v string) ([3]int, error) {
	var out [3]int
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return out, fmt.Errorf("want MAJOR.MINOR.PATCH, got %q", v)
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return out, fmt.Errorf("component %q of %q is not a non-negative integer", p, v)
		}
		out[i] = n
	}
	return out, nil
}

// CompareSchemaVersions orders two schema versions component by component, so
// that 2.10.0 sorts after 2.9.0 — a plain string comparison puts it before. It
// returns -1, 0 or +1 as a is older than, equal to or newer than b.
func CompareSchemaVersions(a, b string) (int, error) {
	va, err := ParseSchemaVersion(a)
	if err != nil {
		return 0, err
	}
	vb, err := ParseSchemaVersion(b)
	if err != nil {
		return 0, err
	}
	for i := range va {
		switch {
		case va[i] < vb[i]:
			return -1, nil
		case va[i] > vb[i]:
			return 1, nil
		}
	}
	return 0, nil
}

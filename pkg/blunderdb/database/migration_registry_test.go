package database

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

// TestMigrationSteps_ContinuousChain guards the shape of the registry: one
// unbroken chain from 1.0.0 to DatabaseVersion, each step strictly newer than
// the one before, no version reached twice. A gap here is what a forgotten
// registry line looks like after a DatabaseVersion bump.
func TestMigrationSteps_ContinuousChain(t *testing.T) {
	if len(migrationSteps) == 0 {
		t.Fatal("migrationSteps is empty")
	}
	if got := migrationSteps[0].from; got != "1.0.0" {
		t.Errorf("chain starts at %s, want 1.0.0", got)
	}
	if got := migrationSteps[len(migrationSteps)-1].to; got != DatabaseVersion {
		t.Errorf("chain ends at %s, want DatabaseVersion %s", got, DatabaseVersion)
	}

	seenFrom := map[string]bool{}
	for i, s := range migrationSteps {
		if s.run == nil {
			t.Errorf("step %s→%s has no run function", s.from, s.to)
		}
		if seenFrom[s.from] {
			t.Errorf("step %d: version %s has two migration steps", i, s.from)
		}
		seenFrom[s.from] = true

		if c, err := compareVersions(s.from, s.to); err != nil {
			t.Errorf("step %d (%s→%s): %v", i, s.from, s.to, err)
		} else if c >= 0 {
			t.Errorf("step %d: %s→%s does not go forward", i, s.from, s.to)
		}
		if i > 0 && migrationSteps[i-1].to != s.from {
			t.Errorf("gap in chain: step %d ends at %s, step %d starts at %s",
				i-1, migrationSteps[i-1].to, i, s.from)
		}
	}

	// Walking the registry from 1.0.0 must reach DatabaseVersion.
	v := "1.0.0"
	for range migrationSteps {
		s, ok := findMigrationStep(v)
		if !ok {
			t.Fatalf("no step from %s", v)
		}
		v = s.to
	}
	if v != DatabaseVersion {
		t.Errorf("walking the registry ends at %s, want %s", v, DatabaseVersion)
	}
}

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "1.1.0", -1},
		{"1.1.0", "1.0.0", 1},
		{"2.0.0", "1.9.0", 1},
		// The cases a string comparison gets wrong.
		{"2.10.0", "2.9.0", 1},
		{"2.9.0", "2.10.0", -1},
		{"1.10.0", "1.4.0", 1},
		{"2.15.0", "2.2.0", 1},
		{"10.0.0", "9.0.0", 1},
	}
	for _, c := range cases {
		got, err := compareVersions(c.a, c.b)
		if err != nil {
			t.Errorf("compareVersions(%q, %q): %v", c.a, c.b, err)
			continue
		}
		if got != c.want {
			t.Errorf("compareVersions(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}

	for _, bad := range []string{"", "2", "2.15", "2.15.0.1", "a.b.c", "2.-1.0", "2.15.0-rc1"} {
		if _, err := parseVersion(bad); err == nil {
			t.Errorf("parseVersion(%q) accepted a malformed version", bad)
		}
	}

	if !versionAtLeast("2.10.0", "2.9.0") {
		t.Error("versionAtLeast(2.10.0, 2.9.0) = false")
	}
	if versionAtLeast("garbage", "1.0.0") {
		t.Error("versionAtLeast(garbage, 1.0.0) = true")
	}
}

// stampVersion overwrites database_version on a freshly created database so
// the chain can be exercised from an arbitrary starting point.
func stampVersion(t *testing.T, path, version string) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`UPDATE metadata SET value = ? WHERE key = 'database_version'`, version); err != nil {
		t.Fatal(err)
	}
}

// TestRunMigrationChain_UnknownVersion: a version this build has no step for
// is refused when it is older than DatabaseVersion (nothing could bring it
// forward) and opened as is when it is newer (the schema only grows).
func TestRunMigrationChain_UnknownVersion(t *testing.T) {
	newDB := func(t *testing.T, version string) string {
		t.Helper()
		path := filepath.Join(t.TempDir(), "v.db")
		d := NewDatabase()
		if err := d.SetupDatabase(path); err != nil {
			t.Fatal(err)
		}
		_ = d.Close()
		stampVersion(t, path, version)
		return path
	}

	t.Run("older_without_step", func(t *testing.T) {
		path := newDB(t, "0.9.0")
		d := NewDatabase()
		err := d.OpenDatabase(path)
		if err == nil {
			_ = d.Close()
			t.Fatal("OpenDatabase accepted a version no step starts from")
		}
	})

	t.Run("malformed", func(t *testing.T) {
		path := newDB(t, "two.point.oh")
		d := NewDatabase()
		if err := d.OpenDatabase(path); err == nil {
			_ = d.Close()
			t.Fatal("OpenDatabase accepted a malformed version")
		}
	})

	t.Run("newer", func(t *testing.T) {
		path := newDB(t, "99.0.0")
		d := &Database{}
		db, err := sql.Open("sqlite", path)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		d.db = db
		if err := d.runMigrationChain(context.Background()); err != nil {
			t.Fatalf("a newer database must open unchanged: %v", err)
		}
		var v string
		if err := db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&v); err != nil {
			t.Fatal(err)
		}
		if v != "99.0.0" {
			t.Errorf("version rewritten to %s", v)
		}
	})
}

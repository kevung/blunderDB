package database

import (
	"maps"
	"testing"
)

// LoadMetadata reports the header keys only, SaveMetadata replaces what is
// stored, and a header key stored without a value reads as "" instead of
// failing the whole read (info, search headers).
func TestMetadata_HeaderKeys(t *testing.T) {
	t.Parallel()
	d := NewDatabase()
	if err := d.SetupDatabase(":memory:"); err != nil {
		t.Fatal(err)
	}
	closeOnCleanup(t, d)

	if err := d.SaveMetadata(map[string]string{"user": "Alice", "description": "first", "other_key": "x"}); err != nil {
		t.Fatalf("SaveMetadata: %v", err)
	}
	if err := d.SaveMetadata(map[string]string{"description": "second"}); err != nil {
		t.Fatalf("SaveMetadata: %v", err)
	}
	if _, err := d.db.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES ('dateOfCreation', NULL)`); err != nil {
		t.Fatal(err)
	}

	got, err := d.LoadMetadata()
	if err != nil {
		t.Fatalf("LoadMetadata: %v", err)
	}
	want := map[string]string{
		"user": "Alice", "description": "second", "dateOfCreation": "",
		"database_version": DatabaseVersion,
	}
	if !maps.Equal(got, want) {
		t.Errorf("LoadMetadata = %v, want %v", got, want)
	}
}

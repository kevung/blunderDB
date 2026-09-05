package bearoffgen

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestVerify_TheShippedTablesAreVerified(t *testing.T) {
	for _, tc := range []struct {
		path string
		want Domain
	}{
		{"../race/gnubg_ts0.bd", Domain{Kind: TwoSidedKind, Points: 6, Checkers: 6}},
		{"../gnubg_os6.bd", Domain{Kind: OneSidedKind, Points: 6, Checkers: 15}},
	} {
		if _, err := os.Stat(tc.path); err != nil {
			t.Skipf("reference table not available: %v", err)
		}
		verdict, domain, err := Verify(tc.path)
		if err != nil {
			t.Fatalf("Verify(%s): %v", tc.path, err)
		}
		if domain != tc.want {
			t.Errorf("Verify(%s) read domain %v, want %v", tc.path, domain, tc.want)
		}
		if verdict != Verified {
			t.Errorf("Verify(%s) = %s, want verified", tc.path, verdict)
		}
	}
}

// What the generator writes must verify: this is the loop the Bearoff tab
// closes after a run.
func TestVerify_AFreshlyGeneratedTableVerifies(t *testing.T) {
	dir := t.TempDir()
	domain := Domain{Kind: TwoSidedKind, Points: 6, Checkers: 6}
	path := filepath.Join(dir, domain.FileName())

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := TwoSided(context.Background(), f, domain.Points, domain.Checkers, nil); err != nil {
		t.Fatal(err)
	}
	f.Close()

	verdict, got, err := Verify(path)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if got != domain {
		t.Errorf("domain = %v, want %v", got, domain)
	}
	if verdict != Verified {
		t.Errorf("a table we just generated must verify, got %s", verdict)
	}
}

func TestVerify_ATruncatedFileIsCorrupt(t *testing.T) {
	dir := t.TempDir()
	full, err := os.ReadFile("../race/gnubg_ts0.bd")
	if err != nil {
		t.Skipf("reference table not available: %v", err)
	}
	path := filepath.Join(dir, "short.bd")
	if err := os.WriteFile(path, full[:len(full)-2], 0o644); err != nil {
		t.Fatal(err)
	}
	if verdict, _, err := Verify(path); verdict != Corrupt || err == nil {
		t.Errorf("a truncated table must be corrupt, got %s (err %v)", verdict, err)
	}
}

func TestVerify_AFlippedByteIsCorrupt(t *testing.T) {
	dir := t.TempDir()
	full, err := os.ReadFile("../race/gnubg_ts0.bd")
	if err != nil {
		t.Skipf("reference table not available: %v", err)
	}
	altered := bytes.Clone(full)
	altered[len(altered)/2] ^= 0x01
	path := filepath.Join(dir, "flipped.bd")
	if err := os.WriteFile(path, altered, 0o644); err != nil {
		t.Fatal(err)
	}
	verdict, _, err := Verify(path)
	if verdict != Corrupt || err == nil {
		t.Errorf("one flipped bit must be caught, got %s (err %v)", verdict, err)
	}
}

// A domain nobody has a reference for is usable, and says so — it is not an
// error, and not a promise either.
func TestVerify_AnUnknownDomainIsUnverified(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gnubg_ts3x3.bd")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := TwoSided(context.Background(), f, 3, 3, nil); err != nil {
		t.Fatal(err)
	}
	f.Close()

	verdict, domain, err := Verify(path)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if verdict != Unverified {
		t.Errorf("TS-03-03 has no recorded fingerprint; got %s", verdict)
	}
	if domain.String() != "TS-03-03" {
		t.Errorf("domain = %s", domain)
	}
}

func TestDomain_NamesAndSizes(t *testing.T) {
	ts := Domain{Kind: TwoSidedKind, Points: 6, Checkers: 11}
	if ts.String() != "TS-06-11" || ts.FileName() != "gnubg_ts6x11.bd" {
		t.Errorf("two-sided: %s / %s", ts, ts.FileName())
	}
	// The size ADR-0027 records for TS-06-11.
	if got := ts.Size(); got != 40+int64(12376)*12376*8 {
		t.Errorf("TS-06-11 size = %d", got)
	}
	os6 := Domain{Kind: OneSidedKind, Points: 6, Checkers: 15}
	if os6.String() != "OS-06" || os6.FileName() != "gnubg_os6.bd" {
		t.Errorf("one-sided: %s / %s", os6, os6.FileName())
	}
	if os6.Size() != 0 {
		t.Error("a one-sided size is only known after generation")
	}
}

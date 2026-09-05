package bearoffgen

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestVerify_TheShippedTablesAreVerified(t *testing.T) {
	for _, tc := range []struct {
		path string
		want Domain
	}{
		{"testdata/gnubg_ts0.bd", Domain{Kind: TwoSidedKind, Points: 6, Checkers: 6}},
		{"testdata/gnubg_os6.bd", Domain{Kind: OneSidedKind, Points: 6, Checkers: 15}},
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
	full, err := os.ReadFile("testdata/gnubg_ts0.bd")
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
	full, err := os.ReadFile("testdata/gnubg_ts0.bd")
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
	// A one-sided table is compressed, so its size is a measurement and not
	// arithmetic: it must be the size of the file gnubg actually produced,
	// which is the fixture in testdata.
	fixture, err := os.Stat("testdata/gnubg_os6.bd")
	if err != nil {
		t.Fatal(err)
	}
	if os6.Size() != fixture.Size() {
		t.Errorf("OS-06 size = %d, want the fixture's %d", os6.Size(), fixture.Size())
	}

	// And the one-sided candidates must be priced and ordered like the
	// two-sided ones: no zero, monotone in the width.
	var lastSize int64
	var lastCost time.Duration
	for _, d := range OneSidedCandidates() {
		if d.Size() <= lastSize {
			t.Errorf("%s: size %d does not exceed the previous domain's %d", d, d.Size(), lastSize)
		}
		if d.RAMNeeded() < d.Size() {
			t.Errorf("%s: RAM needed %d is below the table itself (%d)", d, d.RAMNeeded(), d.Size())
		}
		cost := d.EstimateDuration(0, 8)
		if cost <= lastCost {
			t.Errorf("%s: estimate %v does not exceed the previous domain's %v", d, cost, lastCost)
		}
		lastSize, lastCost = d.Size(), cost
	}
}

// The sizes the interface shows must be the file's real size, and the cost
// model must be monotone in the chequer count — a wider table is never
// cheaper. TS-06-06 and TS-06-11 have known sizes: they are the two tables
// blunderDB shipped and downloaded until ADR-0027.
func TestEstimate_SizesAndCostsGrowWithTheDomain(t *testing.T) {
	t.Parallel()
	// TS-06-06's size is checked against the file gnubg actually produced,
	// not against a number typed here.
	fixture, err := os.Stat("testdata/gnubg_ts0.bd")
	if err != nil {
		t.Fatal(err)
	}
	known := map[int]int64{6: fixture.Size()}
	var prevSize int64
	var prevCost time.Duration
	for _, d := range Candidates() {
		if want, ok := known[d.Checkers]; ok && d.Size() != want {
			t.Errorf("%s: Size = %d, want %d", d, d.Size(), want)
		}
		if d.Size() <= prevSize {
			t.Errorf("%s: size %d does not exceed the previous domain's %d", d, d.Size(), prevSize)
		}
		if d.RAMNeeded() < d.Size() {
			t.Errorf("%s: RAM needed %d is below the table itself (%d)", d, d.RAMNeeded(), d.Size())
		}
		cost := d.EstimateDuration(0, 8)
		if cost <= prevCost {
			t.Errorf("%s: estimate %v does not exceed the previous domain's %v", d, cost, prevCost)
		}
		prevSize, prevCost = d.Size(), cost
	}
}

// The estimate has to fall as cores are added, and a measured run has to feed
// the next estimate.
func TestEstimate_CoresAndMeasurement(t *testing.T) {
	t.Parallel()
	d := Domain{Kind: TwoSidedKind, Points: 6, Checkers: 9}
	one := d.EstimateDuration(0, 1)
	eight := d.EstimateDuration(0, 8)
	if eight >= one {
		t.Errorf("eight cores (%v) are not faster than one (%v)", eight, one)
	}

	// The reference machine measured 78.9 s serially on TS-06-09.
	rate := d.MeasuredRate(78900*time.Millisecond, 1)
	if rate < 5e-10 || rate > 8e-10 {
		t.Errorf("MeasuredRate = %g, expected the reference machine's 6.3e-10", rate)
	}
	// A machine twice as slow should predict about twice the time.
	slow := d.EstimateDuration(2*rate, 1)
	fast := d.EstimateDuration(rate, 1)
	if slow < 2*fast-time.Second || slow > 2*fast+time.Second {
		t.Errorf("doubling the rate gave %v against %v", slow, fast)
	}
	// TS-06-06 is too small to say anything about the sweep's rate.
	if r := (Domain{Kind: TwoSidedKind, Points: 6, Checkers: 6}).MeasuredRate(time.Second, 1); r != 0 {
		t.Errorf("TS-06-06 reported a rate of %g", r)
	}
}

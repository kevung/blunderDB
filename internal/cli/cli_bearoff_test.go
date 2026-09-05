package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine/bearoffgen"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/bearoffgen/bearofftest"
)

func TestParseDomain(t *testing.T) {
	t.Parallel()
	ok := map[string]bearoffgen.Domain{
		"6x6":       {Kind: bearoffgen.TwoSidedKind, Points: 6, Checkers: 6},
		"6X11":      {Kind: bearoffgen.TwoSidedKind, Points: 6, Checkers: 11},
		" 6x15 ":    {Kind: bearoffgen.TwoSidedKind, Points: 6, Checkers: 15},
		"os":        {Kind: bearoffgen.OneSidedKind, Points: 6, Checkers: 15},
		"one-sided": {Kind: bearoffgen.OneSidedKind, Points: 6, Checkers: 15},
	}
	for spec, want := range ok {
		got, err := parseDomain(spec)
		if err != nil {
			t.Errorf("parseDomain(%q): %v", spec, err)
			continue
		}
		if got != want {
			t.Errorf("parseDomain(%q) = %v, want %v", spec, got, want)
		}
	}
	// A bearoff table describes the six-point home board, and the chequer
	// count runs from 1 to 15: everything else is a typo the user should be
	// told about rather than a run that fails an hour later.
	for _, bad := range []string{"", "6", "8x6", "6x0", "6x16", "6xy", "zx6"} {
		if _, err := parseDomain(bad); err == nil {
			t.Errorf("parseDomain(%q) accepted a domain that does not exist", bad)
		}
	}
}

// `bearoff list` must price every domain and say which are already there,
// without touching a database — a bearoff table is arithmetic about the game,
// not about anyone's positions.
func TestCLI_BearoffList(t *testing.T) {
	dir := t.TempDir()
	src, err := os.ReadFile(bearofftest.TwoSidedPath(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "gnubg_ts6x6.bd"), src, 0o644); err != nil {
		t.Fatal(err)
	}

	out := captureStdout(t, func() {
		if err := NewCLI().Run([]string{"bearoff", "list", "--data-dir", dir, "--cores", "4"}); err != nil {
			t.Fatal(err)
		}
	})
	for _, want := range []string{"TS-06-06", "TS-06-15", "OS-06", "verified", dir} {
		if !strings.Contains(out, want) {
			t.Errorf("bearoff list does not mention %q:\n%s", want, out)
		}
	}

	out = captureStdout(t, func() {
		if err := NewCLI().Run([]string{"bearoff", "list", "--data-dir", dir, "--format", "json"}); err != nil {
			t.Fatal(err)
		}
	})
	for _, want := range []string{`"domain": "TS-06-06"`, `"verdict": "verified"`, `"ram_needed"`, `"estimate_seconds"`} {
		if !strings.Contains(out, want) {
			t.Errorf("bearoff list --format json does not carry %q:\n%s", want, out)
		}
	}
}

// Verify is meant to be put in a script: a corrupt file must be an error the
// shell sees, not a line of text with exit code 0.
func TestCLI_BearoffVerify(t *testing.T) {
	dir := t.TempDir()
	src, err := os.ReadFile(bearofftest.TwoSidedPath(t))
	if err != nil {
		t.Fatal(err)
	}
	good := filepath.Join(dir, "gnubg_ts6x6.bd")
	if err := os.WriteFile(good, src, 0o644); err != nil {
		t.Fatal(err)
	}
	out := captureStdout(t, func() {
		if err := NewCLI().Run([]string{"bearoff", "verify", good}); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(out, "verified") {
		t.Errorf("a genuine table is not reported verified:\n%s", out)
	}

	bad := filepath.Join(dir, "broken.bd")
	src[len(src)/2] ^= 0xFF
	if err := os.WriteFile(bad, src, 0o644); err != nil {
		t.Fatal(err)
	}
	captureStdout(t, func() {
		if err := NewCLI().Run([]string{"bearoff", "verify", bad}); err == nil {
			t.Error("a corrupt table exited 0")
		}
	})
}

// Generate is resumable and idempotent: asked for a table that is already
// there, it says so instead of spending the time again.
func TestCLI_BearoffGenerateAndDelete(t *testing.T) {
	dir := t.TempDir()
	out := captureStdout(t, func() {
		if err := NewCLI().Run([]string{"bearoff", "generate", "--ts", "6x3", "--data-dir", dir, "--quiet"}); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(out, "Wrote") {
		t.Errorf("generate said nothing about what it wrote:\n%s", out)
	}
	path := filepath.Join(dir, "gnubg_ts6x3.bd")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("the table was not written: %v", err)
	}

	out = captureStdout(t, func() {
		if err := NewCLI().Run([]string{"bearoff", "generate", "--ts", "6x3", "--data-dir", dir, "--quiet"}); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(out, "already") {
		t.Errorf("generate recomputed a table that was already there:\n%s", out)
	}

	out = captureStdout(t, func() {
		if err := NewCLI().Run([]string{"bearoff", "delete", "--ts", "6x3", "--data-dir", dir}); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(out, "Removed") {
		t.Errorf("delete said nothing:\n%s", out)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("the table survived delete")
	}
}

func TestCLI_BearoffRejectsUnknownSubcommand(t *testing.T) {
	captureStdout(t, func() {
		if err := NewCLI().Run([]string{"bearoff", "frobnicate"}); err == nil {
			t.Error("an unknown sub-command was accepted")
		}
		if err := NewCLI().Run([]string{"bearoff"}); err == nil {
			t.Error("bearoff with no sub-command was accepted")
		}
		if err := NewCLI().Run([]string{"bearoff", "generate", "--ts", "9x9"}); err == nil {
			t.Error("a domain that does not exist was accepted")
		}
	})
}

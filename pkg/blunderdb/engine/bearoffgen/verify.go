package bearoffgen

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// ParseHeader reads a bearoff file's 40-byte header and returns the domain it
// declares. gnubg writes "gnubg-TS-06-06-1xxx…\n" for a two-sided table and
// "gnubg-OS-06-15-…" for a one-sided one.
func ParseHeader(header []byte) (Domain, error) {
	if len(header) < 40 {
		return Domain{}, fmt.Errorf("bearoffgen: header is %d bytes, want 40", len(header))
	}
	s := string(header[:40])
	if !strings.HasPrefix(s, "gnubg-") {
		return Domain{}, fmt.Errorf("bearoffgen: not a gnubg bearoff file (header %q)", firstLine(s))
	}
	fields := strings.Split(s[6:], "-")
	if len(fields) < 3 {
		return Domain{}, fmt.Errorf("bearoffgen: malformed header %q", firstLine(s))
	}
	points, err := strconv.Atoi(fields[1])
	if err != nil {
		return Domain{}, fmt.Errorf("bearoffgen: header point count %q: %w", fields[1], err)
	}
	checkers, err := strconv.Atoi(fields[2])
	if err != nil {
		return Domain{}, fmt.Errorf("bearoffgen: header chequer count %q: %w", fields[2], err)
	}
	switch fields[0] {
	case "TS":
		return Domain{Kind: TwoSidedKind, Points: points, Checkers: checkers}, nil
	case "OS":
		return Domain{Kind: OneSidedKind, Points: points, Checkers: checkers}, nil
	default:
		return Domain{}, fmt.Errorf("bearoffgen: unknown table kind %q", fields[0])
	}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// Verify reads the file at path and reports whether it is the table gnubg would
// produce for the domain its header declares.
//
// Corrupt deletes nothing: the caller decides what to do with a file the user
// may have spent hours generating. It only ever means "this file contradicts
// itself" — a size that does not match its own header, or a hash that differs
// from the recorded one for its domain.
func Verify(path string) (Verdict, Domain, error) {
	f, err := os.Open(path)
	if err != nil {
		return Corrupt, Domain{}, fmt.Errorf("bearoffgen: open %s: %w", path, err)
	}
	defer f.Close()

	header := make([]byte, 40)
	if _, err := io.ReadFull(f, header); err != nil {
		return Corrupt, Domain{}, fmt.Errorf("bearoffgen: read header of %s: %w", path, err)
	}
	domain, err := ParseHeader(header)
	if err != nil {
		return Corrupt, Domain{}, err
	}

	info, err := f.Stat()
	if err != nil {
		return Corrupt, domain, fmt.Errorf("bearoffgen: stat %s: %w", path, err)
	}
	// Only the two-sided size is known from the domain alone; a one-sided
	// table's data section depends on what compressed.
	if want := domain.Size(); want > 0 && info.Size() != want {
		return Corrupt, domain, fmt.Errorf("bearoffgen: %s declares %s (%d bytes) but is %d bytes",
			path, domain, want, info.Size())
	}

	want, known := KnownFingerprints[domain]
	if !known {
		return Unverified, domain, nil
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return Corrupt, domain, err
	}
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return Corrupt, domain, fmt.Errorf("bearoffgen: hash %s: %w", path, err)
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != want {
		return Corrupt, domain, fmt.Errorf("bearoffgen: %s hashes to %s, gnubg's %s hashes to %s",
			path, got, domain, want)
	}
	return Verified, domain, nil
}

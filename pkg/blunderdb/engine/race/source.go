package race

import (
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Source resolution (ADR-0009, amended by ADR-0027): the candidates are an
// optional user-supplied .bd path and every generated table in the data
// directory. The widest valid domain wins; an invalid candidate is skipped
// with a log warning, never a fatal error.
//
// There is no floor any more. The TS-06-06 table used to be compiled into the
// binary, so Resolve could promise a database; since ADR-0027 the tables are
// generated on the machine that needs them, and there is a window — the first
// launch, until the background generation finishes — where there is none.
// Resolve returns nil for that, and callers say "not yet" rather than
// pretending to an exact answer they do not have.
//
// A `.part` file is never a candidate: it is an interrupted run, not a table.

// GeneratedGlob matches the two-sided tables the generator writes into the
// data directory: gnubg_ts6x6.bd, gnubg_ts6x11.bd, and any wider domain the
// user asked for.
const GeneratedGlob = "gnubg_ts*.bd"

var (
	srcMu       sync.Mutex
	externalP   string
	dataDirP    string
	cachedDB    *TwoSided
	cachedStamp string
)

// DefaultDataDir returns $XDG_DATA_HOME/blunderdb (or ~/.local/share/blunderdb).
// Deliberately the data dir, not the cache dir: cache cleaners must not cost
// the user a 1.2 GB re-download.
func DefaultDataDir() string {
	if x := os.Getenv("XDG_DATA_HOME"); x != "" {
		return filepath.Join(x, "blunderdb")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".local", "share", "blunderdb")
}

// SetExternalPath configures the user-supplied .bd path ("" to unset).
func SetExternalPath(p string) {
	srcMu.Lock()
	defer srcMu.Unlock()
	externalP = p
	cachedStamp = "" // force re-resolution
}

// SetDataDir overrides the download directory (tests; "" restores default).
func SetDataDir(d string) {
	srcMu.Lock()
	defer srcMu.Unlock()
	dataDirP = d
	cachedStamp = ""
}

// DataDir returns the effective download directory.
func DataDir() string {
	srcMu.Lock()
	defer srcMu.Unlock()
	return dataDirLocked()
}

func dataDirLocked() string {
	if dataDirP != "" {
		return dataDirP
	}
	return DefaultDataDir()
}

// GeneratedPath returns where a generated table of the given name lives (it
// may not exist).
func GeneratedPath(name string) string {
	return filepath.Join(DataDir(), name)
}

// stamp fingerprints the candidate set so Resolve can cache the open handle
// and still notice config changes, downloads completing, or file swaps.
func stamp(paths []string) string {
	s := ""
	for _, p := range paths {
		s += p + "|"
		if st, err := os.Stat(p); err == nil {
			s += st.ModTime().String() + "|" + itoa64(st.Size())
		}
		s += ";"
	}
	return s
}

func itoa64(n int64) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// Invalidate drops the cached reader so the next Resolve re-probes the
// candidates from scratch. Call it before deleting or replacing the
// downloaded or external file: on Windows a file held open by the cache
// cannot be removed, and its stale directory entry would otherwise be what
// the next Resolve finds. The embedded database is never closed.
func Invalidate() {
	srcMu.Lock()
	defer srcMu.Unlock()
	if cachedDB != nil {
		cachedDB.Close()
	}
	cachedDB, cachedStamp = nil, ""
}

// Resolve returns the widest available two-sided database, or nil when the
// machine has none yet. Callers must handle nil: it is the first launch, or a
// data directory somebody emptied, and the honest answer is that the exact
// regime is unavailable rather than a guess dressed as a lookup.
func Resolve() *TwoSided {
	srcMu.Lock()
	defer srcMu.Unlock()

	candidates := []string{}
	if externalP != "" {
		candidates = append(candidates, externalP)
	}
	generated, _ := filepath.Glob(filepath.Join(dataDirLocked(), GeneratedGlob))
	sort.Strings(generated) // a stable candidate order makes the stamp stable
	candidates = append(candidates, generated...)

	st := stamp(candidates)
	if cachedDB != nil && st == cachedStamp {
		return cachedDB
	}

	// The stamp changed: whatever cachedDB currently holds open is about to be
	// superseded. Close it *before* probing the candidates below — on Windows a
	// file the caller just replaced or removed keeps its old directory entry
	// alive (and refuses a fresh open) for as long as this stale handle stays
	// open, which would make the loop below see the outgoing file.
	if cachedDB != nil {
		cachedDB.Close()
	}
	cachedDB = nil

	var best *TwoSided
	for _, p := range candidates {
		if strings.HasSuffix(p, ".part") {
			continue // an interrupted generation, not a table
		}
		if _, err := os.Stat(p); err != nil {
			continue
		}
		ts, err := OpenTwoSided(p)
		if err != nil {
			slog.Warn("ignoring invalid two-sided bearoff database", "path", p, "err", err)
			continue
		}
		if best == nil || ts.Checkers() > best.Checkers() {
			if best != nil {
				best.Close()
			}
			best = ts
			continue
		}
		ts.Close()
	}
	cachedDB, cachedStamp = best, st
	return best
}

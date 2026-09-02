package race

import (
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

// Source resolution (ADR-0009): three candidates feed one reader — the
// embedded TS-06-06 (always available), an optional user-supplied .bd path,
// and the optionally downloaded TS-06-11. The widest valid domain wins; an
// invalid candidate is skipped with a log warning, never a fatal error.
//
// The daemon never downloads: it only ever sees the embedded database plus
// whatever path its operator configures (volume mount + SetExternalPath).

// DownloadedFileName is the fixed name of the optionally downloaded TS-06-11
// database inside DataDir (published as the bearoff-data-1 release asset).
const DownloadedFileName = "gnubg_ts6x11.bd"

// DownloadedSHA256 is the checksum of the published bearoff-data-1 asset.
const DownloadedSHA256 = "c52133cd59a7db478a71d18c8f2093ba343200fa72ede8004c32c6778c724f46"

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

// DownloadedPath returns where the downloaded TS-06-11 lives (may not exist).
func DownloadedPath() string {
	return filepath.Join(DataDir(), DownloadedFileName)
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

// Resolve returns the widest available two-sided database. It never returns
// nil: the embedded TS-06-06 is the floor.
func Resolve() *TwoSided {
	srcMu.Lock()
	defer srcMu.Unlock()

	candidates := []string{}
	if externalP != "" {
		candidates = append(candidates, externalP)
	}
	candidates = append(candidates, filepath.Join(dataDirLocked(), DownloadedFileName))

	st := stamp(candidates)
	if cachedDB != nil && st == cachedStamp {
		return cachedDB
	}

	// The stamp changed: whatever cachedDB currently holds open (an external
	// or downloaded file) is about to be superseded. Close it *before*
	// probing the candidates below — on Windows a file the caller just
	// replaced or removed keeps its old directory entry alive (and refuses a
	// fresh open) for as long as this stale handle stays open, which would
	// make the stat/open loop below see the outgoing file instead of the
	// caller's change.
	embedded := EmbeddedTwoSided()
	if cachedDB != nil && cachedDB != embedded {
		cachedDB.Close()
	}
	cachedDB = nil

	best := embedded
	for _, p := range candidates {
		if _, err := os.Stat(p); err != nil {
			continue
		}
		ts, err := OpenTwoSided(p)
		if err != nil {
			slog.Warn("ignoring invalid two-sided bearoff database", "path", p, "err", err)
			continue
		}
		if ts.Checkers() > best.Checkers() {
			if best != embedded {
				best.Close()
			}
			best = ts
		} else {
			ts.Close()
		}
	}
	cachedDB, cachedStamp = best, st
	return best
}

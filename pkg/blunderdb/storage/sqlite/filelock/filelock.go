// Package filelock holds the advisory locks every opener of a SQLite library
// file shares, whichever mode opened it: the single-writer lock (ADR-0004) and
// the swap gate that orders opening a file against a vacuum replacing it.
// They live below both the desktop wrapper and the storage backend so that
// the GUI, the CLI, the serve daemon and the migration name the same lock
// files for the same database.
package filelock

import (
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
)

// LockPath returns the file whose advisory lock guards a database against a
// second writer. It lives in the cache directory: a lock file beside the
// database reads as debris yet can never be unlinked on release (another
// instance may hold its inode, letting a third lock a fresh file). Named from
// the canonical path, so two spellings of one file name one lock; falls back
// beside the database if the cache directory cannot be created (ADR-0004).
func LockPath(dbPath string) string {
	dir := filepath.Join(xdg.CacheHome, "blunderDB", "locks")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		slog.Debug("lock directory unavailable; locking beside the database", "err", err)
		return dbPath + ".lock"
	}
	sum := sha256.Sum256([]byte(canonical(dbPath)))
	return filepath.Join(dir, hex.EncodeToString(sum[:16])+".lock")
}

// canonical spells dbPath as SQLite reports it once open (PRAGMA
// database_list resolves symbolic links): an opener holds the gate under the
// path it was given, the vacuum takes it under the one the connection
// reports, and both must land on one lock file — on macOS the temporary
// directory itself sits behind a symbolic link. A file not created yet is
// resolved through its directory.
func canonical(p string) string {
	if abs, err := filepath.Abs(p); err == nil {
		p = abs
	}
	if real, err := filepath.EvalSymlinks(p); err == nil {
		return filepath.Clean(real)
	}
	if dir, err := filepath.EvalSymlinks(filepath.Dir(p)); err == nil {
		return filepath.Join(dir, filepath.Base(p))
	}
	return filepath.Clean(p)
}

// SwapGatePath returns the file whose lock orders opening a database against
// replacing it (a vacuum by swap): an opener holds it shared until its
// connection is established, the swap exclusively from the check that no
// connection exists to the rename. It lives beside the single-writer lock,
// for the same reasons.
func SwapGatePath(dbPath string) string {
	return strings.TrimSuffix(LockPath(dbPath), ".lock") + ".swap"
}

// EnterSwapGate waits until no file replacement of path is under way and
// keeps one from starting until the returned release runs. A gate that
// cannot be taken is no reason to refuse the open: it proceeds unguarded.
// The wait is unbounded: a third party that keeps the gate exclusively keeps
// the opener blocked. :memory: and the empty path have no file to replace.
func EnterSwapGate(path string) func() {
	if path == "" || path == ":memory:" {
		return func() {}
	}
	gate, err := Shared(SwapGatePath(path))
	if err != nil {
		slog.Warn("swap gate unavailable; opening without it", "path", path, "err", err)
		return func() {}
	}
	return func() { _ = gate.Release() }
}

//go:build windows

package filelock

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

// Lock holds an OS lock via the open lock-file handle. Windows releases the
// lock automatically when the handle is closed, including on process death, so
// a crashed instance never leaves a stale lock.
type Lock struct {
	f *os.File
}

// TryExclusive creates/opens lockPath and attempts a non-blocking exclusive
// lock (LockFileEx with LOCKFILE_FAIL_IMMEDIATELY). Return contract matches the
// Unix build: (lock,true,nil) acquired, (nil,false,nil) held elsewhere,
// (nil,false,err) the lock file could not be created/opened.
func TryExclusive(lockPath string) (*Lock, bool, error) {
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, false, err
	}
	var overlapped windows.Overlapped
	err = windows.LockFileEx(
		windows.Handle(f.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
		0, 1, 0, &overlapped,
	)
	if err != nil {
		_ = f.Close()
		if errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
			return nil, false, nil // held by another process
		}
		return nil, false, err
	}
	return &Lock{f: f}, true, nil
}

// lockShared takes a shared lock on f (LockFileEx without
// LOCKFILE_EXCLUSIVE_LOCK): holders of the shared lock exclude one exclusive
// holder, not each other. Without wait it reports false at once when an
// exclusive holder has it.
func lockShared(f *os.File, wait bool) (bool, error) {
	var flags uint32
	if !wait {
		flags = windows.LOCKFILE_FAIL_IMMEDIATELY
	}
	var overlapped windows.Overlapped
	err := windows.LockFileEx(windows.Handle(f.Fd()), flags, 0, 1, 0, &overlapped)
	if errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
		return false, nil
	}
	return err == nil, err
}

// Release drops the lock by closing the handle. Safe on a nil/empty lock.
func (l *Lock) Release() error {
	if l == nil || l.f == nil {
		return nil
	}
	err := l.f.Close()
	l.f = nil
	return err
}

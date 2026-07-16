//go:build windows

package database

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

// fileLock holds an OS lock via the open lock-file handle. Windows releases the
// lock automatically when the handle is closed, including on process death, so
// a crashed instance never leaves a stale lock.
type fileLock struct {
	f *os.File
}

// tryLockExclusive creates/opens lockPath and attempts a non-blocking exclusive
// lock (LockFileEx with LOCKFILE_FAIL_IMMEDIATELY). Return contract matches the
// Unix build: (lock,true,nil) acquired, (nil,false,nil) held elsewhere,
// (nil,false,err) the lock file could not be created/opened.
func tryLockExclusive(lockPath string) (*fileLock, bool, error) {
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
	return &fileLock{f: f}, true, nil
}

// release drops the lock by closing the handle. Safe on a nil/empty lock.
func (l *fileLock) release() error {
	if l == nil || l.f == nil {
		return nil
	}
	err := l.f.Close()
	l.f = nil
	return err
}

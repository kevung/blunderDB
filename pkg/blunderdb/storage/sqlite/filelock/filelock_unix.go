//go:build !windows

package filelock

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

// Lock holds an OS advisory lock via the open lock-file descriptor. The
// lock is released when the descriptor is closed — including on process death
// (crash, kill, power loss), so a crashed instance never leaves a stale lock.
type Lock struct {
	f *os.File
}

// TryExclusive creates/opens lockPath and attempts a NON-blocking exclusive
// advisory lock (BSD flock). It returns:
//   - (lock, true, nil)  the lock was acquired;
//   - (nil, false, nil)  another process already holds it;
//   - (nil, false, err)  the lock file itself could not be created/opened
//     (e.g. a read-only directory) — the caller decides whether that is fatal.
//
// flock (not fcntl) is used deliberately: its lock is tied to the open file
// description, so it is released atomically on close and does not suffer the
// fcntl "closing any fd drops all locks" footgun.
func TryExclusive(lockPath string) (*Lock, bool, error) {
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, false, err
	}
	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		_ = f.Close()
		if errors.Is(err, unix.EWOULDBLOCK) {
			return nil, false, nil // held by another process
		}
		return nil, false, err
	}
	return &Lock{f: f}, true, nil
}

// lockShared takes a shared advisory lock on f: holders of the shared lock
// exclude one exclusive holder, not each other. Without wait it reports
// false at once when an exclusive holder has it.
func lockShared(f *os.File, wait bool) (bool, error) {
	how := unix.LOCK_SH
	if !wait {
		how |= unix.LOCK_NB
	}
	for {
		err := unix.Flock(int(f.Fd()), how)
		switch {
		case errors.Is(err, unix.EINTR):
			continue
		case errors.Is(err, unix.EWOULDBLOCK):
			return false, nil
		}
		return err == nil, err
	}
}

// Release drops the lock by closing the descriptor. Safe on a nil/empty lock.
func (l *Lock) Release() error {
	if l == nil || l.f == nil {
		return nil
	}
	err := l.f.Close()
	l.f = nil
	return err
}

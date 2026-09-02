//go:build !windows

package sqlite

import "syscall"

// freeSpaceBytes returns the free space available to the current user on the
// filesystem containing path (path may be an existing file or directory).
// Used by Vacuum to refuse a run rather than let SQLite's file rebuild fail
// partway through for lack of room.
func freeSpaceBytes(path string) (uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, err
	}
	return uint64(stat.Bsize) * stat.Bavail, nil
}

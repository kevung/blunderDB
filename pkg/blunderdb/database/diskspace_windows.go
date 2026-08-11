//go:build windows

package database

import (
	"path/filepath"

	"golang.org/x/sys/windows"
)

// freeSpaceBytes returns the free space available to the current user on the
// volume containing path (an existing file). GetDiskFreeSpaceEx wants a
// directory, so the file's parent is queried. Used by Vacuum to refuse a run
// rather than let SQLite's file rebuild fail partway through for lack of room.
func freeSpaceBytes(path string) (uint64, error) {
	dir, err := windows.UTF16PtrFromString(filepath.Dir(path))
	if err != nil {
		return 0, err
	}
	var freeBytesAvailable uint64
	if err := windows.GetDiskFreeSpaceEx(dir, &freeBytesAvailable, nil, nil); err != nil {
		return 0, err
	}
	return freeBytesAvailable, nil
}

//go:build !linux

package sqlite

// availableMemoryBytes is only implemented on Linux; elsewhere the memory
// check is skipped.
func availableMemoryBytes() (uint64, bool) { return 0, false }

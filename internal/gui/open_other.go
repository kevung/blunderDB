//go:build !windows

package gui

import "errors"

// shellOpen exists only to let openFile compile everywhere; Windows has its own.
func shellOpen(string) error { return errors.New("shellOpen: windows only") }

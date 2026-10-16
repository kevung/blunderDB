//go:build !linux

package main

import "errors"

// lowerPriority is Linux-only; elsewhere run the tool under the system's
// own nice or start /low.
func lowerPriority() error {
	return errors.New("not supported on this system: start the tool at low priority")
}

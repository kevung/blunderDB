//go:build !windows

package videomux

import "os/exec"

func hideConsole(*exec.Cmd) {}

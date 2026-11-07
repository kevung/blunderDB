package videomux

import (
	"os/exec"
	"syscall"
)

// hideConsole keeps a console window from flashing up over the GUI while ffmpeg runs.
func hideConsole(cmd *exec.Cmd) {
	const createNoWindow = 0x08000000
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
}

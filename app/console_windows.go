package main

import (
	"os/exec"
	"syscall"
)

// createNoWindow is CREATE_NO_WINDOW: a console command started from the app opens no window of its
// own. A terminal it starts gets one anyway: start gives it a new console.
const createNoWindow = 0x08000000

// hideConsole keeps a command's console window from flashing up, as code.cmd's would.
func hideConsole(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNoWindow}
}

//go:build !windows

package main

import "os/exec"

// hideConsole does nothing: only Windows gives a console command a window.
func hideConsole(*exec.Cmd) {}

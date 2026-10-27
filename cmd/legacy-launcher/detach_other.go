//go:build !windows

package main

import "os/exec"

// detach does nothing: the launcher is only shipped for Windows.
func detach(*exec.Cmd) {}

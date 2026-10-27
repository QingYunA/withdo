//go:build !windows

package main

// newWindowsInstall has nothing to look at outside Windows.
func newWindowsInstall() (windowsInstall, bool) { return windowsInstall{}, false }

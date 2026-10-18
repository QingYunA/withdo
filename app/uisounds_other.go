//go:build !darwin

package main

// uiSoundsOn says whether the system plays user interface sound effects; elsewhere there is no such
// setting to read, so they are on.
func uiSoundsOn() bool { return true }

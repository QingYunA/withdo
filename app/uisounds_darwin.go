package main

import (
	"os/exec"
	"strings"
)

// uiSoundsOn says whether macOS plays user interface sound effects (System Settings > Sound). The
// setting is a key of the global domain that is missing until someone changes it, and a missing key
// is on. It runs a command, so it is not for the UI thread.
func uiSoundsOn() bool {
	out, err := exec.Command("defaults", "read", "-g", "com.apple.sound.uiaudio.enabled").Output()
	return err != nil || strings.TrimSpace(string(out)) != "0"
}

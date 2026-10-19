package main

import (
	"path/filepath"

	"github.com/QingYunA/adhd-todo/internal/cli"
)

// cliName is the name the agents run the app under: ~/.local/bin/adhd-core is a symlink to the app's
// executable, so the app is adhd-core too.
const cliName = "adhd-core"

// cliArgs reports whether the process was started as the command line, and with which arguments:
// when it was run under the name adhd-core, or its first argument is one of the command line's own.
// Anything else, like no arguments or the ones macOS and Xcode pass a launched app, starts the window.
func cliArgs(argv []string) ([]string, bool) {
	if len(argv) == 0 {
		return nil, false
	}
	args := argv[1:]
	if filepath.Base(argv[0]) == cliName || (len(args) > 0 && cli.IsCommand(args[0])) {
		return args, true
	}
	return nil, false
}

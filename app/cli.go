package main

import (
	"path/filepath"
	"slices"

	"github.com/QingYunA/adhd-todo/internal/cli"
)

// cliNames are the names the agents run the app under: ~/.local/bin/withdo is a symlink to the app's
// executable, so the app is withdo too. adhd-core is the name before the rename; it stays so that
// an agent config that was not migrated yet keeps working.
var cliNames = []string{"withdo", "adhd-core"}

// cliArgs reports whether the process was started as the command line, and with which arguments:
// when it was run under the name withdo (or adhd-core), or its first argument is one of the command
// line's own. Anything else, like no arguments or the ones macOS and Xcode pass a launched app,
// starts the window.
func cliArgs(argv []string) ([]string, bool) {
	if len(argv) == 0 {
		return nil, false
	}
	args := argv[1:]
	if slices.Contains(cliNames, filepath.Base(argv[0])) || (len(args) > 0 && cli.IsCommand(args[0])) {
		return args, true
	}
	return nil, false
}

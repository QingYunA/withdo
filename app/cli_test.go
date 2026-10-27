package main

import (
	"slices"
	"testing"
)

func TestCLIArgsDecidesBetweenTheCommandLineAndTheWindow(t *testing.T) {
	cases := []struct {
		name string
		argv []string
		args []string
		cli  bool
	}{
		{"linked name without arguments", []string{"/Users/me/.local/bin/withdo"}, []string{}, true},
		{"linked name with a command", []string{"withdo", "mcp"}, []string{"mcp"}, true},
		{"linked name with an unknown argument", []string{"withdo", "bogus"}, []string{"bogus"}, true},
		{"legacy name without arguments", []string{"/Users/me/.local/bin/adhd-core"}, []string{}, true},
		{"legacy name with a command", []string{"adhd-core", "mcp"}, []string{"mcp"}, true},
		{"app name with a command", []string{"/Applications/ADHD Todo.app/Contents/MacOS/ADHD Todo", "--version"}, []string{"--version"}, true},
		{"app name with help", []string{"ADHD Todo", "-h"}, []string{"-h"}, true},
		{"app name without arguments", []string{"ADHD Todo"}, nil, false},
		{"app name with the demo flag", []string{"ADHD Todo", "-fake"}, nil, false},
		{"app name launched by macOS", []string{"ADHD Todo", "-psn_0_12345"}, nil, false},
		{"app name launched by Xcode", []string{"ADHD Todo", "-NSDocumentRevisionsDebugMode", "YES"}, nil, false},
		{"command word after another argument", []string{"ADHD Todo", "-fake", "mcp"}, nil, false},
		{"no argv at all", nil, nil, false},
	}
	for _, c := range cases {
		args, ok := cliArgs(c.argv)
		if ok != c.cli || !slices.Equal(args, c.args) {
			t.Errorf("%s: cliArgs(%q) = %q, %v; want %q, %v", c.name, c.argv, args, ok, c.args, c.cli)
		}
	}
}

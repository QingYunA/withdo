package main

import "testing"

func TestConnectCommand(t *testing.T) {
	cases := []struct{ name, goos, core, home, want string }{
		{"mac shows the link under ~", "darwin", "/Users/me/.local/bin/withdo", "/Users/me",
			"claude mcp add --scope user withdo -- ~/.local/bin/withdo mcp"},
		{"mac outside home keeps the path", "darwin", "/opt/tools/withdo", "/Users/me",
			"claude mcp add --scope user withdo -- /opt/tools/withdo mcp"},
		{"windows quotes the path", "windows", `C:\Users\me\AppData\Local\Programs\Withdo\withdo-cli.exe`, `C:\Users\me`,
			`claude mcp add --scope user withdo -- "C:\Users\me\AppData\Local\Programs\Withdo\withdo-cli.exe" mcp`},
	}
	for _, c := range cases {
		if got := connectCommand(c.goos, c.core, c.home); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}

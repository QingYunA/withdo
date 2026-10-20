package main

import "testing"

func TestConnectCommand(t *testing.T) {
	cases := []struct{ name, goos, core, home, want string }{
		{"mac shows the link under ~", "darwin", "/Users/me/.local/bin/adhd-core", "/Users/me",
			"claude mcp add --scope user adhd-todo -- ~/.local/bin/adhd-core mcp"},
		{"mac outside home keeps the path", "darwin", "/opt/tools/adhd-core", "/Users/me",
			"claude mcp add --scope user adhd-todo -- /opt/tools/adhd-core mcp"},
		{"windows quotes the path", "windows", `C:\Users\me\AppData\Local\Programs\ADHD Todo\adhd-core.exe`, `C:\Users\me`,
			`claude mcp add --scope user adhd-todo -- "C:\Users\me\AppData\Local\Programs\ADHD Todo\adhd-core.exe" mcp`},
	}
	for _, c := range cases {
		if got := connectCommand(c.goos, c.core, c.home); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}

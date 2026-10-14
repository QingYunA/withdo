package paths

import "testing"

func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func TestDBPath(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		goos string
		home string
		want string
	}{
		{"override wins", map[string]string{"ADHD_TODO_DB": "/tmp/x.db"}, "darwin", "/home/u", "/tmp/x.db"},
		{"empty override ignored", map[string]string{"ADHD_TODO_DB": ""}, "linux", "/home/u", "/home/u/.local/share/adhd-todo/adhd-todo.db"},
		{"macOS", nil, "darwin", "/home/u", "/home/u/Library/Application Support/adhd-todo/adhd-todo.db"},
		{"linux XDG", map[string]string{"XDG_DATA_HOME": "/xdg"}, "linux", "/home/u", "/xdg/adhd-todo/adhd-todo.db"},
		{"linux default", nil, "linux", "/home/u", "/home/u/.local/share/adhd-todo/adhd-todo.db"},
		{"windows APPDATA", map[string]string{"APPDATA": `C:\Users\u\AppData\Roaming`}, "windows", "", `C:\Users\u\AppData\Roaming\adhd-todo\adhd-todo.db`},
		{"windows fallback", nil, "windows", `C:\Users\u`, `C:\Users\u\AppData\Roaming\adhd-todo\adhd-todo.db`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DBPath(env(tt.env), tt.goos, tt.home); got != tt.want {
				t.Errorf("DBPath = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDefaultDBPath(t *testing.T) {
	t.Setenv(EnvDB, "/tmp/override.db")
	got, err := DefaultDBPath()
	if err != nil || got != "/tmp/override.db" {
		t.Fatalf("DefaultDBPath = %q, %v", got, err)
	}
	t.Setenv(EnvDB, "")
	got, err = DefaultDBPath()
	if err != nil || got == "" {
		t.Fatalf("DefaultDBPath without override = %q, %v", got, err)
	}
}

func TestDefaultDBPathWithoutHome(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv(EnvDB, "")
	if _, err := DefaultDBPath(); err == nil {
		t.Skip("platform resolves a home directory without $HOME")
	}
	t.Setenv(EnvDB, "/tmp/x.db")
	if got, err := DefaultDBPath(); err != nil || got != "/tmp/x.db" {
		t.Fatalf("override should work without a home directory: %q, %v", got, err)
	}
}

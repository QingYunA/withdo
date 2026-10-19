package views

import (
	"path/filepath"
	"strings"

	"github.com/QingYunA/adhd-todo/app/data"
)

// overviewTask describes a task the way the overview does: with its latest report and whether it
// has gone stale.
func (a *App) overviewTask(t data.Task) data.OverviewTask { return a.snap.all.Describe(t, a.Now()) }

// withTilde writes a path under the home folder the way the user writes it: "~/Library/…".
func withTilde(path, home string) string {
	if home == "" || home == "/" {
		return path
	}
	// Paths come from filepath.Join on the host OS, but config values may keep
	// Unix slashes wherever the app runs: accept either separator.
	for _, sep := range []string{"/", "\\"} {
		if rest, ok := strings.CutPrefix(path, home+sep); ok {
			return "~/" + filepath.ToSlash(rest)
		}
	}
	return path
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

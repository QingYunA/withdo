// Package paths locates adhd-todo's data on disk.
package paths

import (
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	appDir = "adhd-todo"
	dbFile = "adhd-todo.db"
)

// EnvDB overrides the database location when set to a non-empty value.
const EnvDB = "ADHD_TODO_DB"

// DBPath resolves the SQLite database path for the given OS ("darwin",
// "windows", anything else is treated as XDG-style Unix). getenv looks up
// environment variables and home is the user's home directory.
func DBPath(getenv func(string) string, goos, home string) string {
	if p := getenv(EnvDB); p != "" {
		return p
	}
	switch goos {
	case "windows":
		base := getenv("APPDATA")
		if base == "" {
			base = winJoin(home, "AppData", "Roaming")
		}
		return winJoin(base, appDir, dbFile)
	case "darwin":
		return path.Join(home, "Library", "Application Support", appDir, dbFile)
	default:
		base := getenv("XDG_DATA_HOME")
		if base == "" {
			base = path.Join(home, ".local", "share")
		}
		return path.Join(base, appDir, dbFile)
	}
}

// DefaultDBPath resolves the database path for the running process.
func DefaultDBPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		// ADHD_TODO_DB alone is enough, so only fail when it is unset.
		if p := os.Getenv(EnvDB); p != "" {
			return p, nil
		}
		return "", err
	}
	return DBPath(os.Getenv, runtime.GOOS, home), nil
}

// ImagesDir is the folder of the pictures kept for tasks, inside the data folder (the directory
// of the database).
func ImagesDir(dataFolder string) string {
	return filepath.Join(dataFolder, "images")
}

func winJoin(parts ...string) string {
	trimmed := make([]string, len(parts))
	for i, p := range parts {
		if i == 0 {
			trimmed[i] = strings.TrimRight(p, `\`)
		} else {
			trimmed[i] = strings.Trim(p, `\`)
		}
	}
	return strings.Join(trimmed, `\`)
}

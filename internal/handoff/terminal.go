package handoff

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Request is what to start: Argv (the agent's full path, then its arguments) in Dir, with PathDirs
// put before PATH, for an agent installed where a login shell does not look.
type Request struct {
	Dir      string
	Argv     []string
	PathDirs []string
}

// Launcher starts an agent where the user can see it. The app uses Terminal; tests pass a fake,
// so that no test ever opens a window.
type Launcher interface {
	Launch(ctx context.Context, req Request) error
}

const (
	scriptPrefix = "handoff-"
	scriptSuffix = ".command"
	// keepScripts is how long a script stays on disk: it holds a prompt, so it should not pile up.
	keepScripts = 24 * time.Hour
)

// Terminal is the Launcher of a Mac: it writes a .command script into Folder and opens it, which
// runs it in Terminal. The zero value uses the wall clock and the `open` command.
type Terminal struct {
	Folder string                                       // where the scripts go; made if missing
	Now    func() time.Time                             // default time.Now
	Open   func(ctx context.Context, file string) error // default: `open file`
}

var _ Launcher = Terminal{}

// Launch implements Launcher.
func (t Terminal) Launch(ctx context.Context, req Request) error {
	if len(req.Argv) == 0 {
		return fmt.Errorf("launch: no command")
	}
	if err := os.MkdirAll(t.Folder, 0o700); err != nil {
		return fmt.Errorf("launch: %w", err)
	}
	t.sweep()
	f, err := os.CreateTemp(t.Folder, scriptPrefix+"*"+scriptSuffix)
	if err != nil {
		return fmt.Errorf("launch: %w", err)
	}
	_, werr := f.WriteString(Script(req))
	if cerr := f.Close(); werr == nil {
		werr = cerr
	}
	if werr == nil {
		werr = os.Chmod(f.Name(), 0o700)
	}
	if werr != nil {
		_ = os.Remove(f.Name())
		return fmt.Errorf("launch: %w", werr)
	}
	open := t.Open
	if open == nil {
		open = openFile
	}
	if err := open(ctx, f.Name()); err != nil {
		_ = os.Remove(f.Name())
		return fmt.Errorf("launch: %w", err)
	}
	return nil
}

// Script is the text of the .command file for req.
func Script(req Request) string {
	var b strings.Builder
	b.WriteString("#!/bin/zsh -l\n")
	if len(req.PathDirs) > 0 {
		fmt.Fprintf(&b, "export PATH=%s:\"$PATH\"\n", Quote(strings.Join(req.PathDirs, ":")))
	}
	fmt.Fprintf(&b, "cd %s || exit 1\n", Quote(req.Dir))
	fmt.Fprintf(&b, "exec %s\n", QuoteAll(req.Argv))
	return b.String()
}

// sweep removes the scripts older than keepScripts; it is best effort.
func (t Terminal) sweep() {
	now := time.Now()
	if t.Now != nil {
		now = t.Now()
	}
	entries, err := os.ReadDir(t.Folder)
	if err != nil {
		return
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasPrefix(name, scriptPrefix) || !strings.HasSuffix(name, scriptSuffix) {
			continue
		}
		if info, err := e.Info(); err == nil && now.Sub(info.ModTime()) > keepScripts {
			_ = os.Remove(filepath.Join(t.Folder, name))
		}
	}
}

func openFile(ctx context.Context, file string) error {
	if out, err := exec.CommandContext(ctx, "open", file).CombinedOutput(); err != nil {
		return fmt.Errorf("open %s: %w: %s", file, err, strings.TrimSpace(string(out)))
	}
	return nil
}

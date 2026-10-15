package agentsetup

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Runner runs a command and returns what it printed (both streams). The app uses the real one;
// tests pass a fake, so that no agent is ever started by a test.
type Runner func(ctx context.Context, path string, args ...string) ([]byte, error)

// Env is what Detect and Install look at: the home folder, how to find a command and how to run
// one. A zero field means the real thing.
type Env struct {
	Home     string                            // the user's home folder
	Core     string                            // the adhd-core binary the agents run; default $HOME/.local/bin/adhd-core
	LookPath func(string) (string, error)      // finds a command; default: PATH plus the usual places
	Run      Runner                            // runs a command; default: os/exec with a PATH an app launched from the Dock lacks
	Timeout  time.Duration                     // for one command; default 30 s
	Stat     func(string) (os.FileInfo, error) // default os.Stat
}

// New is the Env of the real machine for home.
func New(home string) *Env { return &Env{Home: home} }

// DefaultTimeout bounds one `mcp add` command.
const DefaultTimeout = 30 * time.Second

func (e *Env) core() string {
	if e.Core != "" {
		return e.Core
	}
	return filepath.Join(e.Home, ".local", "bin", "adhd-core")
}

// CorePath is the adhd-core binary the agents will run.
func (e *Env) CorePath() string { return e.core() }

func (e *Env) timeout() time.Duration {
	if e.Timeout > 0 {
		return e.Timeout
	}
	return DefaultTimeout
}

func (e *Env) stat(p string) (os.FileInfo, error) {
	if e.Stat != nil {
		return e.Stat(p)
	}
	return os.Stat(p)
}

func (e *Env) exists(p string) bool {
	_, err := e.stat(p)
	return err == nil
}

func (e *Env) path(rel string) string { return filepath.Join(e.Home, filepath.FromSlash(rel)) }

// extraDirs are where a command line tool usually lives and a GUI app's PATH does not reach.
func (e *Env) extraDirs() []string {
	return []string{
		"/opt/homebrew/bin", "/usr/local/bin",
		filepath.Join(e.Home, ".local", "bin"),
		filepath.Join(e.Home, ".claude", "local"),
		filepath.Join(e.Home, ".npm-global", "bin"),
		filepath.Join(e.Home, ".bun", "bin"),
		filepath.Join(e.Home, ".cargo", "bin"),
		filepath.Join(e.Home, ".volta", "bin"),
		filepath.Join(e.Home, "go", "bin"),
		"/Applications/Visual Studio Code.app/Contents/Resources/app/bin",
	}
}

func (e *Env) lookPath(name string) (string, error) {
	if e.LookPath != nil {
		return e.LookPath(name)
	}
	if p, err := exec.LookPath(name); err == nil {
		return p, nil
	}
	for _, d := range e.extraDirs() {
		p := filepath.Join(d, name)
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() && fi.Mode()&0o111 != 0 {
			return p, nil
		}
	}
	return "", exec.ErrNotFound
}

func (e *Env) run(ctx context.Context, path string, args ...string) ([]byte, error) {
	if e.Run != nil {
		return e.Run(ctx, path, args...)
	}
	cmd := exec.CommandContext(ctx, path, args...)
	// A tool installed with npm starts `node` through env, which needs its own folder on PATH.
	dirs := append([]string{filepath.Dir(path)}, e.extraDirs()...)
	cmd.Env = append(os.Environ(), "PATH="+strings.Join(dirs, ":")+":"+os.Getenv("PATH"))
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	return out.Bytes(), err
}

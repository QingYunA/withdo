package ghpoll

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Output is what a gh run printed.
type Output struct{ Stdout, Stderr []byte }

// Runner runs gh. The real one starts the program; tests use a fake.
type Runner interface {
	// Run runs gh with args. It returns ErrNotInstalled when there is no gh, an *ExitError when gh
	// exits with a code other than 0, and the context's error when ctx ends first.
	Run(ctx context.Context, args ...string) (Output, error)
}

// ErrNotInstalled says there is no gh to run.
var ErrNotInstalled = errors.New("gh is not installed")

// ExitError is a gh run that ended with a non-zero exit code.
type ExitError struct {
	Code           int
	Stdout, Stderr []byte
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("gh exited with code %d: %s", e.Code, strings.TrimSpace(e.Output()))
}

// Output is everything gh printed, for matching against its messages.
func (e *ExitError) Output() string { return string(e.Stdout) + string(e.Stderr) }

// usualFolders are where gh lives when the app is started from Finder with its short PATH.
var usualFolders = []string{"/opt/homebrew/bin", "/usr/local/bin"}

// FindGH looks for an executable named gh in each folder of pathEnv (a PATH), then in extra.
func FindGH(pathEnv string, extra []string) (string, error) {
	folders := append(filepath.SplitList(pathEnv), extra...)
	for _, dir := range folders {
		if dir == "" {
			continue
		}
		candidate := filepath.Join(dir, "gh")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
			return candidate, nil
		}
	}
	return "", ErrNotInstalled
}

// ExecRunner runs the gh found on disk. The zero value looks in $PATH and the usual folders and
// gives each run 30 seconds.
type ExecRunner struct {
	Path    string        // the PATH to search; $PATH when empty
	Extra   []string      // folders to search after it; the usual ones when nil
	Timeout time.Duration // per run; 30 seconds when 0
}

// Run implements Runner. gh never prompts and prints no colors or update notices.
func (r *ExecRunner) Run(ctx context.Context, args ...string) (Output, error) {
	pathEnv, extra, timeout := r.Path, r.Extra, r.Timeout
	if pathEnv == "" {
		pathEnv = os.Getenv("PATH")
	}
	if extra == nil {
		extra = usualFolders
	}
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	path, err := FindGH(pathEnv, extra)
	if err != nil {
		return Output{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	cmd.Env = append(os.Environ(), "GH_PROMPT_DISABLED=1", "NO_COLOR=1", "GH_NO_UPDATE_NOTIFIER=1")
	cmd.WaitDelay = time.Second // a stuck child must not hold the pipes open
	out := func() Output { return Output{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()} }
	err = cmd.Run()
	if ctxErr := ctx.Err(); ctxErr != nil {
		return out(), fmt.Errorf("run gh: %w", ctxErr)
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return out(), &ExitError{Code: exit.ExitCode(), Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}
	}
	if err != nil {
		return out(), fmt.Errorf("run gh: %w", err)
	}
	return out(), nil
}

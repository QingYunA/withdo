package ghpoll

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// ghFile is the name FindGH looks for on this platform.
var ghFile = "gh"

func init() {
	if runtime.GOOS == "windows" {
		ghFile = "gh.exe"
	}
}

// skipWindowsExec skips tests that execute the fake gh: it is a POSIX shell
// script and Windows cannot run it. FindGH coverage still runs there.
func skipWindowsExec(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake gh is a shell script; Windows cannot execute it")
	}
}

// script writes an executable shell script named gh into a new directory and returns the directory.
// On Windows the file is never executed; it only needs to exist for FindGH.
func script(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ghFile), []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestFindGHSearchesPathThenTheUsualFolders(t *testing.T) {
	onPath, extra, none := script(t, "true"), script(t, "true"), t.TempDir()

	got, err := FindGH(none+string(os.PathListSeparator)+onPath, []string{extra})
	if err != nil || got != filepath.Join(onPath, ghFile) {
		t.Errorf("PATH first: got %q, %v", got, err)
	}
	// Started from Finder the PATH is short: the folders Homebrew uses are tried after it.
	got, err = FindGH(none, []string{none, extra})
	if err != nil || got != filepath.Join(extra, ghFile) {
		t.Errorf("extra folders: got %q, %v", got, err)
	}
	if _, err = FindGH(none, []string{none}); !errors.Is(err, ErrNotInstalled) {
		t.Errorf("not found: err = %v, want ErrNotInstalled", err)
	}
}

func TestFindGHSkipsAFolderNamedGHAndPlainFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "gh"), 0o755); err != nil {
		t.Fatal(err)
	}
	plain := t.TempDir()
	if err := os.WriteFile(filepath.Join(plain, "gh"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := FindGH(dir+string(os.PathListSeparator)+plain, nil); !errors.Is(err, ErrNotInstalled) {
		t.Errorf("err = %v, want ErrNotInstalled", err)
	}
}

func execRunner(dir string, timeout time.Duration) *ExecRunner {
	return &ExecRunner{Path: dir, Extra: []string{}, Timeout: timeout}
}

func TestExecRunnerRunsGHWithItsArguments(t *testing.T) {
	skipWindowsExec(t)
	r := execRunner(script(t, `printf '%s|' "$@"; echo oops >&2`), time.Minute)
	out, err := r.Run(context.Background(), "pr", "view", "https://github.com/a/b/pull/1")
	if err != nil {
		t.Fatal(err)
	}
	if got := string(out.Stdout); got != "pr|view|https://github.com/a/b/pull/1|" {
		t.Errorf("stdout = %q", got)
	}
	if strings.TrimSpace(string(out.Stderr)) != "oops" {
		t.Errorf("stderr = %q", out.Stderr)
	}
}

func TestExecRunnerReportsAnExitCodeWithTheOutput(t *testing.T) {
	skipWindowsExec(t)
	r := execRunner(script(t, `echo "nothing here" >&2; echo partial; exit 4`), time.Minute)
	_, err := r.Run(context.Background(), "auth", "status")
	var exit *ExitError
	if !errors.As(err, &exit) {
		t.Fatalf("err = %v, want an ExitError", err)
	}
	if exit.Code != 4 || !strings.Contains(exit.Output(), "nothing here") || !strings.Contains(exit.Output(), "partial") {
		t.Errorf("got code %d, output %q", exit.Code, exit.Output())
	}
}

func TestExecRunnerMissingGH(t *testing.T) {
	r := execRunner(t.TempDir(), time.Minute)
	if _, err := r.Run(context.Background(), "--version"); !errors.Is(err, ErrNotInstalled) {
		t.Errorf("err = %v, want ErrNotInstalled", err)
	}
}

func TestExecRunnerTimesOut(t *testing.T) {
	skipWindowsExec(t)
	r := execRunner(script(t, `sleep 30`), 100*time.Millisecond)
	start := time.Now()
	_, err := r.Run(context.Background(), "pr", "view")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v, want a deadline error", err)
	}
	if time.Since(start) > 5*time.Second {
		t.Errorf("took %v: the child was not killed", time.Since(start))
	}
}

func TestExecRunnerStopsWhenItsContextIsCancelled(t *testing.T) {
	skipWindowsExec(t)
	r := execRunner(script(t, `sleep 30`), time.Minute)
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(100*time.Millisecond, cancel)
	if _, err := r.Run(ctx, "pr", "view"); !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

func TestExecRunnerDoesNotPromptOrColor(t *testing.T) {
	skipWindowsExec(t)
	r := execRunner(script(t, `echo "$GH_PROMPT_DISABLED $NO_COLOR $GH_NO_UPDATE_NOTIFIER"`), time.Minute)
	out, err := r.Run(context.Background(), "x")
	if err != nil || strings.TrimSpace(string(out.Stdout)) != "1 1 1" {
		t.Errorf("env: %q, %v", out.Stdout, err)
	}
}

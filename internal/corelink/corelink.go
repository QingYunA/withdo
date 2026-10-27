// Package corelink keeps the stable withdo path the coding agents run ($HOME/.local/bin/withdo)
// pointing at the app's executable, which is withdo too, so that it updates with the app. The app
// does the same for the legacy adhd-core name beside it, so that agent configs written before the
// rename keep working.
package corelink

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Action is what Ensure did.
type Action string

const (
	Linked    Action = "linked"    // the path is now a symlink to the bundled binary
	Unchanged Action = "unchanged" // it already was
	Kept      Action = "kept"      // it is something of the user's, so it stays
)

// Result is what Ensure did and why.
type Result struct {
	Action Action
	Reason string
}

// versionTimeout bounds one `withdo --version`.
const versionTimeout = 5 * time.Second

// Ensure makes link a symlink to bundled when link is missing, is a symlink into some .app bundle
// (an older install of the app, maybe gone), or is a regular file older than bundled, which is
// kept as link.old. Anything else
// (a symlink elsewhere, a file as new as bundled, a directory) is a developer's own and stays.
func Ensure(link, bundled string) (Result, error) {
	bundledVersion, err := versionOf(bundled)
	if err != nil {
		return Result{}, fmt.Errorf("bundled withdo: %w", err)
	}
	fi, err := os.Lstat(link)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return replace(link, bundled, "missing")
	case err != nil:
		return Result{}, err
	case fi.Mode()&os.ModeSymlink != 0:
		return fromSymlink(link, bundled)
	case fi.Mode().IsRegular():
		return fromFile(link, bundled, bundledVersion)
	}
	return Result{Kept, "not a file or a symlink"}, nil
}

func fromSymlink(link, bundled string) (Result, error) {
	to, err := os.Readlink(link)
	if err != nil {
		return Result{}, err
	}
	switch {
	case to == bundled:
		return Result{Unchanged, "already points at the app"}, nil
	case strings.Contains(to, ".app/"):
		return replace(link, bundled, "pointed into "+to)
	}
	return Result{Kept, "points at " + to}, nil
}

func fromFile(link, bundled, bundledVersion string) (Result, error) {
	have, err := versionOf(link)
	if err != nil || older(have, bundledVersion) {
		if err := keepOld(link); err != nil {
			return Result{}, err
		}
		return replace(link, bundled, "version "+quote(have)+" is below "+bundledVersion+"; the file is kept as "+link+".old")
	}
	return Result{Kept, "version " + have + " is not below " + bundledVersion}, nil
}

func quote(v string) string {
	if v == "" {
		return "unknown"
	}
	return v
}

// keepOld keeps the file at link as link.old before it is replaced, in case it was someone's own
// build: a hard link, so link itself stays in place until replace renames over it.
func keepOld(link string) error {
	old := link + ".old"
	if err := os.Remove(old); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Link(link, old)
}

// replace points link at bundled: a new symlink renamed over link, so link is never missing.
func replace(link, bundled, reason string) (Result, error) {
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		return Result{}, err
	}
	tmp := link + ".new"
	_ = os.Remove(tmp)
	if err := os.Symlink(bundled, tmp); err != nil {
		return Result{}, err
	}
	if err := os.Rename(tmp, link); err != nil {
		_ = os.Remove(tmp)
		return Result{}, err
	}
	return Result{Linked, reason}, nil
}

// versionOf asks a binary for its version: the last word of what `--version` prints.
func versionOf(path string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), versionTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, path, "--version").Output()
	if err != nil {
		return "", err
	}
	words := strings.Fields(string(out))
	if len(words) == 0 {
		return "", errors.New("--version printed nothing")
	}
	return words[len(words)-1], nil
}

// older reports whether version a is below b. A version that is not dotted numbers is below any.
func older(a, b string) bool {
	pa, okA := parse(a)
	pb, okB := parse(b)
	if !okA || !okB {
		return !okA && okB
	}
	for i := range max(len(pa), len(pb)) {
		if x, y := at(pa, i), at(pb, i); x != y {
			return x < y
		}
	}
	return false
}

func at(v []int, i int) int {
	if i < len(v) {
		return v[i]
	}
	return 0
}

func parse(v string) ([]int, bool) {
	var out []int
	for _, part := range strings.Split(v, ".") {
		n, err := strconv.Atoi(part)
		if err != nil {
			return nil, false
		}
		out = append(out, n)
	}
	return out, v != ""
}

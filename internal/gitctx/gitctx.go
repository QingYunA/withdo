// Package gitctx describes the git repository that contains a directory:
// which repo it is (GitHub slug or main working tree), and the current branch.
package gitctx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ErrNotRepo means the directory is missing or not inside a git repository.
var ErrNotRepo = errors.New("not inside a git repository")

// Info describes the repository containing a directory.
type Info struct {
	// Toplevel is the root of the working tree containing the directory.
	Toplevel string
	// RepoRoot is the main working tree (or the bare repo itself). It differs
	// from Toplevel inside a linked worktree.
	RepoRoot string
	// Branch is the current branch; empty when HEAD is detached.
	Branch string
	// RepoSlug is "owner/repo" when origin is a GitHub remote, else empty.
	RepoSlug string
	// RepoName is the repo part of RepoSlug, else the RepoRoot directory name.
	RepoName string
}

// Resolver finds the repository for a directory. It is a function type so
// callers can substitute a fake in tests.
type Resolver func(ctx context.Context, dir string) (*Info, error)

// Resolve describes the repository containing dir. It returns ErrNotRepo when
// dir does not exist or is outside a repository, and another error when git
// itself cannot run.
func Resolve(ctx context.Context, dir string) (*Info, error) {
	toplevel, err := git(ctx, dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, err
	}
	// symbolic-ref also works on an unborn branch and fails when detached.
	branch, _ := git(ctx, toplevel, "symbolic-ref", "--short", "-q", "HEAD")
	slug := ""
	if remote, rerr := git(ctx, toplevel, "remote", "get-url", "origin"); rerr == nil {
		slug = ParseRemoteURL(remote)
	}
	root := repoRoot(ctx, toplevel)
	name := filepath.Base(root)
	if slug != "" {
		name = slug[strings.Index(slug, "/")+1:]
	}
	return &Info{Toplevel: toplevel, RepoRoot: root, Branch: branch, RepoSlug: slug, RepoName: name}, nil
}

// repoRoot maps a linked worktree to its main working tree.
func repoRoot(ctx context.Context, toplevel string) string {
	out, err := git(ctx, toplevel, "rev-parse", "--path-format=absolute", "--git-dir", "--git-common-dir")
	if err != nil {
		return toplevel
	}
	lines := strings.Split(out, "\n")
	if len(lines) < 2 || lines[0] == "" || lines[1] == "" || lines[0] == lines[1] {
		return toplevel
	}
	common := lines[1]
	// A bare repository's common dir is the repository itself.
	if filepath.Base(common) == ".git" {
		return filepath.Dir(common)
	}
	return common
}

// git runs one git command in dir and returns its trimmed stdout.
func git(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return "", fmt.Errorf("run git: %w", err)
		}
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", ErrNotRepo
	}
	return strings.TrimSpace(stdout.String()), nil
}

// BranchFinished reports whether work on branch in the repository at repo is over: the branch no
// longer exists, or it is merged into repo's HEAD and its tip commit was made at or after began (a
// branch cut from HEAD and not yet committed to is trivially an ancestor of it, and is not done).
// It returns an error, and false, when git cannot read the repository.
func BranchFinished(ctx context.Context, repo, branch string, began time.Time) (bool, error) {
	if _, err := git(ctx, repo, "rev-parse", "--git-dir"); err != nil {
		return false, err
	}
	// refs/heads/ keeps a branch name from being read as an option.
	tip, err := git(ctx, repo, "rev-parse", "--verify", "--quiet", "refs/heads/"+branch+"^{commit}")
	if err != nil || tip == "" {
		if ctx.Err() != nil {
			return false, ctx.Err()
		}
		return true, nil
	}
	if _, err := git(ctx, repo, "merge-base", "--is-ancestor", tip, "HEAD"); err != nil {
		return false, ctx.Err()
	}
	secs, err := git(ctx, repo, "show", "-s", "--format=%ct", tip)
	if err != nil {
		return false, err
	}
	made, err := strconv.ParseInt(secs, 10, 64)
	if err != nil {
		return false, fmt.Errorf("read the date of %s: %w", tip, err)
	}
	return made >= began.Unix(), nil
}

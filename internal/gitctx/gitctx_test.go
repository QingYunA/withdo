package gitctx_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/QingYunA/adhd-todo/internal/gitctx"
	"github.com/QingYunA/adhd-todo/internal/testutil"
)

func resolve(t *testing.T, dir string) (*gitctx.Info, error) {
	t.Helper()
	return gitctx.Resolve(context.Background(), dir)
}

func TestResolveRepos(t *testing.T) {
	root := t.TempDir()
	remote := testutil.NewRepo(t, root, "widgets", testutil.RepoOptions{Remote: "git@github.com:acme/widgets.git", Branch: "feat/x"})
	local := testutil.NewRepo(t, root, "localonly", testutil.RepoOptions{Commit: true})
	gitlab := testutil.NewRepo(t, root, "gl", testutil.RepoOptions{Remote: "https://gitlab.com/acme/gl.git"})
	sub := filepath.Join(remote, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	detached := testutil.NewRepo(t, root, "det", testutil.RepoOptions{Commit: true})
	testutil.Git(t, detached, "checkout", "-q", "--detach")

	tests := []struct {
		name string
		dir  string
		want gitctx.Info
	}{
		{"github remote", remote, gitctx.Info{Toplevel: remote, RepoRoot: remote, Branch: "feat/x", RepoSlug: "acme/widgets", RepoName: "widgets"}},
		{"subdirectory resolves to toplevel", sub, gitctx.Info{Toplevel: remote, RepoRoot: remote, Branch: "feat/x", RepoSlug: "acme/widgets", RepoName: "widgets"}},
		{"no remote uses directory name", local, gitctx.Info{Toplevel: local, RepoRoot: local, Branch: "main", RepoName: "localonly"}},
		{"non-GitHub remote falls back to directory name", gitlab, gitctx.Info{Toplevel: gitlab, RepoRoot: gitlab, Branch: "main", RepoName: "gl"}},
		{"detached HEAD has no branch", detached, gitctx.Info{Toplevel: detached, RepoRoot: detached, RepoName: "det"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolve(t, tt.dir)
			if err != nil {
				t.Fatal(err)
			}
			if *got != tt.want {
				t.Errorf("got %+v, want %+v", *got, tt.want)
			}
		})
	}
}

func TestResolveWorktrees(t *testing.T) {
	root := t.TempDir()
	withRemote := testutil.NewRepo(t, root, "widgets", testutil.RepoOptions{Remote: "git@github.com:acme/widgets.git", Commit: true})
	noRemote := testutil.NewRepo(t, root, "localonly", testutil.RepoOptions{Commit: true})
	wtRemote := filepath.Join(root, "widgets-wt")
	wtLocal := filepath.Join(root, "localonly-wt")
	testutil.Git(t, withRemote, "worktree", "add", "-q", "-b", "feat/wt", wtRemote)
	testutil.Git(t, noRemote, "worktree", "add", "-q", "-b", "feat/wt", wtLocal)

	got, err := resolve(t, wtRemote)
	if err != nil {
		t.Fatal(err)
	}
	want := gitctx.Info{Toplevel: testutil.Real(t, wtRemote), RepoRoot: withRemote, Branch: "feat/wt", RepoSlug: "acme/widgets", RepoName: "widgets"}
	if *got != want {
		t.Errorf("remote worktree: got %+v, want %+v", *got, want)
	}

	got, err = resolve(t, wtLocal)
	if err != nil {
		t.Fatal(err)
	}
	want = gitctx.Info{Toplevel: testutil.Real(t, wtLocal), RepoRoot: noRemote, Branch: "feat/wt", RepoName: "localonly"}
	if *got != want {
		t.Errorf("remote-less worktree must map to the main repo: got %+v, want %+v", *got, want)
	}
}

func TestResolveBareRepoWorktree(t *testing.T) {
	root := t.TempDir()
	src := testutil.NewRepo(t, root, "src", testutil.RepoOptions{Commit: true})
	bare := filepath.Join(root, "widgets.git")
	testutil.Git(t, root, "clone", "-q", "--bare", src, bare)
	wt := filepath.Join(root, "widgets-wt")
	testutil.Git(t, bare, "worktree", "add", "-q", "-b", "feat/wt", wt)
	got, err := resolve(t, wt)
	if err != nil {
		t.Fatal(err)
	}
	if got.RepoRoot != testutil.Real(t, bare) || got.RepoName != "widgets.git" {
		t.Errorf("bare worktree: got %+v", *got)
	}
}

func TestResolveNotARepo(t *testing.T) {
	testutil.IsolateGit(t)
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(t.TempDir()))
	for name, dir := range map[string]string{
		"plain directory":   t.TempDir(),
		"missing directory": filepath.Join(t.TempDir(), "nope"),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := resolve(t, dir); !errors.Is(err, gitctx.ErrNotRepo) {
				t.Errorf("err = %v, want ErrNotRepo", err)
			}
		})
	}
}

func TestResolveGitMissing(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	_, err := resolve(t, t.TempDir())
	if err == nil || errors.Is(err, gitctx.ErrNotRepo) {
		t.Errorf("err = %v, want a git-not-found error", err)
	}
}

func TestResolveCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := gitctx.Resolve(ctx, t.TempDir()); !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

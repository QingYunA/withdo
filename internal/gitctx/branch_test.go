package gitctx_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/gitctx"
	"github.com/QingYunA/adhd-todo/internal/testutil"
)

func TestBranchFinished(t *testing.T) {
	root := t.TempDir()
	repo := testutil.NewRepo(t, root, "widgets", testutil.RepoOptions{Commit: true})
	began := time.Now().Add(time.Minute)
	finished := func(branch string) bool {
		t.Helper()
		got, err := gitctx.BranchFinished(context.Background(), repo, branch, began)
		if err != nil {
			t.Fatal(err)
		}
		return got
	}

	if !finished("never-existed") {
		t.Error("a branch that does not exist is finished")
	}
	testutil.Git(t, repo, "branch", "fresh")
	if finished("fresh") {
		t.Error("a branch cut from HEAD with no commit of its own is still being worked on")
	}
	wt := filepath.Join(root, "wt")
	testutil.Git(t, repo, "worktree", "add", "-q", "-b", "work", wt)
	t.Setenv("GIT_COMMITTER_DATE", began.Add(time.Minute).Format(time.RFC3339))
	if err := os.WriteFile(filepath.Join(wt, "a.txt"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	testutil.Git(t, wt, "add", ".")
	testutil.Git(t, wt, "commit", "-q", "-m", "a")
	if finished("work") {
		t.Error("a branch with commits not in HEAD is not finished")
	}
	testutil.Git(t, repo, "merge", "-q", "--no-edit", "work")
	if !finished("work") {
		t.Error("a branch with commits, merged into HEAD, is finished")
	}
	if !finished("--delete-everything") {
		t.Error("a name that looks like an option is just a missing branch")
	}

	if _, err := gitctx.BranchFinished(context.Background(), filepath.Join(root, "missing"), "work", began); err == nil {
		t.Error("an unreadable repository is an error, not a finished branch")
	}
}

// Package testutil holds helpers shared by the tests of other packages:
// throwaway git repositories and databases.
package testutil

import (
	"context"
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/QingYunA/adhd-todo/internal/store"
)

// IsolateGit keeps the developer's global git config out of the test.
func IsolateGit(t *testing.T) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")
}

// Git runs git in dir and returns trimmed stdout, failing the test on error.
func Git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@t"}, args...)...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

// RepoOptions configure NewRepo.
type RepoOptions struct {
	Remote string // origin URL; none when empty
	Branch string // initial branch; "main" when empty
	Commit bool   // make one commit (needed for worktrees and detached HEAD)
}

// NewRepo creates <root>/<name> as a git repository and returns its real path.
func NewRepo(t *testing.T, root, name string, opts RepoOptions) string {
	t.Helper()
	IsolateGit(t)
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	branch := opts.Branch
	if branch == "" {
		branch = "main"
	}
	Git(t, dir, "init", "-q", "-b", branch)
	if opts.Remote != "" {
		Git(t, dir, "remote", "add", "origin", opts.Remote)
	}
	if opts.Commit {
		if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		Git(t, dir, "add", ".")
		Git(t, dir, "commit", "-q", "-m", "init")
	}
	real, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	return real
}

// Real resolves symlinks (macOS temp dirs live behind /var -> /private/var).
func Real(t *testing.T, path string) string {
	t.Helper()
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	return real
}

// OpenDB opens a fresh migrated database in a temp dir and closes it with the test.
func OpenDB(t *testing.T) (*sql.DB, string) {
	t.Helper()
	return OpenDBAt(t, filepath.Join(t.TempDir(), "test.db"))
}

// OpenDBAt opens the database at path and closes it with the test.
func OpenDBAt(t *testing.T, path string) (*sql.DB, string) {
	t.Helper()
	db, err := store.Open(context.Background(), path)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db, path
}

package testutil

import (
	"testing"
)

func TestNewRepoAndOpenDB(t *testing.T) {
	root := t.TempDir()
	repo := NewRepo(t, root, "r", RepoOptions{Remote: "git@github.com:a/b.git", Branch: "feat/x", Commit: true})
	if got := Git(t, repo, "symbolic-ref", "--short", "HEAD"); got != "feat/x" {
		t.Fatalf("branch = %q", got)
	}
	if got := Git(t, repo, "remote", "get-url", "origin"); got != "git@github.com:a/b.git" {
		t.Fatalf("remote = %q", got)
	}
	if Real(t, repo) != repo {
		t.Fatal("NewRepo should return a real path")
	}
	db, _ := OpenDB(t)
	var v int
	if err := db.QueryRow("PRAGMA user_version").Scan(&v); err != nil || v != 3 {
		t.Fatalf("user_version = %d, err %v", v, err)
	}
}

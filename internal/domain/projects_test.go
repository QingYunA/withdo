package domain_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/testutil"
)

func resolve(t *testing.T, c *domain.Core, dir string) *domain.ResolvedProject {
	t.Helper()
	return must[*domain.ResolvedProject](t)(c.ResolveProjectFromCwd(ctx, dir))
}

func TestResolveProjectFromCwd(t *testing.T) {
	c, _ := newCore(t)
	root := t.TempDir()
	dir := testutil.NewRepo(t, root, "widgets", testutil.RepoOptions{Remote: "git@github.com:acme/widgets.git"})
	a, b := resolve(t, c, dir), resolve(t, c, dir)
	eq(t, "name", a.Project.Name, "widgets")
	eqPtr(t, "slug", a.Project.RepoSlug, ptr("acme/widgets"))
	eqPtr(t, "path", a.Project.RepoPath, &dir)
	eq(t, "same project reused", b.Project.ID, a.Project.ID)
	eq(t, "projects", len(must[[]domain.Project](t)(c.ListProjects(ctx))), 1)

	if got := resolve(t, c, t.TempDir()); got != nil {
		t.Errorf("non-git cwd should resolve to nil, got %+v", got)
	}
}

func TestWorktreesShareTheMainRepoProject(t *testing.T) {
	c, _ := newCore(t)
	root := t.TempDir()
	tests := []struct {
		name string
		opts testutil.RepoOptions
	}{
		{"with remote", testutil.RepoOptions{Remote: "git@github.com:acme/widgets.git", Commit: true}},
		{"without remote", testutil.RepoOptions{Commit: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			main := testutil.NewRepo(t, root, "repo-"+filepath.Base(t.Name()), tt.opts)
			wt := filepath.Join(root, "wt-"+filepath.Base(t.Name()))
			testutil.Git(t, main, "worktree", "add", "-q", "-b", "feat/a", wt)
			a, b := resolve(t, c, main), resolve(t, c, wt)
			eq(t, "project id", b.Project.ID, a.Project.ID)
			eqPtr(t, "repo_path", b.Project.RepoPath, &main)
			eq(t, "branch", b.Git.Branch, "feat/a")
		})
	}
	eq(t, "projects", len(must[[]domain.Project](t)(c.ListProjects(ctx))), 2)
}

func TestRemoteLessReposAreKeyedByPath(t *testing.T) {
	c, _ := newCore(t)
	root := t.TempDir()
	one := testutil.NewRepo(t, root, "one", testutil.RepoOptions{})
	two := testutil.NewRepo(t, root, "two", testutil.RepoOptions{})
	a, b := resolve(t, c, one), resolve(t, c, two)
	if a.Project.RepoSlug != nil || a.Project.ID == b.Project.ID {
		t.Errorf("distinct remote-less repos need distinct projects: %+v %+v", a.Project, b.Project)
	}
	eq(t, "stable", resolve(t, c, one).Project.ID, a.Project.ID)
}

func TestSlugProjectFollowsMovedCheckout(t *testing.T) {
	c, _ := newCore(t)
	p := mustProject(t, c, "acme/widgets") // repo_path unset
	dir := testutil.NewRepo(t, t.TempDir(), "widgets", testutil.RepoOptions{Remote: "https://github.com/acme/widgets.git"})
	got := resolve(t, c, dir)
	eq(t, "id", got.Project.ID, p.ID)
	eqPtr(t, "repo_path filled in", got.Project.RepoPath, &dir)
}

func TestResolveProjectFromCwdReportsGitFailure(t *testing.T) {
	c, _ := newCore(t)
	t.Setenv("PATH", t.TempDir())
	if _, err := c.ResolveProjectFromCwd(ctx, t.TempDir()); err == nil {
		t.Error("a missing git binary must be an error, not 'not a repo'")
	}
}

func TestUpsertProject(t *testing.T) {
	c, _ := newCore(t)
	dir := testutil.NewRepo(t, t.TempDir(), "widgets", testutil.RepoOptions{Remote: "https://github.com/acme/widgets.git"})
	a := must[*domain.Project](t)(c.UpsertProject(ctx, domain.UpsertProjectInput{Cwd: dir}))
	b := must[*domain.Project](t)(c.UpsertProject(ctx, domain.UpsertProjectInput{Cwd: dir, Name: "Widgets App"}))
	eq(t, "same id", b.ID, a.ID)
	eq(t, "renamed", b.Name, "Widgets App")
	same := must[*domain.Project](t)(c.UpsertProject(ctx, domain.UpsertProjectInput{Cwd: dir, Name: "Widgets App"}))
	eq(t, "no-op rename keeps updated_at", same.UpdatedAt, b.UpdatedAt)

	s := must[*domain.Project](t)(c.UpsertProject(ctx, domain.UpsertProjectInput{RepoSlug: "acme/api"}))
	eq(t, "slug project name", s.Name, "api")
	if s.RepoPath != nil {
		t.Error("slug-only project has no path")
	}
	r := must[*domain.Project](t)(c.UpsertProject(ctx, domain.UpsertProjectInput{RepoSlug: "acme/api", Name: "API"}))
	eq(t, "slug rename", r.Name, "API")
	eq(t, "projects", len(must[[]domain.Project](t)(c.ListProjects(ctx))), 2)
}

func TestUpsertProjectErrors(t *testing.T) {
	c, _ := newCore(t)
	tests := []struct {
		name  string
		in    domain.UpsertProjectInput
		field string
	}{
		{"neither", domain.UpsertProjectInput{}, "cwd"},
		{"both", domain.UpsertProjectInput{Cwd: "/x", RepoSlug: "a/b"}, "cwd"},
		{"bad slug", domain.UpsertProjectInput{RepoSlug: "nope"}, "repo_slug"},
		{"non-git cwd", domain.UpsertProjectInput{Cwd: t.TempDir()}, "cwd"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := c.UpsertProject(ctx, tt.in)
			wantFieldError(t, err, tt.field)
		})
	}
}

func TestCheckCwd(t *testing.T) {
	dir := t.TempDir()
	if err := domain.CheckCwd(dir); err != nil {
		t.Errorf("existing directory: %v", err)
	}
	if err := domain.CheckCwd(""); err != nil {
		t.Errorf("absent cwd: %v", err)
	}
	missing := dir + "/nope"
	err := domain.CheckCwd(missing)
	wantFieldError(t, err, "cwd")
	if !strings.Contains(err.Error(), "no such directory: "+missing) {
		t.Errorf("error = %q", err)
	}
	file := dir + "/file"
	if werr := os.WriteFile(file, nil, 0o600); werr != nil {
		t.Fatal(werr)
	}
	wantFieldError(t, domain.CheckCwd(file), "cwd")

	c, _ := newCore(t)
	_, err = c.UpsertProject(ctx, domain.UpsertProjectInput{Cwd: missing})
	if err == nil || !strings.Contains(err.Error(), "no such directory") {
		t.Errorf("upsert with a missing cwd: %v", err)
	}
}

func TestResolveProjectID(t *testing.T) {
	c, _ := newCore(t)
	p := mustProject(t, c, "a/b")
	dir := testutil.NewRepo(t, t.TempDir(), "r", testutil.RepoOptions{Remote: "git@github.com:a/r.git"})

	got := must[*int64](t)(c.ResolveProjectID(ctx, &p.ID, ""))
	eqPtr(t, "explicit id", got, &p.ID)
	_, err := c.ResolveProjectID(ctx, ptr(int64(999)), "")
	wantFieldError(t, err, "project_id")
	if got := must[*int64](t)(c.ResolveProjectID(ctx, nil, dir)); got == nil {
		t.Error("cwd inside a repo should resolve")
	}
	if got := must[*int64](t)(c.ResolveProjectID(ctx, nil, t.TempDir())); got != nil {
		t.Error("non-git cwd resolves to the inbox")
	}
	if got := must[*int64](t)(c.ResolveProjectID(ctx, nil, "")); got != nil {
		t.Error("nothing given resolves to the inbox")
	}
}

func TestGetProject(t *testing.T) {
	c, _ := newCore(t)
	p := mustProject(t, c, "a/b")
	eq(t, "found", must[*domain.Project](t)(c.GetProject(ctx, p.ID)).ID, p.ID)
	if must[*domain.Project](t)(c.GetProject(ctx, 99)) != nil {
		t.Error("unknown project should be nil")
	}
}

func TestMilestones(t *testing.T) {
	c, _ := newCore(t)
	p := mustProject(t, c, "a/b")
	due := "2026-12-01T00:00:00.000Z"
	m := must[*domain.Milestone](t)(c.CreateMilestone(ctx, domain.CreateMilestoneInput{ProjectID: p.ID, Title: "v1", DueAt: &due}))
	eq(t, "status", m.Status, "open")
	eqPtr(t, "due", m.DueAt, &due)
	list := must[[]domain.Milestone](t)(c.ListMilestones(ctx, p.ID))
	if len(list) != 1 || list[0] != *m && list[0].ID != m.ID {
		t.Errorf("list = %+v", list)
	}
	_, err := c.CreateMilestone(ctx, domain.CreateMilestoneInput{ProjectID: 5, Title: "x"})
	wantFieldError(t, err, "project_id")
	_, err = c.CreateMilestone(ctx, domain.CreateMilestoneInput{ProjectID: p.ID})
	wantFieldError(t, err, "title")
}

func TestListActiveProjects(t *testing.T) {
	c, _, db := newCoreDB(t)
	a, b := mustProject(t, c, "acme/a"), mustProject(t, c, "acme/b")
	for _, in := range []domain.CreateTaskInput{
		{ProjectID: &a.ID, Status: domain.StatusDoing}, {ProjectID: &a.ID, Status: domain.StatusDoing},
		{ProjectID: &a.ID, Status: domain.StatusDone}, {},
	} {
		mustTask(t, c, in)
	}
	exec(t, db, "UPDATE projects SET archived_at = 'x' WHERE id = ?", b.ID)
	list := must[[]domain.ProjectWithCounts](t)(c.ListActiveProjects(ctx))
	if len(list) != 1 || list[0].ID != a.ID {
		t.Fatalf("list = %+v", list)
	}
	want := map[domain.Status]int{"backlog": 0, "todo": 0, "doing": 2, "blocked": 0, "review": 0, "done": 1, "cancelled": 0}
	for s, n := range want {
		if list[0].Counts[s] != n {
			t.Errorf("count[%s] = %d, want %d", s, list[0].Counts[s], n)
		}
	}
}

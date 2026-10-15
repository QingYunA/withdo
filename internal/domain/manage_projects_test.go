package domain_test

import (
	"testing"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

func TestCreateProjectTakesANameAndAnOptionalRepository(t *testing.T) {
	c, _ := newCore(t)
	plain := must[*domain.Project](t)(c.CreateProject(ctx, domain.CreateProjectInput{Name: "  Garden  "}))
	eq(t, "name trimmed", plain.Name, "Garden")
	if plain.RepoSlug != nil || plain.RepoPath != nil || plain.ArchivedAt != nil {
		t.Errorf("a project without a repository has no slug, path or archive time: %+v", plain)
	}
	linked := must[*domain.Project](t)(c.CreateProject(ctx, domain.CreateProjectInput{Name: "Widgets", RepoSlug: "acme/widgets"}))
	eqPtr(t, "slug", linked.RepoSlug, ptr("acme/widgets"))
	// The slug is what an agent's report finds the project by.
	again := must[*domain.Project](t)(c.UpsertProject(ctx, domain.UpsertProjectInput{RepoSlug: "acme/widgets"}))
	eq(t, "upsert finds it", again.ID, linked.ID)
}

func TestCreateProjectAcceptsAPastedGitHubAddress(t *testing.T) {
	c, _ := newCore(t)
	for _, in := range []string{"https://github.com/acme/rocket", "git@github.com:acme/rocket.git", " acme/rocket "} {
		p, err := c.CreateProject(ctx, domain.CreateProjectInput{Name: "Rocket " + in, RepoSlug: in})
		if in == "https://github.com/acme/rocket" {
			if err != nil {
				t.Fatal(err)
			}
			eqPtr(t, "slug from a URL", p.RepoSlug, ptr("acme/rocket"))
			continue
		}
		if !domain.IsConflict(err) {
			t.Errorf("%q should be read as acme/rocket, already taken; got %v", in, err)
		}
	}
}

func TestCreateProjectRejectsWhatIsWrong(t *testing.T) {
	c, _ := newCore(t)
	mustProject(t, c, "acme/widgets")
	tests := []struct {
		name  string
		in    domain.CreateProjectInput
		field string
		taken bool
	}{
		{"blank name", domain.CreateProjectInput{Name: "   "}, "name", false},
		{"not owner/repo", domain.CreateProjectInput{Name: "x", RepoSlug: "widgets"}, "repo_slug", false},
		{"too many parts", domain.CreateProjectInput{Name: "x", RepoSlug: "a/b/c"}, "repo_slug", false},
		{"already used", domain.CreateProjectInput{Name: "x", RepoSlug: "acme/widgets"}, "repo_slug", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := c.CreateProject(ctx, tt.in)
			wantFieldError(t, err, tt.field)
			eq(t, "conflict", domain.IsConflict(err), tt.taken)
		})
	}
	eq(t, "nothing was created", len(must[[]domain.Project](t)(c.ListProjects(ctx))), 1)
}

func TestRenameProject(t *testing.T) {
	c, _ := newCore(t)
	p := mustProject(t, c, "acme/widgets")
	got := must[*domain.Project](t)(c.RenameProject(ctx, p.ID, "  Widget factory "))
	eq(t, "name", got.Name, "Widget factory")
	_, err := c.RenameProject(ctx, p.ID, " ")
	wantFieldError(t, err, "name")
	_, err = c.RenameProject(ctx, 999, "x")
	if !domain.IsNotFound(err) {
		t.Errorf("a missing project is not found: %v", err)
	}
}

func TestArchiveAndRestoreProject(t *testing.T) {
	c, clk := newCore(t)
	p := mustProject(t, c, "acme/widgets")
	task := mustTask(t, c, domain.CreateTaskInput{ProjectID: &p.ID})
	clk.advance(1)
	archived := must[*domain.Project](t)(c.ArchiveProject(ctx, p.ID))
	if archived.ArchivedAt == nil {
		t.Fatal("archived_at not set")
	}
	eq(t, "active projects", len(must[[]domain.ProjectWithCounts](t)(c.ListActiveProjects(ctx))), 0)
	if got := must[*domain.Task](t)(c.GetTask(ctx, task.ID)); got == nil {
		t.Error("archiving keeps the tasks")
	}
	again := must[*domain.Project](t)(c.ArchiveProject(ctx, p.ID))
	eqPtr(t, "archiving twice keeps the first time", again.ArchivedAt, archived.ArchivedAt)

	restored := must[*domain.Project](t)(c.RestoreProject(ctx, p.ID))
	if restored.ArchivedAt != nil {
		t.Error("archived_at not cleared")
	}
	eq(t, "active projects", len(must[[]domain.ProjectWithCounts](t)(c.ListActiveProjects(ctx))), 1)
	for _, f := range []func(int64) error{
		func(id int64) error { _, err := c.ArchiveProject(ctx, id); return err },
		func(id int64) error { _, err := c.RestoreProject(ctx, id); return err },
	} {
		if err := f(999); !domain.IsNotFound(err) {
			t.Errorf("a missing project is not found: %v", err)
		}
	}
}

func TestDeleteProjectRemovesEverythingAndCountsIt(t *testing.T) {
	c, _, db := newCoreDB(t)
	p := mustProject(t, c, "acme/widgets")
	other := mustProject(t, c, "acme/other")
	m := must[*domain.Milestone](t)(c.CreateMilestone(ctx, domain.CreateMilestoneInput{ProjectID: p.ID, Title: "v1"}))
	must[*domain.Milestone](t)(c.CreateMilestone(ctx, domain.CreateMilestoneInput{ProjectID: p.ID, Title: "v2"}))
	a := mustTask(t, c, domain.CreateTaskInput{ProjectID: &p.ID, MilestoneID: &m.ID})
	b := mustTask(t, c, domain.CreateTaskInput{ProjectID: &p.ID, Status: domain.StatusDone})
	keep := mustTask(t, c, domain.CreateTaskInput{ProjectID: &other.ID})
	inbox := mustTask(t, c, domain.CreateTaskInput{})
	for _, id := range []int64{a.ID, a.ID, b.ID, keep.ID} {
		must[*domain.Report](t)(c.AddReport(ctx, domain.AddReportInput{TaskID: id, Agent: "x", Summary: "s"}))
	}

	got, err := c.DeleteProject(ctx, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	eq(t, "tasks", got.Tasks, 2)
	eq(t, "reports", got.Reports, 3)
	eq(t, "milestones", got.Milestones, 2)

	if p2 := must[*domain.Project](t)(c.GetProject(ctx, p.ID)); p2 != nil {
		t.Error("the project is still there")
	}
	for _, id := range []int64{a.ID, b.ID} {
		if task := must[*domain.Task](t)(c.GetTask(ctx, id)); task != nil {
			t.Errorf("task %d is still there", id)
		}
	}
	for _, id := range []int64{keep.ID, inbox.ID} {
		if task := must[*domain.Task](t)(c.GetTask(ctx, id)); task == nil {
			t.Errorf("task %d of another project was deleted", id)
		}
	}
	eq(t, "other project's reports", len(must[[]domain.Report](t)(c.ListReports(ctx, keep.ID))), 1)
	var orphans int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM reports WHERE task_id NOT IN (SELECT id FROM tasks)").Scan(&orphans); err != nil {
		t.Fatal(err)
	}
	eq(t, "orphaned reports", orphans, 0)
	eq(t, "other project's milestones", len(must[[]domain.Milestone](t)(c.ListMilestones(ctx, other.ID))), 0)
}

func TestDeleteProjectIsAllOrNothing(t *testing.T) {
	c, _, db := newCoreDB(t)
	p := mustProject(t, c, "acme/widgets")
	must[*domain.Milestone](t)(c.CreateMilestone(ctx, domain.CreateMilestoneInput{ProjectID: p.ID, Title: "v1"}))
	task := mustTask(t, c, domain.CreateTaskInput{ProjectID: &p.ID})
	must[*domain.Report](t)(c.AddReport(ctx, domain.AddReportInput{TaskID: task.ID, Agent: "x", Summary: "s"}))
	// The last step before the project row fails; the reports and tasks gone by then come back.
	exec(t, db, `CREATE TRIGGER no_milestone_delete BEFORE DELETE ON milestones BEGIN SELECT RAISE(ABORT, 'nope'); END`)

	if _, err := c.DeleteProject(ctx, p.ID); err == nil {
		t.Fatal("expected the delete to fail")
	}
	if must[*domain.Project](t)(c.GetProject(ctx, p.ID)) == nil || must[*domain.Task](t)(c.GetTask(ctx, task.ID)) == nil {
		t.Error("the project or its task was lost")
	}
	eq(t, "reports kept", len(must[[]domain.Report](t)(c.ListReports(ctx, task.ID))), 1)
	eq(t, "milestones kept", len(must[[]domain.Milestone](t)(c.ListMilestones(ctx, p.ID))), 1)

	_, err := c.DeleteProject(ctx, 999)
	if !domain.IsNotFound(err) {
		t.Errorf("a missing project is not found: %v", err)
	}
}

func TestAReportFromAnAgentRestoresAnArchivedProject(t *testing.T) {
	c, _ := newCore(t)
	dir := widgetsRepo(t, "feat/login")
	first := progress(t, c, domain.ProgressInput{Summary: "one", Cwd: dir})
	pid := *first.Task.ProjectID

	archive := func() {
		must[*domain.Project](t)(c.ArchiveProject(ctx, pid))
		if len(must[[]domain.ProjectWithCounts](t)(c.ListActiveProjects(ctx))) != 0 {
			t.Fatal("not archived")
		}
	}
	archive()
	progress(t, c, domain.ProgressInput{Summary: "two", Cwd: dir}) // found by repository
	eq(t, "restored by cwd", len(must[[]domain.ProjectWithCounts](t)(c.ListActiveProjects(ctx))), 1)

	archive()
	id := first.Task.ID
	progress(t, c, domain.ProgressInput{Summary: "three", TaskID: &id}) // found by task id
	eq(t, "restored by task id", len(must[[]domain.ProjectWithCounts](t)(c.ListActiveProjects(ctx))), 1)

	// Anything that is not a report leaves it archived.
	archive()
	must[*domain.Task](t)(c.UpdateTask(ctx, id, domain.TaskPatch{Priority: ptr(domain.P1)}))
	eq(t, "still archived", len(must[[]domain.ProjectWithCounts](t)(c.ListActiveProjects(ctx))), 0)
}

package domain_test

import (
	"testing"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

func subproject(t *testing.T, c *domain.Core, name string, parent *domain.Project) *domain.Project {
	t.Helper()
	return must[*domain.Project](t)(c.CreateProject(ctx, domain.CreateProjectInput{Name: name, ParentID: &parent.ID}))
}

func TestCreateProjectUnderAnotherProject(t *testing.T) {
	c, _ := newCore(t)
	root := must[*domain.Project](t)(c.CreateProject(ctx, domain.CreateProjectInput{Name: "adhd-todo"}))
	app := subproject(t, c, "App", root)
	ui := subproject(t, c, "UI", app)
	eqPtr(t, "the sub-project's parent", app.ParentID, &root.ID)
	eqPtr(t, "a sub-project can have one of its own", ui.ParentID, &app.ID)

	ps := must[[]domain.Project](t)(c.ListProjects(ctx))
	eq(t, "all three are projects", len(ps), 3)
	for _, p := range ps {
		if p.ID == root.ID && p.ParentID != nil {
			t.Errorf("the top-level project has a parent: %+v", p)
		}
	}
	if got := must[*domain.Project](t)(c.GetProject(ctx, app.ID)); got == nil || got.ParentID == nil {
		t.Errorf("GetProject loses the parent of %q: %+v", app.Name, got)
	}
}

func TestCreateProjectRejectsAParentThatCannotHoldIt(t *testing.T) {
	c, _ := newCore(t)
	root := must[*domain.Project](t)(c.CreateProject(ctx, domain.CreateProjectInput{Name: "root"}))
	tests := []struct {
		name   string
		parent int64
	}{
		{"missing", 999},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := c.CreateProject(ctx, domain.CreateProjectInput{Name: "child", ParentID: &tt.parent})
			if !domain.IsNotFound(err) {
				t.Errorf("a missing parent is not found: %v", err)
			}
		})
	}
	must[*domain.Project](t)(c.ArchiveProject(ctx, root.ID))
	archived := root.ID
	_, err := c.CreateProject(ctx, domain.CreateProjectInput{Name: "child", ParentID: &archived})
	wantFieldError(t, err, "parent_id")
	eq(t, "nothing was created", len(must[[]domain.Project](t)(c.ListProjects(ctx))), 1)
}

func TestArchivingAProjectTakesItsSubprojects(t *testing.T) {
	c, _ := newCore(t)
	root := must[*domain.Project](t)(c.CreateProject(ctx, domain.CreateProjectInput{Name: "adhd-todo"}))
	app := subproject(t, c, "App", root)
	ui := subproject(t, c, "UI", app)
	other := must[*domain.Project](t)(c.CreateProject(ctx, domain.CreateProjectInput{Name: "other"}))

	must[*domain.Project](t)(c.ArchiveProject(ctx, root.ID))
	for _, p := range []*domain.Project{root, app, ui} {
		got := must[*domain.Project](t)(c.GetProject(ctx, p.ID))
		if got.ArchivedAt == nil {
			t.Errorf("%q is still active after its parent was archived", p.Name)
		}
	}
	eq(t, "only the other project is active", len(must[[]domain.ProjectWithCounts](t)(c.ListActiveProjects(ctx))), 1)
	if got := must[*domain.Project](t)(c.GetProject(ctx, other.ID)); got.ArchivedAt != nil {
		t.Error("a sibling project was archived too")
	}

	again := must[*domain.Project](t)(c.ArchiveProject(ctx, root.ID))
	appGot := must[*domain.Project](t)(c.GetProject(ctx, app.ID))
	eqPtr(t, "archiving twice keeps the first time", again.ArchivedAt, appGot.ArchivedAt)

	must[*domain.Project](t)(c.RestoreProject(ctx, root.ID))
	for _, p := range []*domain.Project{root, app, ui} {
		if got := must[*domain.Project](t)(c.GetProject(ctx, p.ID)); got.ArchivedAt != nil {
			t.Errorf("%q did not come back with its parent", p.Name)
		}
	}
}

func TestRestoringASubprojectBringsTheProjectItHangsUnder(t *testing.T) {
	c, _ := newCore(t)
	root := must[*domain.Project](t)(c.CreateProject(ctx, domain.CreateProjectInput{Name: "adhd-todo"}))
	app := subproject(t, c, "App", root)
	ui := subproject(t, c, "UI", app)
	must[*domain.Project](t)(c.ArchiveProject(ctx, root.ID))

	must[*domain.Project](t)(c.RestoreProject(ctx, app.ID))
	for _, p := range []*domain.Project{root, app, ui} {
		if got := must[*domain.Project](t)(c.GetProject(ctx, p.ID)); got.ArchivedAt != nil {
			t.Errorf("%q is not visible: a restored sub-project needs its parent and its own sub-projects", p.Name)
		}
	}
}

func TestDeletingAProjectLeavesItsSubprojectsTopLevel(t *testing.T) {
	c, _ := newCore(t)
	root := must[*domain.Project](t)(c.CreateProject(ctx, domain.CreateProjectInput{Name: "adhd-todo"}))
	app := subproject(t, c, "App", root)
	ui := subproject(t, c, "UI", app)
	task := mustTask(t, c, domain.CreateTaskInput{ProjectID: &app.ID, Title: "keep me"})

	must[domain.Deleted](t)(c.DeleteProject(ctx, root.ID))
	if got := must[*domain.Project](t)(c.GetProject(ctx, app.ID)); got == nil || got.ParentID != nil {
		t.Errorf("the sub-project should stand on its own: %+v", got)
	}
	eqPtr(t, "the sub-project below it keeps its parent", must[*domain.Project](t)(c.GetProject(ctx, ui.ID)).ParentID, &app.ID)
	if got := must[*domain.Task](t)(c.GetTask(ctx, task.ID)); got == nil {
		t.Error("the sub-project's task went with the deleted parent")
	}
}

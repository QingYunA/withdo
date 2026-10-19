package domain_test

import (
	"testing"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

func TestListTasksHideUserInbox(t *testing.T) {
	c, _ := newCore(t)
	w := mustProject(t, c, "acme/widgets")
	mine := mustTask(t, c, domain.CreateTaskInput{Title: "mine"})
	handed := mustTask(t, c, domain.CreateTaskInput{Title: "handed"})
	must[*domain.Task](t)(c.HandOff(ctx, handed.ID))
	mustTask(t, c, domain.CreateTaskInput{Title: "repo", ProjectID: &w.ID})
	mustTask(t, c, domain.CreateTaskInput{Title: "agent inbox", Source: domain.SourceAgent})

	all := must[[]domain.Task](t)(c.ListTasks(ctx, domain.TaskFilter{}))
	eq(t, "unfiltered", len(all), 4)
	vis := must[[]domain.Task](t)(c.ListTasks(ctx, domain.TaskFilter{HideUserInbox: true}))
	eq(t, "hidden", len(vis), 3)
	for _, k := range vis {
		if k.ID == mine.ID {
			t.Errorf("user inbox task leaked: %+v", k)
		}
	}
	must[*domain.Task](t)(c.TakeBack(ctx, handed.ID))
	vis = must[[]domain.Task](t)(c.ListTasks(ctx, domain.TaskFilter{HideUserInbox: true}))
	eq(t, "after take back", len(vis), 2)
}

func TestTaskUserPrivate(t *testing.T) {
	id := int64(1)
	eq(t, "user inbox", domain.Task{Source: domain.SourceUser}.UserPrivate(), true)
	eq(t, "user project", domain.Task{Source: domain.SourceUser, ProjectID: &id}.UserPrivate(), false)
	eq(t, "agent inbox", domain.Task{Source: domain.SourceAgent}.UserPrivate(), false)
}

func TestOverviewHideUserInbox(t *testing.T) {
	c, _ := newCore(t)
	mustTask(t, c, domain.CreateTaskInput{Title: "mine", Status: domain.StatusDoing})
	h := mustTask(t, c, domain.CreateTaskInput{Title: "handed"})
	must[*domain.Task](t)(c.HandOff(ctx, h.ID))
	must[*domain.Task](t)(c.UpdateTask(ctx, h.ID, domain.TaskPatch{Status: ptrTo(domain.StatusDoing)}))
	o := must[*domain.Overview](t)(c.GetOverview(ctx, domain.OverviewOptions{HideUserInbox: true}))
	eq(t, "groups", len(o.Projects), 1)
	eq(t, "tasks", len(o.Projects[0].Tasks), 1)
	eq(t, "title", o.Projects[0].Tasks[0].Title, "handed")
	o = must[*domain.Overview](t)(c.GetOverview(ctx, domain.OverviewOptions{}))
	eq(t, "unfiltered tasks", len(o.Projects[0].Tasks), 2)
}

func ptrTo[T any](v T) *T { return &v }

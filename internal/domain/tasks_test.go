package domain_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

func TestParsePRNumber(t *testing.T) {
	tests := []struct {
		url  string
		want int64
		ok   bool
	}{
		{"https://github.com/acme/widgets/pull/42", 42, true},
		{"https://github.com/acme/widgets/pull/42/files", 42, true},
		{"http://github.com/acme/widgets/pull/42?x=1", 42, true},
		{"  https://github.com/acme/widgets/pull/42#c  ", 42, true},
		{"https://github.com/acme/widgets/issues/42", 0, false},
		{"https://gitlab.com/acme/widgets/pull/42", 0, false},
		{"not a url", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			n, ok := domain.ParsePRNumber(tt.url)
			if n != tt.want || ok != tt.ok {
				t.Errorf("got (%d, %v), want (%d, %v)", n, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestCreateTaskDefaults(t *testing.T) {
	c, _ := newCore(t)
	p := mustProject(t, c, "acme/widgets")
	task := mustTask(t, c, domain.CreateTaskInput{Title: "Write docs", ProjectID: &p.ID})
	eq(t, "status", task.Status, domain.StatusTodo)
	eq(t, "priority", task.Priority, domain.P2)
	eq(t, "source", task.Source, domain.SourceUser)
	eq(t, "created_at", task.CreatedAt, "2026-10-01T00:00:00.000Z")
	if task.CompletedAt != nil || task.PRNumber != nil || task.RemindedAt != nil {
		t.Errorf("unexpected optional fields: %+v", task)
	}
	got := must[*domain.Task](t)(c.GetTask(ctx, task.ID))
	if !reflect.DeepEqual(got, task) {
		t.Errorf("GetTask = %+v, want %+v", got, task)
	}
	if missing := must[*domain.Task](t)(c.GetTask(ctx, 999)); missing != nil {
		t.Errorf("GetTask(999) = %+v, want nil", missing)
	}
}

func TestCreateTaskVariants(t *testing.T) {
	c, _ := newCore(t)
	tests := []struct {
		name  string
		in    domain.CreateTaskInput
		check func(*testing.T, *domain.Task)
	}{
		{"inbox has no project", domain.CreateTaskInput{Title: "x"}, func(t *testing.T, k *domain.Task) {
			if k.ProjectID != nil {
				t.Error("project_id should be nil")
			}
		}},
		{"pr_url sets pr_number", domain.CreateTaskInput{Title: "x", Source: domain.SourceAgent, PRURL: ptr("https://github.com/a/b/pull/7")}, func(t *testing.T, k *domain.Task) {
			eqPtr(t, "pr_number", k.PRNumber, ptr(int64(7)))
		}},
		{"non-PR url keeps url without number", domain.CreateTaskInput{Title: "x", PRURL: ptr("https://example.com/x")}, func(t *testing.T, k *domain.Task) {
			if k.PRNumber != nil || k.PRURL == nil {
				t.Errorf("got %+v", k)
			}
		}},
		{"created as done is completed", domain.CreateTaskInput{Title: "x", Status: domain.StatusDone}, func(t *testing.T, k *domain.Task) {
			eqPtr(t, "completed_at", k.CompletedAt, &k.CreatedAt)
		}},
		{"reminder starts un-reminded", domain.CreateTaskInput{Title: "x", RemindAt: ptr("2026-01-01T09:00:00.000Z")}, func(t *testing.T, k *domain.Task) {
			if k.RemindedAt != nil {
				t.Error("reminded_at should be nil")
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { tt.check(t, mustTask(t, c, tt.in)) })
	}
}

func TestCreateTaskReferences(t *testing.T) {
	c, _ := newCore(t)
	p := mustProject(t, c, "acme/widgets")
	other := mustProject(t, c, "acme/other")
	m := must[*domain.Milestone](t)(c.CreateMilestone(ctx, domain.CreateMilestoneInput{ProjectID: p.ID, Title: "v1"}))

	_, err := c.CreateTask(ctx, domain.CreateTaskInput{Title: "x", ProjectID: &other.ID, MilestoneID: &m.ID})
	wantFieldError(t, err, "milestone_id")
	inferred := mustTask(t, c, domain.CreateTaskInput{Title: "x", MilestoneID: &m.ID})
	eqPtr(t, "inferred project", inferred.ProjectID, &p.ID)
	_, err = c.CreateTask(ctx, domain.CreateTaskInput{Title: "x", MilestoneID: ptr(int64(99))})
	wantFieldError(t, err, "milestone_id")
	_, err = c.CreateTask(ctx, domain.CreateTaskInput{Title: "x", ProjectID: ptr(int64(99))})
	wantFieldError(t, err, "project_id")
	if !domain.IsNotFound(err) {
		t.Error("unknown project should be a not-found error")
	}
}

func TestCreateTaskValidation(t *testing.T) {
	c, _ := newCore(t)
	tests := []struct {
		name  string
		in    domain.CreateTaskInput
		field string
	}{
		{"empty title", domain.CreateTaskInput{}, "title"},
		{"blank title", domain.CreateTaskInput{Title: " \t\n "}, "title"},
		{"bad status", domain.CreateTaskInput{Title: "x", Status: "weird"}, "status"},
		{"bad priority", domain.CreateTaskInput{Title: "x", Priority: "P9"}, "priority"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := c.CreateTask(ctx, tt.in)
			wantFieldError(t, err, tt.field)
			if domain.IsNotFound(err) {
				t.Error("validation errors are not not-found errors")
			}
		})
	}
}

func TestTitlesAreTrimmed(t *testing.T) {
	c, _ := newCore(t)
	task := mustTask(t, c, domain.CreateTaskInput{Title: "  Write docs\n"})
	eq(t, "created", task.Title, "Write docs")
	title := "\tShip it  "
	u := must[*domain.Task](t)(c.UpdateTask(ctx, task.ID, domain.TaskPatch{Title: &title}))
	eq(t, "updated", u.Title, "Ship it")
}

func TestUpdateTaskPatchesFields(t *testing.T) {
	c, clk := newCore(t)
	task := mustTask(t, c, domain.CreateTaskInput{Title: "a"})
	clk.advance(time.Minute)
	title, prio := "b", domain.P0
	u := must[*domain.Task](t)(c.UpdateTask(ctx, task.ID, domain.TaskPatch{Title: &title, Priority: &prio, Notes: domain.Set("n")}))
	eq(t, "title", u.Title, "b")
	eq(t, "priority", u.Priority, domain.P0)
	eq(t, "status unchanged", u.Status, domain.StatusTodo)
	eqPtr(t, "notes", u.Notes, ptr("n"))
	eq(t, "updated_at", u.UpdatedAt, "2026-10-01T00:01:00.000Z")
	eq(t, "original not mutated", task.Title, "a")
}

func TestUpdateTaskCompletedAt(t *testing.T) {
	c, clk := newCore(t)
	task := mustTask(t, c, domain.CreateTaskInput{Title: "a"})
	status := func(s domain.Status) domain.TaskPatch { return domain.TaskPatch{Status: &s} }
	clk.advance(time.Hour)
	done := must[*domain.Task](t)(c.UpdateTask(ctx, task.ID, status(domain.StatusDone)))
	eqPtr(t, "completed_at", done.CompletedAt, ptr("2026-10-01T01:00:00.000Z"))
	clk.advance(time.Hour)
	cancelled := must[*domain.Task](t)(c.UpdateTask(ctx, task.ID, status(domain.StatusCancelled)))
	eqPtr(t, "completed_at stays from first close", cancelled.CompletedAt, done.CompletedAt)
	reopened := must[*domain.Task](t)(c.UpdateTask(ctx, task.ID, status(domain.StatusDoing)))
	if reopened.CompletedAt != nil {
		t.Errorf("reopened completed_at = %v, want nil", *reopened.CompletedAt)
	}
}

func TestUpdateTaskNullableFields(t *testing.T) {
	c, _ := newCore(t)
	task := mustTask(t, c, domain.CreateTaskInput{Title: "a", Notes: ptr("keep"), DueAt: ptr("2026-01-01T00:00:00.000Z")})
	withPR := must[*domain.Task](t)(c.UpdateTask(ctx, task.ID, domain.TaskPatch{PRURL: domain.Set("https://github.com/a/b/pull/9")}))
	eqPtr(t, "pr_number", withPR.PRNumber, ptr(int64(9)))
	cleared := must[*domain.Task](t)(c.UpdateTask(ctx, task.ID, domain.TaskPatch{
		PRURL: domain.Clear[string](), Notes: domain.Clear[string](), DueAt: domain.Clear[string](),
	}))
	if cleared.PRURL != nil || cleared.PRNumber != nil || cleared.Notes != nil || cleared.DueAt != nil {
		t.Errorf("fields should be cleared: %+v", cleared)
	}
	untouched := must[*domain.Task](t)(c.UpdateTask(ctx, task.ID, domain.TaskPatch{}))
	eq(t, "empty patch keeps title", untouched.Title, "a")
}

func TestUpdateTaskErrors(t *testing.T) {
	c, _ := newCore(t)
	task := mustTask(t, c, domain.CreateTaskInput{Title: "a"})
	empty, blank, bad, badPrio := "", " \t ", domain.Status("nope"), domain.Priority("P9")
	tests := []struct {
		name  string
		id    int64
		patch domain.TaskPatch
		field string
	}{
		{"unknown task", 999, domain.TaskPatch{}, "task_id"},
		{"unknown project", task.ID, domain.TaskPatch{ProjectID: domain.Set(int64(999))}, "project_id"},
		{"unknown milestone", task.ID, domain.TaskPatch{MilestoneID: domain.Set(int64(999))}, "milestone_id"},
		{"empty title", task.ID, domain.TaskPatch{Title: &empty}, "title"},
		{"blank title", task.ID, domain.TaskPatch{Title: &blank}, "title"},
		{"bad status", task.ID, domain.TaskPatch{Status: &bad}, "status"},
		{"bad priority", task.ID, domain.TaskPatch{Priority: &badPrio}, "priority"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := c.UpdateTask(ctx, tt.id, tt.patch)
			wantFieldError(t, err, tt.field)
		})
	}
}

func TestUpdateTaskMovesBetweenProjects(t *testing.T) {
	c, _ := newCore(t)
	p := mustProject(t, c, "acme/widgets")
	task := mustTask(t, c, domain.CreateTaskInput{Title: "a"})
	m := must[*domain.Milestone](t)(c.CreateMilestone(ctx, domain.CreateMilestoneInput{ProjectID: p.ID, Title: "v1"}))
	u := must[*domain.Task](t)(c.UpdateTask(ctx, task.ID, domain.TaskPatch{ProjectID: domain.Set(p.ID), MilestoneID: domain.Set(m.ID)}))
	eqPtr(t, "project", u.ProjectID, &p.ID)
	eqPtr(t, "milestone", u.MilestoneID, &m.ID)
	inbox := must[*domain.Task](t)(c.UpdateTask(ctx, task.ID, domain.TaskPatch{ProjectID: domain.Clear[int64](), MilestoneID: domain.Clear[int64]()}))
	if inbox.ProjectID != nil || inbox.MilestoneID != nil {
		t.Errorf("task should be back in the inbox: %+v", inbox)
	}
}

func TestRemindedAtRules(t *testing.T) {
	c, _ := newCore(t)
	const remind = "2026-01-01T09:00:00.000Z"
	task := mustTask(t, c, domain.CreateTaskInput{Title: "a", RemindAt: ptr(remind)})
	arm := func() {
		t.Helper()
		if _, err := c.ClaimDueReminders(ctx, t0); err != nil {
			t.Fatal(err)
		}
	}
	reminded := func() *string { return must[*domain.Task](t)(c.GetTask(ctx, task.ID)).RemindedAt }

	arm()
	if reminded() == nil {
		t.Fatal("claiming should set reminded_at")
	}
	for name, patch := range map[string]domain.TaskPatch{
		"same value": {RemindAt: domain.Set(remind)},
		"new value":  {RemindAt: domain.Set("2026-02-02T09:00:00.000Z")},
		"cleared":    {RemindAt: domain.Clear[string]()},
	} {
		arm()
		if _, err := c.UpdateTask(ctx, task.ID, patch); err != nil {
			t.Fatal(err)
		}
		if reminded() != nil {
			t.Errorf("%s: writing remind_at must re-arm the reminder", name)
		}
	}
	// Setting it back and claiming, then an unrelated update leaves reminded_at alone.
	if _, err := c.UpdateTask(ctx, task.ID, domain.TaskPatch{RemindAt: domain.Set(remind)}); err != nil {
		t.Fatal(err)
	}
	arm()
	title := "b"
	if _, err := c.UpdateTask(ctx, task.ID, domain.TaskPatch{Title: &title}); err != nil {
		t.Fatal(err)
	}
	if reminded() == nil {
		t.Error("unrelated updates must not reset reminded_at")
	}
}

func TestListTasksFilters(t *testing.T) {
	c, _ := newCore(t)
	p := mustProject(t, c, "acme/widgets")
	a := mustTask(t, c, domain.CreateTaskInput{Title: "a", ProjectID: &p.ID, Status: domain.StatusDoing})
	b := mustTask(t, c, domain.CreateTaskInput{Title: "b", ProjectID: &p.ID, Status: domain.StatusTodo})
	inbox := mustTask(t, c, domain.CreateTaskInput{Title: "c", Status: domain.StatusDoing})
	m := must[*domain.Milestone](t)(c.CreateMilestone(ctx, domain.CreateMilestoneInput{ProjectID: p.ID, Title: "v1"}))
	d := mustTask(t, c, domain.CreateTaskInput{Title: "d", MilestoneID: &m.ID, Status: domain.StatusReview})

	ids := func(f domain.TaskFilter) []int64 {
		var out []int64
		for _, k := range must[[]domain.Task](t)(c.ListTasks(ctx, f)) {
			out = append(out, k.ID)
		}
		return out
	}
	tests := []struct {
		name string
		f    domain.TaskFilter
		want []int64
	}{
		{"all newest first", domain.TaskFilter{}, []int64{d.ID, inbox.ID, b.ID, a.ID}},
		{"project", domain.TaskFilter{ProjectID: &p.ID}, []int64{d.ID, b.ID, a.ID}},
		{"status", domain.TaskFilter{Statuses: []domain.Status{domain.StatusDoing}}, []int64{inbox.ID, a.ID}},
		{"empty status list means all", domain.TaskFilter{Statuses: []domain.Status{}}, []int64{d.ID, inbox.ID, b.ID, a.ID}},
		{"limit", domain.TaskFilter{Limit: 1}, []int64{d.ID}},
		{"inbox", domain.TaskFilter{Inbox: true}, []int64{inbox.ID}},
		{"milestone", domain.TaskFilter{MilestoneID: &m.ID}, []int64{d.ID}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ids(tt.f)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("got %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestDeleteTask(t *testing.T) {
	c, _ := newCore(t)
	keep := mustTask(t, c, domain.CreateTaskInput{Title: "keep"})
	gone := mustTask(t, c, domain.CreateTaskInput{Title: "gone"})
	if err := c.DeleteTask(ctx, gone.ID); err != nil {
		t.Fatal(err)
	}
	if got := must[*domain.Task](t)(c.GetTask(ctx, gone.ID)); got != nil {
		t.Errorf("task still there: %+v", got)
	}
	if got := must[*domain.Task](t)(c.GetTask(ctx, keep.ID)); got == nil {
		t.Error("deleted the wrong task")
	}
	withReport := mustTask(t, c, domain.CreateTaskInput{Title: "with report"})
	must[*domain.Report](t)(c.AddReport(ctx, domain.AddReportInput{TaskID: withReport.ID, Agent: "claude", Summary: "x"}))
	if err := c.DeleteTask(ctx, withReport.ID); err != nil {
		t.Fatalf("task with reports: %v", err)
	}
	if err := c.DeleteTask(ctx, gone.ID); !domain.IsNotFound(err) {
		t.Errorf("second delete = %v, want not found", err)
	}
}

func TestCreateTaskCategory(t *testing.T) {
	c, _ := newCore(t)
	reading := newCategory(t, c, "Reading")
	eq(t, "unclassified by default", mustTask(t, c, domain.CreateTaskInput{Title: "x"}).CategoryID == nil, true)
	work := mustTask(t, c, domain.CreateTaskInput{Title: "x", CategoryID: ptr(int64(1))})
	eqPtr(t, "work", work.CategoryID, ptr(int64(1)))
	eqPtr(t, "work's name", work.Category, ptr("Work"))
	book := mustTask(t, c, domain.CreateTaskInput{Title: "x", CategoryID: &reading.ID})
	got := must[*domain.Task](t)(c.GetTask(ctx, book.ID))
	eqPtr(t, "user category read back", got.CategoryID, &reading.ID)
	eqPtr(t, "its name", got.Category, ptr("Reading"))
	_, err := c.CreateTask(ctx, domain.CreateTaskInput{Title: "x", CategoryID: ptr(int64(99))})
	wantFieldError(t, err, "category_id")
	if !domain.IsNotFound(err) {
		t.Errorf("a category that does not exist is not found, got %v", err)
	}
}

func TestUpdateTaskCategory(t *testing.T) {
	c, _ := newCore(t)
	reading := newCategory(t, c, "Reading")
	task := mustTask(t, c, domain.CreateTaskInput{Title: "x"})
	set := must[*domain.Task](t)(c.UpdateTask(ctx, task.ID, domain.TaskPatch{CategoryID: domain.Set(reading.ID)}))
	eqPtr(t, "set", set.CategoryID, &reading.ID)
	eqPtr(t, "set name", set.Category, ptr("Reading"))
	kept := must[*domain.Task](t)(c.UpdateTask(ctx, task.ID, domain.TaskPatch{Title: ptr("y")}))
	eqPtr(t, "an omitted category stays", kept.CategoryID, &reading.ID)
	cleared := must[*domain.Task](t)(c.UpdateTask(ctx, task.ID, domain.TaskPatch{CategoryID: domain.Clear[int64]()}))
	if cleared.CategoryID != nil || cleared.Category != nil {
		t.Errorf("category = %v %v, want cleared", cleared.CategoryID, cleared.Category)
	}
	_, err := c.UpdateTask(ctx, task.ID, domain.TaskPatch{CategoryID: domain.Set(int64(99))})
	wantFieldError(t, err, "category_id")
}

// An active task carries its category's name through the overview's joined query as well.
func TestOverviewTasksCarryTheirCategoryName(t *testing.T) {
	c, _ := newCore(t)
	reading := newCategory(t, c, "Reading")
	mustTask(t, c, domain.CreateTaskInput{Title: "x", Status: domain.StatusDoing, CategoryID: &reading.ID})
	o := must[*domain.Overview](t)(c.GetOverview(ctx, domain.OverviewOptions{}))
	got := o.Projects[0].Tasks[0]
	eqPtr(t, "category id", got.CategoryID, &reading.ID)
	eqPtr(t, "category name", got.Category, ptr("Reading"))
}

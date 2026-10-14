package domain_test

import (
	"strings"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

func at(h int) string { return domain.FormatTime(t0.Add(time.Duration(h) * time.Hour)) }

func titles(g domain.OverviewGroup) []string {
	var out []string
	for _, k := range g.Tasks {
		out = append(out, k.Title)
	}
	return out
}

func TestOverviewGroupsAndOrdersActiveTasks(t *testing.T) {
	c, clk := newCore(t)
	w := mustProject(t, c, "acme/widgets")
	a := mustProject(t, c, "acme/Api")
	for _, in := range []domain.CreateTaskInput{
		{Title: "a", ProjectID: &w.ID, Status: domain.StatusDoing},
		{Title: "b", ProjectID: &w.ID, Status: domain.StatusBlocked},
		{Title: "c", ProjectID: &w.ID, Status: domain.StatusReview},
		{Title: "d", ProjectID: &w.ID, Status: domain.StatusTodo},
		{Title: "e", ProjectID: &w.ID, Status: domain.StatusDone},
		{Title: "a2", ProjectID: &w.ID, Status: domain.StatusDoing},
		{Title: "api", ProjectID: &a.ID, Status: domain.StatusDoing},
		{Title: "inbox", Status: domain.StatusBlocked},
	} {
		mustTask(t, c, in)
	}
	clk.advance(time.Hour)
	o := must[*domain.Overview](t)(c.GetOverview(ctx, domain.OverviewOptions{}))
	eq(t, "groups", len(o.Projects), 3)
	eq(t, "first project (case-insensitive name order)", o.Projects[0].Project.Name, "Api")
	eq(t, "second project", o.Projects[1].Project.Name, "widgets")
	if o.Projects[2].Project != nil {
		t.Error("inbox must be last")
	}
	got := strings.Join(titles(o.Projects[1]), ",")
	eq(t, "attention order, newest first within a status", got, "b,c,a2,a")
	eq(t, "stale_hours default", o.StaleHours, float64(48))
	eq(t, "generated_at", o.GeneratedAt, at(1))
}

func TestOverviewStaleness(t *testing.T) {
	c, clk := newCore(t)
	p := mustProject(t, c, "acme/widgets")
	task := mustTask(t, c, domain.CreateTaskInput{Title: "a", ProjectID: &p.ID, Status: domain.StatusDoing})
	first := func() domain.OverviewTask {
		o := must[*domain.Overview](t)(c.GetOverview(ctx, domain.OverviewOptions{}))
		return o.Projects[0].Tasks[0]
	}

	clk.advance(47 * time.Hour)
	if first().Stale {
		t.Error("47h without a report is not stale (falls back to updated_at)")
	}
	clk.advance(2 * time.Hour)
	if got := first(); !got.Stale || got.HoursSinceActivity != 49 {
		t.Errorf("49h: stale=%v hours=%d", got.Stale, got.HoursSinceActivity)
	}
	must[*domain.Report](t)(c.AddReport(ctx, domain.AddReportInput{TaskID: task.ID, Agent: "x", Summary: "old", CreatedAt: at(1)}))
	must[*domain.Report](t)(c.AddReport(ctx, domain.AddReportInput{TaskID: task.ID, Agent: "x", Summary: "newer", CreatedAt: at(40)}))
	got := first()
	if got.LatestReport == nil || got.LatestReport.Summary != "newer" {
		t.Fatalf("latest report = %+v", got.LatestReport)
	}
	if got.Stale || got.HoursSinceActivity != 9 {
		t.Errorf("after a report at +40h (now +49h): stale=%v hours=%d", got.Stale, got.HoursSinceActivity)
	}
	custom := must[*domain.Overview](t)(c.GetOverview(ctx, domain.OverviewOptions{StaleHours: 5}))
	if !custom.Projects[0].Tasks[0].Stale || custom.StaleHours != 5 {
		t.Errorf("stale_hours=5 should flag a 9h-old task: %+v", custom)
	}
	negative := must[*domain.Overview](t)(c.GetOverview(ctx, domain.OverviewOptions{StaleHours: -1}))
	eq(t, "non-positive falls back to default", negative.StaleHours, float64(48))
}

func TestOverviewHidesArchivedAndEmptyProjects(t *testing.T) {
	c, _, db := newCoreDB(t)
	mustProject(t, c, "acme/empty")
	arch := mustProject(t, c, "acme/arch")
	mustTask(t, c, domain.CreateTaskInput{Title: "hidden", ProjectID: &arch.ID, Status: domain.StatusDoing})
	exec(t, db, "UPDATE projects SET archived_at = ? WHERE id = ?", at(0), arch.ID)
	if o := must[*domain.Overview](t)(c.GetOverview(ctx, domain.OverviewOptions{})); len(o.Projects) != 0 {
		t.Errorf("expected no groups, got %+v", o.Projects)
	}
}

func TestOverviewWithBlankTimestampDoesNotFail(t *testing.T) {
	c, _, db := newCoreDB(t)
	task := mustTask(t, c, domain.CreateTaskInput{Title: "a", Status: domain.StatusDoing})
	exec(t, db, "UPDATE tasks SET updated_at = 'garbage' WHERE id = ?", task.ID)
	o := must[*domain.Overview](t)(c.GetOverview(ctx, domain.OverviewOptions{}))
	eq(t, "hours", o.Projects[0].Tasks[0].HoursSinceActivity, int64(0))
}

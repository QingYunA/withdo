package domain_test

import (
	"testing"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

func TestListReportsOfReadsManyTasksAtOnce(t *testing.T) {
	c, _ := newCore(t)
	a := mustTask(t, c, domain.CreateTaskInput{Title: "a"})
	b := mustTask(t, c, domain.CreateTaskInput{Title: "b"})
	quiet := mustTask(t, c, domain.CreateTaskInput{Title: "quiet"})
	for _, in := range []domain.AddReportInput{
		{TaskID: a.ID, Agent: "x", Summary: "a1", CreatedAt: at(1)},
		{TaskID: b.ID, Agent: "x", Summary: "b1", CreatedAt: at(2)},
		{TaskID: a.ID, Agent: "x", Summary: "a0", CreatedAt: at(0)},
	} {
		must[*domain.Report](t)(c.AddReport(ctx, in))
	}
	got := must[map[int64][]domain.Report](t)(c.ListReportsOf(ctx, []int64{a.ID, b.ID, quiet.ID}))
	eq(t, "tasks with reports", len(got), 2)
	eq(t, "a is oldest first", got[a.ID][0].Summary+got[a.ID][1].Summary, "a0a1")
	eq(t, "b", got[b.ID][0].Summary, "b1")
	one := must[map[int64][]domain.Report](t)(c.ListReportsOf(ctx, []int64{b.ID}))
	eq(t, "only the asked task", len(one), 1)
	eq(t, "no ids", len(must[map[int64][]domain.Report](t)(c.ListReportsOf(ctx, nil))), 0)
	for id, rs := range got {
		want := must[[]domain.Report](t)(c.ListReports(ctx, id))
		eq(t, "same as ListReports", len(rs), len(want))
		for i := range rs {
			eq(t, "same report", rs[i].ID, want[i].ID)
		}
	}
}

func TestListAllMilestonesGroupsByProject(t *testing.T) {
	c, _ := newCore(t)
	p := mustProject(t, c, "acme/one")
	q := mustProject(t, c, "acme/two")
	must[*domain.Milestone](t)(c.CreateMilestone(ctx, domain.CreateMilestoneInput{ProjectID: p.ID, Title: "v1"}))
	must[*domain.Milestone](t)(c.CreateMilestone(ctx, domain.CreateMilestoneInput{ProjectID: q.ID, Title: "x"}))
	must[*domain.Milestone](t)(c.CreateMilestone(ctx, domain.CreateMilestoneInput{ProjectID: p.ID, Title: "v2"}))
	got := must[map[int64][]domain.Milestone](t)(c.ListAllMilestones(ctx))
	eq(t, "projects", len(got), 2)
	eq(t, "oldest first", got[p.ID][0].Title+got[p.ID][1].Title, "v1v2")
	eq(t, "other", got[q.ID][0].Title, "x")
}

func TestOverviewLatestReportBreaksATieByTheNewerId(t *testing.T) {
	c, _ := newCore(t)
	a := mustTask(t, c, domain.CreateTaskInput{Title: "a", Status: domain.StatusDoing})
	b := mustTask(t, c, domain.CreateTaskInput{Title: "b", Status: domain.StatusDoing})
	for _, in := range []domain.AddReportInput{
		{TaskID: a.ID, Agent: "x", Summary: "first", CreatedAt: at(1)},
		{TaskID: a.ID, Agent: "x", Summary: "second", CreatedAt: at(1)},
		{TaskID: b.ID, Agent: "x", Summary: "b-old", CreatedAt: at(2)},
		{TaskID: b.ID, Agent: "x", Summary: "b-older", CreatedAt: at(0)},
	} {
		must[*domain.Report](t)(c.AddReport(ctx, in))
	}
	o := must[*domain.Overview](t)(c.GetOverview(ctx, domain.OverviewOptions{}))
	got := map[string]string{}
	for _, k := range o.Projects[0].Tasks {
		got[k.Title] = k.LatestReport.Summary
	}
	eq(t, "tie", got["a"], "second")
	eq(t, "newest by time", got["b"], "b-old")
}

package domain_test

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

func TestReportsSinceReadsTheReportsAtOrAfterATimeWithTheirTask(t *testing.T) {
	c, clk := newCore(t)
	review := domain.StatusReview
	first := progress(t, c, domain.ProgressInput{Summary: "one", Cwd: widgetsRepo(t, "feat/a"), Title: "Task A"})
	at := must[string](t)(c.LatestReportAt(ctx))
	eq(t, "latest", at, first.Report.CreatedAt)
	clk.advance(time.Second)
	second := progress(t, c, domain.ProgressInput{
		TaskID: &first.Task.ID, Summary: "two", Status: &review, NextStep: "wait", PRURL: "https://github.com/acme/widgets/pull/9",
	})

	got := must[[]domain.ReportEvent](t)(c.ReportsSince(ctx, second.Report.CreatedAt))
	if len(got) != 1 || got[0].ID != second.Report.ID {
		t.Fatalf("ReportsSince = %+v, want only the second report", got)
	}
	e := got[0]
	eq(t, "task title", e.TaskTitle, "Task A")
	eqPtr(t, "project", e.ProjectName, ptr("widgets"))
	eqPtr(t, "pr number", e.PRNumber, ptr(int64(9)))
	eq(t, "status", *e.Status, domain.StatusReview)
	eqPtr(t, "next step", e.NextStep, ptr("wait"))
	if all := must[[]domain.ReportEvent](t)(c.ReportsSince(ctx, "")); len(all) != 2 {
		t.Errorf("ReportsSince(\"\") = %d reports, want 2", len(all))
	}
}

func TestLatestReportAtIsEmptyWithNoReports(t *testing.T) {
	c, _ := newCore(t)
	eq(t, "latest", must[string](t)(c.LatestReportAt(ctx)), "")
}

package domain_test

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

const prURL = "https://github.com/acme/widgets/pull/12"

func prTask(t *testing.T, c *domain.Core, status domain.Status) *domain.Task {
	t.Helper()
	return mustTask(t, c, domain.CreateTaskInput{Title: "ship it", Source: domain.SourceAgent, Status: status, PRURL: ptr(prURL)})
}

func TestNewTaskHasNoPullRequestState(t *testing.T) {
	c, _ := newCore(t)
	task := prTask(t, c, domain.StatusReview)
	if task.PRState != nil || task.PRChecks != nil || task.PRReview != nil || task.PRCheckedAt != nil || task.PRClosedSeenAt != nil {
		t.Errorf("unexpected pull request state: %+v", task)
	}
	if got := task.PullRequest(); got != (domain.PRSnapshot{}) {
		t.Errorf("PullRequest() = %+v, want the zero snapshot", got)
	}
}

func TestListPRTasksIsOpenTasksWithAPullRequest(t *testing.T) {
	c, _ := newCore(t)
	open := prTask(t, c, domain.StatusReview)
	prTask(t, c, domain.StatusDone)
	prTask(t, c, domain.StatusCancelled)
	mustTask(t, c, domain.CreateTaskInput{Title: "no pr"})
	got := must[[]domain.Task](t)(c.ListPRTasks(ctx))
	if len(got) != 1 || got[0].ID != open.ID {
		t.Fatalf("ListPRTasks = %+v, want only task %d", got, open.ID)
	}
}

func TestRecordPRStoresStateAndFilesReports(t *testing.T) {
	c, clk := newCore(t)
	task := prTask(t, c, domain.StatusReview)
	clk.advance(5 * time.Minute)
	snap := domain.PRSnapshot{State: domain.PROpen, Checks: domain.PRChecksFailing, Review: domain.PRReviewNone}
	applied := must[bool](t)(c.RecordPR(ctx, task.ID, domain.PRUpdate{URL: prURL, Snapshot: snap, Reports: []domain.PRReport{{Kind: domain.KindChecksFailing, Summary: "Checks are failing on PR #12"}}}))
	if !applied {
		t.Fatal("update was not applied")
	}
	got := must[*domain.Task](t)(c.GetTask(ctx, task.ID))
	if got.PullRequest() != snap {
		t.Errorf("PullRequest() = %+v, want %+v", got.PullRequest(), snap)
	}
	eqPtr(t, "pr_checked_at", got.PRCheckedAt, ptr("2026-10-01T00:05:00.000Z"))
	eq(t, "status untouched", got.Status, domain.StatusReview)
	eq(t, "updated_at untouched", got.UpdatedAt, task.UpdatedAt)
	reports := must[[]domain.Report](t)(c.ListReports(ctx, task.ID))
	if len(reports) != 1 {
		t.Fatalf("reports = %+v", reports)
	}
	eq(t, "agent", reports[0].Agent, domain.GitHubAgent)
	eq(t, "summary", reports[0].Summary, "Checks are failing on PR #12")
	eq(t, "kind", reports[0].Kind, domain.KindChecksFailing)
	if reports[0].Status != nil {
		t.Errorf("a plain report has no status, got %v", *reports[0].Status)
	}
}

func TestRecordPRDoneClosesTheTask(t *testing.T) {
	c, clk := newCore(t)
	task := prTask(t, c, domain.StatusReview)
	clk.advance(time.Minute)
	snap := domain.PRSnapshot{State: domain.PRMerged, Checks: domain.PRChecksPassing, Review: domain.PRReviewApproved}
	must[bool](t)(c.RecordPR(ctx, task.ID, domain.PRUpdate{URL: prURL, Snapshot: snap, Done: true, Reports: []domain.PRReport{{Kind: domain.KindPRMerged, Summary: "PR #12 merged"}}}))
	got := must[*domain.Task](t)(c.GetTask(ctx, task.ID))
	eq(t, "status", got.Status, domain.StatusDone)
	eqPtr(t, "completed_at", got.CompletedAt, ptr("2026-10-01T00:01:00.000Z"))
	reports := must[[]domain.Report](t)(c.ListReports(ctx, task.ID))
	if len(reports) != 1 || reports[0].Status == nil || *reports[0].Status != domain.StatusDone {
		t.Fatalf("the merge report should carry status done: %+v", reports)
	}
	// A done task is out of the poll.
	if left := must[[]domain.Task](t)(c.ListPRTasks(ctx)); len(left) != 0 {
		t.Errorf("ListPRTasks = %+v, want none", left)
	}
}

func TestRecordPRSkipsWhatChangedUnderneath(t *testing.T) {
	c, _ := newCore(t)
	snap := domain.PRSnapshot{State: domain.PRMerged}
	changed := prTask(t, c, domain.StatusReview)
	must[*domain.Task](t)(c.UpdateTask(ctx, changed.ID, domain.TaskPatch{PRURL: domain.Set("https://github.com/acme/widgets/pull/99")}))
	finished := prTask(t, c, domain.StatusReview)
	must[*domain.Task](t)(c.UpdateTask(ctx, finished.ID, domain.TaskPatch{Status: ptr(domain.StatusDone)}))

	for name, id := range map[string]int64{"pr_url edited": changed.ID, "task finished": finished.ID} {
		applied := must[bool](t)(c.RecordPR(ctx, id, domain.PRUpdate{URL: prURL, Snapshot: snap, Reports: []domain.PRReport{{Summary: "x"}}}))
		if applied {
			t.Errorf("%s: update was applied", name)
		}
		if reports := must[[]domain.Report](t)(c.ListReports(ctx, id)); len(reports) != 0 {
			t.Errorf("%s: reports = %+v", name, reports)
		}
	}
	got := must[*domain.Task](t)(c.GetTask(ctx, changed.ID))
	if got.PRState != nil {
		t.Errorf("pr_state = %v, want nil", *got.PRState)
	}
	if applied, err := c.RecordPR(ctx, 999, domain.PRUpdate{URL: prURL}); err != nil || applied {
		t.Errorf("unknown task: applied %v, err %v", applied, err)
	}
}

func TestDismissClosedPRHidesTheRowUntilTheStateChanges(t *testing.T) {
	c, clk := newCore(t)
	task := prTask(t, c, domain.StatusReview)
	closed := domain.PRSnapshot{State: domain.PRClosed, Checks: domain.PRChecksNone, Review: domain.PRReviewNone}
	must[bool](t)(c.RecordPR(ctx, task.ID, domain.PRUpdate{URL: prURL, Snapshot: closed}))

	clk.advance(time.Minute)
	if err := c.DismissClosedPR(ctx, task.ID); err != nil {
		t.Fatal(err)
	}
	got := must[*domain.Task](t)(c.GetTask(ctx, task.ID))
	eqPtr(t, "pr_closed_seen_at", got.PRClosedSeenAt, ptr("2026-10-01T00:01:00.000Z"))

	// Checking again while it stays closed keeps the dismissal.
	must[bool](t)(c.RecordPR(ctx, task.ID, domain.PRUpdate{URL: prURL, Snapshot: closed}))
	got = must[*domain.Task](t)(c.GetTask(ctx, task.ID))
	if got.PRClosedSeenAt == nil {
		t.Error("a second check of a closed PR cleared the dismissal")
	}
	// Reopened: the dismissal is forgotten, so a later close shows again.
	must[bool](t)(c.RecordPR(ctx, task.ID, domain.PRUpdate{URL: prURL, Snapshot: domain.PRSnapshot{State: domain.PROpen}}))
	got = must[*domain.Task](t)(c.GetTask(ctx, task.ID))
	if got.PRClosedSeenAt != nil {
		t.Errorf("pr_closed_seen_at = %v after reopening", *got.PRClosedSeenAt)
	}
}

func TestDismissClosedPRNeedsATask(t *testing.T) {
	c, _ := newCore(t)
	err := c.DismissClosedPR(ctx, 999)
	if !domain.IsNotFound(err) {
		t.Errorf("err = %v, want not found", err)
	}
}

func TestChangingTheURLForgetsThePullRequestState(t *testing.T) {
	c, _ := newCore(t)
	task := prTask(t, c, domain.StatusReview)
	snap := domain.PRSnapshot{State: domain.PROpen, Checks: domain.PRChecksPassing, Review: domain.PRReviewApproved}
	must[bool](t)(c.RecordPR(ctx, task.ID, domain.PRUpdate{URL: prURL, Snapshot: snap}))

	// Saying the same URL again, as agents do with every report, keeps what was learned.
	same := must[*domain.Task](t)(c.UpdateTask(ctx, task.ID, domain.TaskPatch{PRURL: domain.Set(prURL)}))
	if same.PullRequest() != snap || same.PRCheckedAt == nil {
		t.Errorf("same URL: %+v", same)
	}
	other := must[*domain.Task](t)(c.UpdateTask(ctx, task.ID, domain.TaskPatch{PRURL: domain.Set("https://github.com/acme/widgets/pull/13")}))
	if other.PullRequest() != (domain.PRSnapshot{}) || other.PRCheckedAt != nil || other.PRClosedSeenAt != nil {
		t.Errorf("new URL: %+v", other)
	}
}

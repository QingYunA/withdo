package data

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

const prLink = "https://github.com/acme/rocket/pull/12"

func TestRealReadsPullRequestStateAndKeepsAClosedOne(t *testing.T) {
	now := time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC)
	r, core := realPair(t, func() time.Time { return now })
	made, err := core.CreateTask(bg, domain.CreateTaskInput{Title: "Ship it", Source: domain.SourceAgent, Status: domain.StatusReview, PRURL: strp(prLink)})
	if err != nil {
		t.Fatal(err)
	}
	snap := domain.PRSnapshot{State: domain.PRClosed, Checks: domain.PRChecksNone, Review: domain.PRReviewNone}
	if _, err := core.RecordPR(bg, made.ID, domain.PRUpdate{URL: prLink, Snapshot: snap, Reports: []string{"PR #12 was closed without merging"}}); err != nil {
		t.Fatal(err)
	}

	task := mustTaskOf(t, r, int(made.ID))
	if task.PRState != PRClosed || task.PRChecks != PRChecksNone || task.PRReview != PRReviewNone ||
		task.PRCheckedAt == nil || !task.PRCheckedAt.Equal(now) || task.PRClosedSeenAt != nil || !task.PRClosedUnseen() {
		t.Fatalf("task = %+v", task)
	}
	detail, err := r.GetTask(int(made.ID))
	if err != nil || len(detail.Reports) != 1 || detail.Reports[0].Agent != GitHubAgent {
		t.Fatalf("reports = %+v, %v", detail.Reports, err)
	}

	if err := r.KeepClosedPR(int(made.ID)); err != nil {
		t.Fatal(err)
	}
	task = mustTaskOf(t, r, int(made.ID))
	if task.PRClosedSeenAt == nil || task.PRClosedUnseen() || task.Status != Review {
		t.Errorf("after Keep: %+v", task)
	}
	if err := r.KeepClosedPR(999); err == nil {
		t.Error("keeping an unknown task should fail")
	}
}

func mustTaskOf(t *testing.T, r *Real, id int) Task {
	t.Helper()
	d, err := r.GetTask(id)
	if err != nil {
		t.Fatal(err)
	}
	return d.Task
}

func TestFakeKeepsAClosedPullRequestAndHoldsPullRequestState(t *testing.T) {
	f := NewFake(fixedNow)
	if err := f.SetPullRequest(2, PRClosed, PRChecksNone, PRReviewNone); err != nil {
		t.Fatal(err)
	}
	changes := 0
	f.Subscribe(func() { changes++ })
	if !mustFakeTask(t, f, 2).PRClosedUnseen() {
		t.Fatal("task 2 should have an unseen closed pull request")
	}
	if err := f.KeepClosedPR(2); err != nil {
		t.Fatal(err)
	}
	got := mustFakeTask(t, f, 2)
	if got.PRClosedUnseen() || got.PRClosedSeenAt == nil || !got.PRClosedSeenAt.Equal(fixedNow()) || changes != 1 {
		t.Errorf("after Keep: %+v, %d change notifications", got, changes)
	}
	if err := f.KeepClosedPR(999); err == nil {
		t.Error("keeping an unknown task should fail")
	}
}

func mustFakeTask(t *testing.T, f *Fake, id int) Task {
	t.Helper()
	d, err := f.GetTask(id)
	if err != nil {
		t.Fatal(err)
	}
	return d.Task
}

func TestTaskPullRequestPredicates(t *testing.T) {
	seen := fixedNow()
	tests := []struct {
		name                     string
		task                     Task
		closed, failing, changes bool
	}{
		{"never checked", Task{}, false, false, false},
		{"closed", Task{PRState: PRClosed}, true, false, false},
		{"closed and kept", Task{PRState: PRClosed, PRClosedSeenAt: &seen}, false, false, false},
		{"open failing", Task{PRState: PROpen, PRChecks: PRChecksFailing}, false, true, false},
		{"draft failing", Task{PRState: PRDraft, PRChecks: PRChecksFailing}, false, false, false},
		{"open changes", Task{PRState: PROpen, PRReview: PRReviewChangesRequested}, false, false, true},
		{"merged changes", Task{PRState: PRMerged, PRReview: PRReviewChangesRequested}, false, false, false},
	}
	for _, tt := range tests {
		if tt.task.PRClosedUnseen() != tt.closed || tt.task.PRChecksFailing() != tt.failing || tt.task.PRChangesRequested() != tt.changes {
			t.Errorf("%s: closed %v failing %v changes %v", tt.name, tt.task.PRClosedUnseen(), tt.task.PRChecksFailing(), tt.task.PRChangesRequested())
		}
	}
}

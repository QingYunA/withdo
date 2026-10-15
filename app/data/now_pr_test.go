package data

import (
	"testing"
	"time"
)

// prTask is an agent task in review with a pull request in the given state.
func prTask(id int, title string, st Status, pr Task) Task {
	pr.ID, pr.Title, pr.Status, pr.Source = id, title, st, SourceAgent
	pr.PRURL, pr.PRNumber = "https://github.com/acme/widgets/pull/"+itoa(id), id
	pr.UpdatedAt = fixedNow().Add(-time.Hour)
	return pr
}

func nowOfTasks(tasks []Task, latest, event func(int) *Report) Now {
	if latest == nil {
		latest = func(int) *Report { return nil }
	}
	return BuildNow(NowInput{
		Now: fixedNow(), Tasks: tasks, Latest: latest, Event: event,
		ProjectName: func(*int) string { return "" },
	})
}

func kinds(items []NowItem) []NeedKind {
	out := make([]NeedKind, len(items))
	for i, it := range items {
		out[i] = it.Kind
	}
	return out
}

func TestNeedsYouOrderWithPullRequests(t *testing.T) {
	reminder := fixedNow().Add(-time.Minute)
	tasks := []Task{
		prTask(1, "stale", Doing, Task{UpdatedAt: fixedNow().Add(-100 * time.Hour)}),
		prTask(2, "review", Review, Task{}),
		prTask(3, "changes", Doing, Task{PRState: PROpen, PRReview: PRReviewChangesRequested}),
		prTask(4, "failing", Doing, Task{PRState: PROpen, PRChecks: PRChecksFailing}),
		prTask(5, "closed", Doing, Task{PRState: PRClosed}),
		prTask(6, "reminder", Todo, Task{RemindAt: &reminder}),
		prTask(7, "blocked", Blocked, Task{}),
	}
	// Task 1 is stale only through its age; keep its update old.
	tasks[0].UpdatedAt = fixedNow().Add(-100 * time.Hour)
	n := nowOfTasks(tasks, nil, nil)
	want := []NeedKind{NeedBlocked, NeedReminder, NeedPRClosed, NeedPRFailing, NeedPRChanges, NeedReview, NeedStale}
	if got := kinds(n.Needs); len(got) != len(want) {
		t.Fatalf("kinds %v, want %v", got, want)
	} else {
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("kinds %v, want %v", got, want)
			}
		}
	}
	if got := titles(n.Needs); !equal(got, []string{"blocked", "reminder", "closed", "failing", "changes", "review", "stale"}) {
		t.Errorf("titles %q", got)
	}
}

func TestPullRequestRowsAreForOpenOrClosedPullRequestsOnly(t *testing.T) {
	seen := fixedNow().Add(-time.Minute)
	tests := []struct {
		name string
		task Task
		want []NeedKind
	}{
		{"closed", Task{PRState: PRClosed}, []NeedKind{NeedPRClosed}},
		{"closed and dismissed", Task{PRState: PRClosed, PRClosedSeenAt: &seen}, nil},
		{"failing", Task{PRState: PROpen, PRChecks: PRChecksFailing}, []NeedKind{NeedPRFailing}},
		{"failing on a draft", Task{PRState: PRDraft, PRChecks: PRChecksFailing}, nil},
		{"failing on a merged pull request", Task{PRState: PRMerged, PRChecks: PRChecksFailing}, nil},
		{"pending", Task{PRState: PROpen, PRChecks: PRChecksPending}, nil},
		{"changes requested", Task{PRState: PROpen, PRReview: PRReviewChangesRequested}, []NeedKind{NeedPRChanges}},
		{"changes requested on a draft", Task{PRState: PRDraft, PRReview: PRReviewChangesRequested}, nil},
		{"failing and changes together show once, as failing", Task{PRState: PROpen, PRChecks: PRChecksFailing, PRReview: PRReviewChangesRequested}, []NeedKind{NeedPRFailing}},
		{"never checked", Task{}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// An agent task in doing: when it has no reason for Needs you it is at work.
			n := nowOfTasks([]Task{prTask(1, "x", Doing, tt.task)}, nil, nil)
			got := kinds(n.Needs)
			if len(got) != len(tt.want) || len(got) > 0 && got[0] != tt.want[0] {
				t.Errorf("kinds %v, want %v", got, tt.want)
			}
			if len(tt.want) == 0 && len(n.Agents) != 1 {
				t.Errorf("without a reason the task is at work: needs %d agents %d", len(n.Needs), len(n.Agents))
			}
		})
	}
}

func TestPullRequestRowsApplyToMyTasksAndSkipBacklogAndFinishedOnes(t *testing.T) {
	closed := Task{PRState: PRClosed}
	mine := prTask(1, "mine", Todo, closed)
	mine.Source = SourceUser
	n := nowOfTasks([]Task{mine, prTask(2, "backlog", Backlog, closed), prTask(3, "done", Done, closed), prTask(4, "cancelled", Cancelled, closed)}, nil, nil)
	if got := titles(n.Needs); !equal(got, []string{"mine"}) {
		t.Errorf("needs %q", got)
	}
	if len(n.Yours) != 0 {
		t.Errorf("a task in Needs you is not also in Yours: %d", len(n.Yours))
	}
}

func TestBlockedStaysBlockedWhenItsPullRequestClosed(t *testing.T) {
	n := nowOfTasks([]Task{prTask(1, "stuck", Blocked, Task{PRState: PRClosed})}, nil, nil)
	if len(n.Needs) != 1 || n.Needs[0].Kind != NeedBlocked {
		t.Errorf("kinds %v", kinds(n.Needs))
	}
}

func TestPullRequestRowsAgeFromTheGitHubReport(t *testing.T) {
	closedAt := fixedNow().Add(-5 * time.Hour)
	agentSaid := &Report{Agent: "claude-code", Summary: "Pushed the fix", CreatedAt: fixedNow().Add(-30 * time.Minute)}
	githubSaid := &Report{Agent: "github", Summary: "PR #5 was closed without merging", CreatedAt: closedAt}
	tasks := []Task{prTask(5, "closed", Doing, Task{PRState: PRClosed})}
	n := nowOfTasks(tasks,
		func(int) *Report { return agentSaid },
		func(int) *Report { return githubSaid })
	it := n.Needs[0]
	if !it.Since.Equal(closedAt) || it.Hours != 5 {
		t.Errorf("since %v hours %d, want the closing report 5 hours ago", it.Since, it.Hours)
	}
	if it.Stale {
		t.Error("a pull request row is not stale because of its age")
	}
}

func TestSamePullRequestKindIsOrderedByLongestWaitFirst(t *testing.T) {
	older := prTask(1, "older", Doing, Task{PRState: PROpen, PRChecks: PRChecksFailing})
	newer := prTask(2, "newer", Doing, Task{PRState: PROpen, PRChecks: PRChecksFailing})
	at := map[int]time.Time{1: fixedNow().Add(-4 * time.Hour), 2: fixedNow().Add(-time.Hour)}
	n := nowOfTasks([]Task{newer, older}, nil, func(id int) *Report { return &Report{Agent: "github", CreatedAt: at[id]} })
	if got := titles(n.Needs); !equal(got, []string{"older", "newer"}) {
		t.Errorf("order %q", got)
	}
}

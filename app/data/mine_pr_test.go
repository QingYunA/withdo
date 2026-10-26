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

// snapshotWith is a snapshot of tasks whose newest agent report is latest(id) and whose newest GitHub
// report is event(id); either may be nil.
func snapshotWith(tasks []Task, latest, event func(int) *Report) Snapshot {
	s := Snapshot{Tasks: tasks, Reports: map[int][]Report{}}
	for _, t := range tasks {
		if event != nil {
			if r := event(t.ID); r != nil {
				gh := *r
				gh.Agent = GitHubAgent
				s.Reports[t.ID] = append(s.Reports[t.ID], gh)
			}
		}
		if latest != nil {
			if r := latest(t.ID); r != nil {
				s.Reports[t.ID] = append(s.Reports[t.ID], *r)
			}
		}
	}
	return s
}

func mineInputOfTasks(tasks []Task, latest, event func(int) *Report) MineInput {
	return MineInput{Now: fixedNow(), Snapshot: snapshotWith(tasks, latest, event)}
}

func mineOfTasks(tasks []Task, latest, event func(int) *Report) Mine {
	return BuildMine(mineInputOfTasks(tasks, latest, event))
}

func kinds(items []Item) []NeedKind {
	out := make([]NeedKind, len(items))
	for i, it := range items {
		out[i] = it.Kind
	}
	return out
}

func TestFromYourAgentsOrderWithPullRequests(t *testing.T) {
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
	n := mineOfTasks(tasks, nil, nil)
	want := []NeedKind{NeedBlocked, NeedReminder, NeedPRClosed, NeedPRFailing, NeedPRChanges, NeedReview, NeedStale}
	if got := kinds(n.FromAgents); len(got) != len(want) {
		t.Fatalf("kinds %v, want %v", got, want)
	} else {
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("kinds %v, want %v", got, want)
			}
		}
	}
	if got := titles(n.FromAgents); !equal(got, []string{"blocked", "reminder", "closed", "failing", "changes", "review", "stale"}) {
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
			n := mineOfTasks([]Task{prTask(1, "x", Doing, tt.task)}, nil, nil)
			got := kinds(n.FromAgents)
			if len(got) != len(tt.want) || len(got) > 0 && got[0] != tt.want[0] {
				t.Errorf("kinds %v, want %v", got, tt.want)
			}
			if a := BuildAgents(mineInputOfTasks([]Task{prTask(1, "x", Doing, tt.task)}, nil, nil)); len(a.Working) != 1 {
				t.Errorf("an agent task in doing is at work on the Agents screen: %d", len(a.Working))
			}
		})
	}
}

func TestPullRequestRowsApplyToMyTasksAndSkipBacklogAndFinishedOnes(t *testing.T) {
	closed := Task{PRState: PRClosed}
	mine := prTask(1, "mine", Todo, closed)
	mine.Source = SourceUser
	n := mineOfTasks([]Task{mine, prTask(2, "backlog", Backlog, closed), prTask(3, "done", Done, closed), prTask(4, "cancelled", Cancelled, closed)}, nil, nil)
	if got := titles(n.FromAgents); !equal(got, []string{"mine"}) {
		t.Errorf("needs %q", got)
	}
	if len(n.Today)+len(n.Next) != 0 {
		t.Errorf("a task handed over is not also in Today or Later: %d %d", len(n.Today), len(n.Next))
	}
}

func TestBlockedStaysBlockedWhenItsPullRequestClosed(t *testing.T) {
	n := mineOfTasks([]Task{prTask(1, "stuck", Blocked, Task{PRState: PRClosed})}, nil, nil)
	if len(n.FromAgents) != 1 || n.FromAgents[0].Kind != NeedBlocked {
		t.Errorf("kinds %v", kinds(n.FromAgents))
	}
}

func TestPullRequestRowsAgeFromTheGitHubReport(t *testing.T) {
	closedAt := fixedNow().Add(-5 * time.Hour)
	agentSaid := &Report{Agent: "claude-code", Summary: "Pushed the fix", CreatedAt: fixedNow().Add(-30 * time.Minute)}
	githubSaid := &Report{Agent: "github", Summary: "PR #5 was closed without merging", CreatedAt: closedAt}
	tasks := []Task{prTask(5, "closed", Doing, Task{PRState: PRClosed})}
	n := mineOfTasks(tasks,
		func(int) *Report { return agentSaid },
		func(int) *Report { return githubSaid })
	it := n.FromAgents[0]
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
	n := mineOfTasks([]Task{newer, older}, nil, func(id int) *Report { return &Report{Agent: "github", CreatedAt: at[id]} })
	if got := titles(n.FromAgents); !equal(got, []string{"older", "newer"}) {
		t.Errorf("order %q", got)
	}
}

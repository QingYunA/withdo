package views

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
)

// prApp is the seeded workspace with three tasks whose pull requests need attention: one closed
// without merging (task 101), one failing its checks (102) and one with changes requested (103).
func prApp(t *testing.T) (*App, *data.Real, *recorder) {
	t.Helper()
	app, store, rec := newTestApp(t)
	add := func(id int, title, branch string, number int, state data.PRState, checks data.PRChecks, review data.PRReview, ago time.Duration, summary string) {
		ts := store.SeedTask(data.Task{
			ID: id, ProjectID: data.IntPtr(1), Title: title, Status: data.Doing, Source: data.SourceAgent, Branch: branch,
			PRNumber: number, PRURL: "https://github.com/QingYunA/adhd-todo/pull/" + itoa(number),
			UpdatedAt: fixedNow().Add(-ago),
		})
		store.SeedReport(data.Report{TaskID: ts.ID, Agent: "claude-code", Status: data.Doing, Summary: "Pushed " + branch, CreatedAt: fixedNow().Add(-ago - 20*time.Minute)})
		store.SeedPullRequest(ts.ID, state, checks, review)
		store.SeedReport(data.Report{TaskID: ts.ID, Agent: data.GitHubAgent, Summary: summary, CreatedAt: fixedNow().Add(-ago)})
	}
	add(101, "Rework the sync engine", "feat/sync", 14, data.PRClosed, data.PRChecksNone, data.PRReviewNone, 3*time.Hour, "PR #14 was closed without merging")
	add(102, "Add dark mode icons", "feat/icons", 15, data.PROpen, data.PRChecksFailing, data.PRReviewRequired, 40*time.Minute, "Checks are failing on PR #15")
	add(103, "Fix the export crash", "fix/export", 16, data.PROpen, data.PRChecksPassing, data.PRReviewChangesRequested, 26*time.Hour, "Changes were requested on PR #16")
	app.Refresh()
	return app, store, rec
}

func TestFromYourAgentsListsPullRequestsInThePlannedOrder(t *testing.T) {
	app, _, _ := prApp(t)
	tt, _, _ := newTesterFor(app, inlineWidth)
	has(t, tt, "PR #14 was closed without merging", "Checks are failing on PR #15", "Changes were requested on PR #16")
	order := []string{
		"Fix tray race on launch", // blocked
		"Rework the sync engine",  // closed pull request
		"Add dark mode icons",     // failing checks
		"Fix the export crash",    // changes requested
		"Native UI with mygo",     // ready for review
	}
	// A reminder of mine is not handed over: it is Today.
	if !above(t, tt, "Native UI with mygo", "Reply to design feedback") {
		t.Errorf("the reminder should be in Today, below what agents handed over: %q", tt.Texts())
	}
	for i := 1; i < len(order); i++ {
		if !above(t, tt, order[i-1], order[i]) {
			t.Errorf("%q should be above %q in %q", order[i-1], order[i], tt.Texts())
		}
	}
	// Each task shows once on the screen.
	for _, title := range order[1:4] {
		if n := countText(tt, title); n != 1 {
			t.Errorf("%q is drawn %d times", title, n)
		}
	}
}

func countText(tt interface{ Texts() []string }, s string) int {
	n := 0
	for _, text := range tt.Texts() {
		if text == s {
			n++
		}
	}
	return n
}

func TestClosedPullRequestOffersCancelTaskAndKeep(t *testing.T) {
	app, _, _ := prApp(t)
	tt, _, _ := newTesterFor(app, 1100)
	has(t, tt, "Cancel task", "Keep")
	if tt.HasText("Open PR") == false {
		t.Error("the failing and changes rows should offer Open PR")
	}
}

func TestCancelTaskCancelsTheTaskOfAClosedPullRequest(t *testing.T) {
	app, _, _ := prApp(t)
	tt, m, _ := newTesterFor(app, 1100)
	if err := tt.Click("Cancel task: Rework the sync engine"); err != nil {
		t.Fatal(err)
	}
	if task, _ := app.Task(101); task.Status != data.Cancelled {
		t.Errorf("status %v, want cancelled", task.Status)
	}
	if m.openID != 0 {
		t.Error("the action button also opened the inspector")
	}
	tt.Frame()
	hasNot(t, tt, "PR #14 was closed without merging", "Rework the sync engine")
}

func TestKeepDismissesTheClosedRowAndLeavesTheTask(t *testing.T) {
	app, _, _ := prApp(t)
	tt, m, _ := newTesterFor(app, 1100)
	if err := tt.Click("Keep task: Rework the sync engine"); err != nil {
		t.Fatal(err)
	}
	task, _ := app.Task(101)
	if task.PRClosedSeenAt == nil || task.Status != data.Doing {
		t.Errorf("after Keep: seen %v, status %v", task.PRClosedSeenAt, task.Status)
	}
	tt.Frame()
	hasNot(t, tt, "PR #14 was closed without merging", "Cancel task", "Keep")
	// The task did not disappear: it is agent work again, on the Agents screen.
	m.goTo(nav{screen: screenAgents})
	tt.Frame()
	has(t, tt, "Rework the sync engine")
	if !above(t, tt, "Claude Code", "Rework the sync engine") {
		t.Error("a kept task should be under its agent's header")
	}
}

func TestOpenPROnFailingChecksAndChangesOpensThePullRequest(t *testing.T) {
	app, _, rec := prApp(t)
	tt, m, _ := newTesterFor(app, 1100)
	for _, label := range []string{"Open pull request #15", "Open pull request #16"} {
		if err := tt.Click(label); err != nil {
			t.Fatal(err)
		}
	}
	want := []string{"https://github.com/QingYunA/adhd-todo/pull/15", "https://github.com/QingYunA/adhd-todo/pull/16"}
	if len(rec.urls) != 2 || rec.urls[0] != want[0] || rec.urls[1] != want[1] {
		t.Errorf("opened %v, want %v", rec.urls, want)
	}
	if m.openID != 0 {
		t.Error("an action button also opened the inspector")
	}
}

func TestPullRequestRowsAgeFromTheGitHubReport(t *testing.T) {
	app, _, _ := prApp(t)
	tt, _, _ := newTesterFor(app, inlineWidth)
	// 3 hours since the pull request closed; the agent's own report is older and not what counts.
	has(t, tt, "3h", "40m", "1d")
}

func TestClickingAPullRequestRowOpensTheInspector(t *testing.T) {
	app, _, _ := prApp(t)
	tt, m, _ := newTesterFor(app, 1100)
	if err := tt.Click("Add dark mode icons"); err != nil {
		t.Fatal(err)
	}
	if m.openID != 102 || !tt.HasText("Activity") {
		t.Errorf("open %d, texts %q", m.openID, tt.Texts())
	}
	// GitHub's own report shows in the activity timeline under its name.
	has(t, tt, "GitHub", "Checks are failing on PR #15")
}

func TestAgentWordsIgnoreGitHubReports(t *testing.T) {
	// A blocked task with a closed pull request is a blocked row, still worded by the agent.
	app, store, _ := newTestApp(t)
	store.SeedPullRequest(3, data.PRClosed, data.PRChecksNone, data.PRReviewNone)
	store.SeedReport(data.Report{TaskID: 3, Agent: data.GitHubAgent, Summary: "PR #14 was closed without merging", CreatedAt: fixedNow()})
	app.Refresh()
	tt, _, _ := newTesterFor(app, 1100)
	has(t, tt, "Claude Code is stuck: “Needs a decision on the activation policy”")
	hasNot(t, tt, "PR #14 was closed without merging")
}

func TestPullRequestRowsInChinese(t *testing.T) {
	app, _, _ := prApp(t)
	tt, _, _ := newTesterFor(app, 1100)
	inChinese(tt)
	tt.Frame()
	has(t, tt, "PR #14 已关闭，未合并", "PR #15 的检查未通过", "PR #16 被要求修改", "取消任务", "保留", "打开 PR")
	hasNot(t, tt, "Cancel task", "Keep")
}

func TestPullRequestRowsFitBesideTheInspector(t *testing.T) {
	app, _, _ := prApp(t)
	tt, m, _ := newTesterFor(app, 860)
	m.OpenTask(1)
	tt.Frame()
	has(t, tt, "Cancel task", "Keep", "Rework the sync engine")
	for _, name := range []string{"Cancel task", "Keep"} {
		r, ok := tt.Find(name)
		if !ok || r.X+r.W > 860 {
			t.Errorf("%q is at %+v, outside the window", name, r)
		}
	}
}

package data

import (
	"context"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
	"github.com/QingYunA/adhd-todo/internal/store"
)

var bg = context.Background()

// realPair returns the app's store and a second, independent domain.Core on the same file, which
// stands for the MCP server process.
func realPair(t *testing.T, now func() time.Time) (*Real, *domain.Core) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "t.db")
	r, err := OpenReal(bg, path, RealOptions{Now: now, Interval: 10 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close() })
	db2, err := store.Open(bg, path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db2.Close() })
	return r, domain.New(db2, domain.Options{Now: now})
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestRealReadsWhatDomainWrote(t *testing.T) {
	r, core := realPair(t, nil)
	p, err := core.UpsertProject(bg, domain.UpsertProjectInput{RepoSlug: "acme/rocket"})
	if err != nil {
		t.Fatal(err)
	}
	due := "2030-01-02T03:04:05.000Z"
	notes := "details"
	made, err := core.CreateTask(bg, domain.CreateTaskInput{
		Title: "Ship it", ProjectID: &p.ID, Priority: domain.P1, Notes: &notes, DueAt: &due,
	})
	if err != nil {
		t.Fatal(err)
	}

	tasks, err := r.ListTasks(Filter{ProjectID: IntPtr(int(p.ID))})
	if err != nil || len(tasks) != 1 {
		t.Fatalf("ListTasks = %v, %v", tasks, err)
	}
	got := tasks[0]
	if got.ID != int(made.ID) || got.Title != "Ship it" || got.Priority != P1 || got.Status != Todo ||
		got.Notes != "details" || got.Source != SourceUser || *got.ProjectID != int(p.ID) {
		t.Errorf("task = %+v", got)
	}
	if got.DueAt == nil || !got.DueAt.Equal(time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)) {
		t.Errorf("DueAt = %v", got.DueAt)
	}
	if ps, _ := r.ListProjects(); len(ps) != 1 || ps[0].Name != "rocket" || ps[0].RepoSlug != "acme/rocket" {
		t.Errorf("projects = %+v", ps)
	}
}

func TestRealWritesThroughDomain(t *testing.T) {
	r, core := realPair(t, nil)
	created, err := r.CreateTask(CreateTaskInput{Title: "From the UI", Status: Backlog})
	if err != nil || created.Status != Backlog || created.ProjectID != nil || created.Source != SourceUser {
		t.Fatalf("CreateTask = %+v, %v", created, err)
	}
	if _, err := r.CreateTask(CreateTaskInput{}); err == nil {
		t.Error("an empty title must be rejected")
	}
	seen, err := core.GetTask(bg, int64(created.ID))
	if err != nil || seen == nil || seen.Title != "From the UI" {
		t.Fatalf("domain sees %+v, %v", seen, err)
	}

	remind := time.Date(2031, 5, 6, 7, 8, 9, 0, time.UTC)
	up, err := r.UpdateTask(created.ID, Patch{
		Title: StrPtr("Renamed"), Notes: StrPtr("n"), Status: StatusPtr(Done), Priority: PriorityPtr(P0),
		Branch: StrPtr("feat/x"), RemindAt: TimePtr(remind), DueAt: TimePtr(remind),
	})
	if err != nil {
		t.Fatal(err)
	}
	if up.Title != "Renamed" || up.Notes != "n" || up.Status != Done || up.CompletedAt == nil || up.Branch != "feat/x" ||
		up.RemindAt == nil || !up.RemindAt.Equal(remind) {
		t.Errorf("update = %+v", up)
	}

	cleared, err := r.UpdateTask(created.ID, Patch{
		Notes: StrPtr(""), Branch: StrPtr(""), DueAt: TimePtr(time.Time{}), RemindAt: TimePtr(time.Time{}), Status: StatusPtr(Todo),
	})
	if err != nil {
		t.Fatal(err)
	}
	if cleared.Notes != "" || cleared.Branch != "" || cleared.DueAt != nil || cleared.RemindAt != nil || cleared.CompletedAt != nil {
		t.Errorf("cleared = %+v", cleared)
	}
	if _, err := r.UpdateTask(created.ID, Patch{Title: StrPtr("")}); err == nil {
		t.Error("an empty title must be rejected")
	}
	if _, err := r.UpdateTask(9999, Patch{Title: StrPtr("x")}); err == nil {
		t.Error("a missing task must be an error")
	}
}

func TestRealMilestonesAndGetTask(t *testing.T) {
	r, core := realPair(t, nil)
	p, _ := core.UpsertProject(bg, domain.UpsertProjectInput{RepoSlug: "acme/rocket"})
	due := "2030-06-01T00:00:00.000Z"
	ms, err := core.CreateMilestone(bg, domain.CreateMilestoneInput{ProjectID: p.ID, Title: "v1", DueAt: &due})
	if err != nil {
		t.Fatal(err)
	}
	got, err := r.ListMilestones(int(p.ID))
	if err != nil || len(got) != 1 || got[0].Title != "v1" || !got[0].Open || got[0].DueAt == nil {
		t.Fatalf("milestones = %+v, %v", got, err)
	}

	task, _ := r.CreateTask(CreateTaskInput{Title: "t", ProjectID: IntPtr(int(p.ID))})
	if up, err := r.UpdateTask(task.ID, Patch{MilestoneID: IntPtr(int(ms.ID))}); err != nil || up.MilestoneID == nil {
		t.Fatalf("set milestone = %+v, %v", up, err)
	}
	if up, err := r.UpdateTask(task.ID, Patch{MilestoneID: IntPtr(0)}); err != nil || up.MilestoneID != nil {
		t.Fatalf("clear milestone = %+v, %v", up, err)
	}

	status := domain.StatusDoing
	for _, s := range []string{"first", "second"} {
		if _, err := core.AddReport(bg, domain.AddReportInput{TaskID: int64(task.ID), Agent: "claude", Summary: s, Status: &status, NextStep: strp("go on")}); err != nil {
			t.Fatal(err)
		}
	}
	d, err := r.GetTask(task.ID)
	if err != nil || len(d.Reports) != 2 || d.Reports[0].Summary != "second" || d.Reports[0].Status != Doing || d.Reports[0].NextStep != "go on" {
		t.Fatalf("detail = %+v, %v", d, err)
	}
	if _, err := r.GetTask(4242); err == nil {
		t.Error("a missing task must be an error")
	}
}

func strp(s string) *string { return &s }

func TestRealFilterAndArchivedProjects(t *testing.T) {
	r, core := realPair(t, nil)
	p, _ := core.UpsertProject(bg, domain.UpsertProjectInput{RepoSlug: "acme/old"})
	core.UpsertProject(bg, domain.UpsertProjectInput{RepoSlug: "acme/new"})
	if _, err := r.db.Exec("UPDATE projects SET archived_at = '2030-01-01T00:00:00.000Z' WHERE id = ?", p.ID); err != nil {
		t.Fatal(err)
	}
	if ps, _ := r.ListProjects(); len(ps) != 1 || ps[0].RepoSlug != "acme/new" {
		t.Errorf("archived projects must be hidden: %+v", ps)
	}

	r.CreateTask(CreateTaskInput{Title: "mine"})
	r.CreateTask(CreateTaskInput{Title: "agent's", Source: SourceAgent, Status: Doing})
	if ts, _ := r.ListTasks(Filter{Source: SourceAgent}); len(ts) != 1 || ts[0].Title != "agent's" {
		t.Errorf("source filter = %+v", ts)
	}
	if ts, _ := r.ListTasks(Filter{Status: []Status{Todo}, Inbox: true}); len(ts) != 1 || ts[0].Title != "mine" {
		t.Errorf("status filter = %+v", ts)
	}
	if ts, _ := r.ListTasks(Filter{}); len(ts) != 2 || ts[0].Title != "agent's" {
		t.Errorf("newest first = %+v", ts)
	}
}

func TestRealOverview(t *testing.T) {
	r, core := realPair(t, nil)
	p, _ := core.UpsertProject(bg, domain.UpsertProjectInput{RepoSlug: "acme/rocket"})
	doing, blocked := domain.StatusDoing, domain.StatusBlocked
	core.CreateTask(bg, domain.CreateTaskInput{Title: "quiet", ProjectID: &p.ID, Status: doing})
	res, err := core.ReportProgress(bg, domain.ProgressInput{Summary: "stuck", Status: &blocked, Title: "inbox job"})
	if err != nil {
		t.Fatal(err)
	}
	o, err := r.Overview()
	if err != nil || len(o.Groups) != 2 {
		t.Fatalf("overview = %+v, %v", o, err)
	}
	if o.Groups[0].Project == nil || o.Groups[0].Project.RepoSlug != "acme/rocket" || o.Groups[0].Tasks[0].LatestReport != nil {
		t.Errorf("project group = %+v", o.Groups[0])
	}
	inbox := o.Groups[1]
	if inbox.Project != nil || inbox.Tasks[0].ID != int(res.Task.ID) || inbox.Tasks[0].LatestReport.Summary != "stuck" || inbox.Tasks[0].Status != Blocked {
		t.Errorf("inbox group = %+v", inbox)
	}
	if o.StaleHours != DefaultStaleHours {
		t.Errorf("stale hours = %d", o.StaleHours)
	}
}

func TestRealNotifiesAboutOtherProcessesWrites(t *testing.T) {
	r, core := realPair(t, nil)
	var changes atomic.Int32
	cancel := r.Subscribe(func() { changes.Add(1) })
	r.Start()
	r.Start() // a second Start is harmless
	time.Sleep(50 * time.Millisecond)
	base := changes.Load()

	if _, err := core.CreateTask(bg, domain.CreateTaskInput{Title: "written elsewhere"}); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "a change event", func() bool { return changes.Load() > base })
	if ts, _ := r.ListTasks(Filter{}); len(ts) != 1 {
		t.Errorf("the change must be readable: %+v", ts)
	}

	cancel()
	after := changes.Load()
	core.CreateTask(bg, domain.CreateTaskInput{Title: "unheard"})
	time.Sleep(100 * time.Millisecond)
	if changes.Load() != after {
		t.Error("a cancelled subscription must stay quiet")
	}
}

func TestRealReminderFiresOnce(t *testing.T) {
	var clock atomic.Int64
	clock.Store(time.Date(2030, 1, 1, 9, 0, 0, 0, time.UTC).UnixNano())
	now := func() time.Time { return time.Unix(0, clock.Load()).UTC() }
	r, core := realPair(t, now)
	p, _ := core.UpsertProject(bg, domain.UpsertProjectInput{RepoSlug: "acme/rocket"})
	at := "2030-01-01T09:30:00.000Z"
	task, _ := core.CreateTask(bg, domain.CreateTaskInput{Title: "call back", ProjectID: &p.ID, RemindAt: &at})
	inbox, _ := core.CreateTask(bg, domain.CreateTaskInput{Title: "stretch", RemindAt: &at})

	got := make(chan Reminder, 8)
	cancel := r.SubscribeReminders(func(rem Reminder) { got <- rem })
	defer cancel()
	r.Start()
	time.Sleep(60 * time.Millisecond)
	select {
	case rem := <-got:
		t.Fatalf("not due yet, got %+v", rem)
	default:
	}

	clock.Add(int64(time.Hour))
	byTask := map[int]Reminder{}
	for len(byTask) < 2 {
		select {
		case rem := <-got:
			byTask[rem.TaskID] = rem
		case <-time.After(3 * time.Second):
			t.Fatalf("reminders = %+v", byTask)
		}
	}
	if byTask[int(task.ID)].ProjectName != "rocket" || byTask[int(task.ID)].Title != "call back" {
		t.Errorf("project reminder = %+v", byTask[int(task.ID)])
	}
	if byTask[int(inbox.ID)].ProjectName != "" {
		t.Errorf("inbox reminder = %+v", byTask[int(inbox.ID)])
	}
	time.Sleep(100 * time.Millisecond)
	select {
	case rem := <-got:
		t.Fatalf("a reminder must fire once, got %+v again", rem)
	default:
	}
}

func TestRealCloseStopsTheWatcher(t *testing.T) {
	r, _ := realPair(t, nil)
	r.Start()
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestOpenRealReportsBadPath(t *testing.T) {
	if _, err := OpenReal(bg, "/dev/null/nope/x.db", RealOptions{}); err == nil {
		t.Error("expected an error")
	}
}

func TestParseTimeTolerance(t *testing.T) {
	if !parseTime("garbage").IsZero() {
		t.Error("a bad timestamp maps to the zero time")
	}
}

// TestRealFeedsTheNowScreen writes what agents and the user write, reads it back through the
// adapter and checks the Now sections: next_step, the latest report, the due reminder, the PR.
func TestRealFeedsTheNowScreen(t *testing.T) {
	now := time.Date(2026, 10, 14, 14, 30, 0, 0, time.UTC)
	r, core := realPair(t, func() time.Time { return now })
	p, _ := core.UpsertProject(bg, domain.UpsertProjectInput{RepoSlug: "acme/rocket"})
	blocked := domain.StatusBlocked
	_, err := core.ReportProgress(bg, domain.ProgressInput{
		TaskID: nil, Title: "Fix launch race", Summary: "Reproduced it", Status: &blocked,
		NextStep: "Needs a decision on the policy", Branch: "fix/launch", Cwd: "", Agent: "claude-code",
	})
	if err != nil {
		t.Fatal(err)
	}
	review := domain.StatusReview
	if _, err := core.ReportProgress(bg, domain.ProgressInput{
		Title: "Native UI", Summary: "Ready", Status: &review, Branch: "feat/ui",
		PRURL: "https://github.com/acme/rocket/pull/12", Agent: "claude-code",
	}); err != nil {
		t.Fatal(err)
	}
	mine, _ := r.CreateTask(CreateTaskInput{Title: "Reply to feedback", ProjectID: IntPtr(int(p.ID))})
	if _, err := r.UpdateTask(mine.ID, Patch{RemindAt: TimePtr(now.Add(-time.Minute))}); err != nil {
		t.Fatal(err)
	}

	tasks, _ := r.ListTasks(Filter{})
	latest := func(id int) *Report {
		d, err := r.GetTask(id)
		if err != nil || len(d.Reports) == 0 {
			return nil
		}
		return &d.Reports[0]
	}
	n := BuildNow(NowInput{Now: now, Tasks: tasks, Latest: latest, ProjectName: func(*int) string { return "rocket" }})
	if len(n.Needs) != 3 || len(n.Agents) != 0 || len(n.Yours) != 0 {
		t.Fatalf("sections %d/%d/%d: %+v", len(n.Needs), len(n.Agents), len(n.Yours), n.Needs)
	}
	if n.Needs[0].Kind != NeedBlocked || n.Needs[1].Kind != NeedReminder || n.Needs[2].Kind != NeedReview {
		t.Errorf("order %v %v %v", n.Needs[0].Kind, n.Needs[1].Kind, n.Needs[2].Kind)
	}
	if n.Needs[0].Words != "Needs a decision on the policy" || n.Needs[1].RemindAt == nil || n.Needs[2].PRNumber != 12 {
		t.Errorf("needs %+v", n.Needs)
	}
}

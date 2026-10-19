package data

import (
	"reflect"
	"testing"
	"time"
)

// The snapshot holds what the single reads gave, one task and one project at a time.
func TestSnapshotEqualsTheSingleReadsOfTheSeededDemo(t *testing.T) {
	r := seededDemo(t)
	gone := r.SeedProject("gone", "me/gone")
	r.SeedTask(Task{Title: "in a project that is archived", ProjectID: &gone.ID, Status: Doing, Source: SourceAgent})
	if err := r.ArchiveProject(gone.ID); err != nil {
		t.Fatal(err)
	}
	snap := must[Snapshot](t)(r.Snapshot())

	if want := must[[]Project](t)(r.ListProjects()); !reflect.DeepEqual(snap.Projects, want) {
		t.Errorf("projects %+v, want %+v", snap.Projects, want)
	}
	if want := must[[]Project](t)(r.ListArchivedProjects()); len(want) != 1 || !reflect.DeepEqual(snap.Archived, want) {
		t.Errorf("archived %+v, want %+v", snap.Archived, want)
	}
	if want := must[[]Category](t)(r.ListCategories()); !reflect.DeepEqual(snap.Categories, want) {
		t.Errorf("categories %+v, want %+v", snap.Categories, want)
	}
	tasks := must[[]Task](t)(r.ListTasks(Filter{}))
	if len(tasks) == 0 || !reflect.DeepEqual(snap.Tasks, tasks) {
		t.Errorf("tasks %d, want %d", len(snap.Tasks), len(tasks))
	}
	for _, p := range snap.Projects {
		want := must[[]Milestone](t)(r.ListMilestones(p.ID))
		if len(want) != len(snap.Milestones[p.ID]) || len(want) > 0 && !reflect.DeepEqual(snap.Milestones[p.ID], want) {
			t.Errorf("milestones of %s: %+v, want %+v", p.Name, snap.Milestones[p.ID], want)
		}
	}
	if want := must[Overview](t)(r.Overview()); !reflect.DeepEqual(snap.Overview, want) {
		t.Errorf("overview %+v, want %+v", snap.Overview, want)
	}
	withReports := 0
	for _, task := range tasks {
		want := must[TaskDetail](t)(r.GetTask(task.ID))
		got := snap.Details()[task.ID]
		if !reflect.DeepEqual(got.Task, want.Task) || len(got.Reports) != len(want.Reports) ||
			len(want.Reports) > 0 && !reflect.DeepEqual(got.Reports, want.Reports) {
			t.Errorf("task %d: %+v, want %+v", task.ID, got, want)
		}
		if len(want.Reports) > 0 {
			withReports++
		}
	}
	if withReports == 0 {
		t.Error("the demo has no reports to compare")
	}
	if len(snap.Reports) != withReports {
		t.Errorf("reports of %d tasks, want %d: a task without any is not in the map", len(snap.Reports), withReports)
	}
}

// Reading the snapshot takes as many queries for sixty-five tasks as for twenty-five.
func TestSnapshotTakesAFixedNumberOfQueries(t *testing.T) {
	count := func(extra int) int64 {
		r, n := countedDemo(t, func(p *Real) {
			for i := 0; i < extra; i++ {
				task := p.SeedTask(Task{Title: "more", Status: Doing, Source: SourceAgent})
				p.SeedReportAs(task.ID, "claude-code", "said", fixedNow())
			}
		})
		snap := must[Snapshot](t)(r.Snapshot())
		if len(snap.Tasks) < 25+extra {
			t.Fatalf("%d tasks, want at least %d", len(snap.Tasks), 25+extra)
		}
		return n.Load()
	}
	few, many := count(0), count(40)
	if few != many {
		t.Errorf("%d queries for the demo, %d with forty tasks more", few, many)
	}
	t.Logf("Snapshot: %d queries", few)
}

func TestTheLatestReportsOfATask(t *testing.T) {
	at := func(m int) time.Time { return fixedNow().Add(time.Duration(m) * time.Minute) }
	rep := func(agent, said string, m int) Report { return Report{Agent: agent, Summary: said, CreatedAt: at(m)} }
	for _, tc := range []struct {
		name          string
		reports       []Report // newest first
		agent, github string   // the summaries wanted, "" for none
	}{
		{"no reports", nil, "", ""},
		{"an agent only", []Report{rep("claude-code", "b", 2), rep("codex", "a", 1)}, "b", ""},
		{"GitHub only", []Report{rep(GitHubAgent, "PR #1 merged", 2)}, "", "PR #1 merged"},
		{"GitHub is not the agent's last word", []Report{rep(GitHubAgent, "merged", 3), rep("claude-code", "done", 2), rep(GitHubAgent, "checks", 1)}, "done", "merged"},
		{"each is the newest of its kind", []Report{rep("codex", "c", 4), rep(GitHubAgent, "g2", 3), rep("claude-code", "a", 2), rep(GitHubAgent, "g1", 1)}, "c", "g2"},
	} {
		d := TaskDetail{Reports: tc.reports}
		for _, rule := range []struct {
			name string
			got  *Report
			want string
		}{{"agent", d.LatestReport(), tc.agent}, {"github", d.GitHubReport(), tc.github}} {
			switch {
			case rule.want == "" && rule.got != nil:
				t.Errorf("%s, %s: %+v, want none", tc.name, rule.name, rule.got)
			case rule.want != "" && (rule.got == nil || rule.got.Summary != rule.want):
				t.Errorf("%s, %s: %+v, want %q", tc.name, rule.name, rule.got, rule.want)
			}
		}
		s := Snapshot{Reports: map[int][]Report{7: tc.reports}}
		if s.LatestReport(7) != nil != (tc.agent != "") || s.GitHubReport(7) != nil != (tc.github != "") {
			t.Errorf("%s: the snapshot disagrees with the detail", tc.name)
		}
	}
	if (Snapshot{}).LatestReport(1) != nil || (Snapshot{}).GitHubReport(1) != nil {
		t.Error("a task the snapshot does not know has reports")
	}
}

// Two reports at the same moment: the one filed later is the newest, as the single read had it.
func TestTwoReportsAtTheSameMomentTheLaterFiledIsTheNewest(t *testing.T) {
	r := seededDemo(t)
	task := r.SeedTask(Task{Title: "tie", Status: Doing, Source: SourceAgent})
	at := fixedNow().Add(-time.Hour)
	r.SeedReportAs(task.ID, "claude-code", "first", at)
	r.SeedReportAs(task.ID, "codex", "second", at)
	snap := must[Snapshot](t)(r.Snapshot())
	if got := snap.LatestReport(task.ID); got == nil || got.Summary != "second" {
		t.Errorf("latest %+v, want second", got)
	}
	if want := must[TaskDetail](t)(r.GetTask(task.ID)).LatestReport(); want.Summary != "second" {
		t.Errorf("single read %+v", want)
	}
}

// Describe is the overview's rule, for the tasks no GitHub report touched.
func TestDescribeAgreesWithTheOverview(t *testing.T) {
	r := seededDemo(t)
	snap := must[Snapshot](t)(r.Snapshot())
	checked := 0
	for _, g := range snap.Overview.Groups {
		for _, want := range g.Tasks {
			if snap.GitHubReport(want.ID) != nil {
				continue
			}
			got := snap.Describe(want.Task, r.Now())
			if got.Stale != want.Stale || got.HoursSinceActivity != want.HoursSinceActivity ||
				(got.LatestReport == nil) != (want.LatestReport == nil) ||
				got.LatestReport != nil && got.LatestReport.ID != want.LatestReport.ID {
				t.Errorf("task %d: %+v, overview %+v", want.ID, got, want)
			}
			checked++
		}
	}
	if checked < 4 {
		t.Errorf("only %d tasks compared", checked)
	}
}

// The reminder counts with CountSummary from a snapshot; the page builds its summary from the same
// snapshot with more in it (kinds, who is who). The counts are the same.
func TestCountSummaryEqualsTheCountsOfAFullSummary(t *testing.T) {
	r := seededDemo(t)
	r.SeedCompleteAt(7, fixedNow().Add(-2*time.Hour))
	r.SeedPullRequest(10, PRMerged, PRChecksPassing, PRReviewApproved)
	r.SeedCompleteAt(10, fixedNow().Add(-time.Hour))
	snap := must[Snapshot](t)(r.Snapshot())
	canon := func(name string) string { return "x" + name }
	saw := 0
	for back := 0; back < 3; back++ {
		for _, p := range []Period{DayPeriod(fixedNow(), back), WeekPeriod(fixedNow(), back, time.Monday)} {
			full := BuildSummary(SummaryInput{
				Snapshot: snap, Now: fixedNow(), Period: p, Agent: canon, Kinds: map[int]string{7: "gem"},
			}).Counts()
			if got := CountSummary(snap, p, nil); got != full {
				t.Errorf("%s: %+v, want %+v", p.Kind(), got, full)
			}
			saw += full.Finished
		}
	}
	if saw == 0 {
		t.Error("nothing finished in any period")
	}
}

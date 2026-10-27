package data

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/domain"
)

// eventRig is a Real store with its watcher running, the second core standing for the MCP server,
// and a channel of the events it told.
type eventRig struct {
	t    *testing.T
	r    *Real
	core *domain.Core
	got  chan Event
}

func newEventRig(t *testing.T, before func(*domain.Core)) *eventRig {
	t.Helper()
	r, core := realPair(t, nil)
	if before != nil {
		before(core)
	}
	rig := &eventRig{t: t, r: r, core: core, got: make(chan Event, 16)}
	cancel := r.SubscribeEvents(func(e Event) { rig.got <- e })
	t.Cleanup(cancel)
	r.Start()
	return rig
}

func (g *eventRig) report(in domain.ProgressInput) *domain.ProgressResult {
	g.t.Helper()
	res, err := g.core.ReportProgress(bg, in)
	if err != nil {
		g.t.Fatal(err)
	}
	return res
}

func (g *eventRig) next() Event {
	g.t.Helper()
	select {
	case e := <-g.got:
		return e
	case <-time.After(3 * time.Second):
		g.t.Fatal("timed out waiting for an event")
		return Event{}
	}
}

// settled waits until a marker event arrives, so that anything told before it would have been
// seen: the events in between must be none.
func (g *eventRig) settled() {
	g.t.Helper()
	review := domain.StatusReview
	g.report(domain.ProgressInput{Agent: "marker", Summary: "marker", Title: "marker", Status: &review, Branch: "marker"})
	if e := g.next(); e.Agent != "marker" {
		g.t.Fatalf("got %+v before the marker", e)
	}
}

func status(s domain.Status) *domain.Status { return &s }

func TestEventsAgentReportsThatHandBackOrGetStuck(t *testing.T) {
	g := newEventRig(t, nil)
	p, _ := g.core.UpsertProject(bg, domain.UpsertProjectInput{RepoSlug: "acme/rocket"})
	task, err := g.core.CreateTask(bg, domain.CreateTaskInput{Title: "Ship it", ProjectID: &p.ID, PRURL: strp(prLink)})
	if err != nil {
		t.Fatal(err)
	}
	id := task.ID
	g.report(domain.ProgressInput{TaskID: &id, Agent: "claude-code", Summary: "all done", Status: status(domain.StatusReview)})
	want := Event{Kind: EventHandedBack, TaskID: int(id), Title: "Ship it", ProjectName: "rocket", Agent: "claude-code", PRNumber: 12, Detail: "all done"}
	if e := g.next(); e != want {
		t.Errorf("handed back = %+v, want %+v", e, want)
	}

	g.report(domain.ProgressInput{TaskID: &id, Agent: "codex", Summary: "cannot go on", NextStep: "which activation policy?", Status: status(domain.StatusBlocked)})
	if e := g.next(); e.Kind != EventStuck || e.Detail != "which activation policy?" || e.Agent != "codex" {
		t.Errorf("stuck = %+v", e)
	}
	g.report(domain.ProgressInput{TaskID: &id, Agent: "codex", Summary: "no idea", Status: status(domain.StatusBlocked)})
	if e := g.next(); e.Kind != EventStuck || e.Detail != "no idea" {
		t.Errorf("stuck without next step = %+v", e)
	}
}

func TestEventsOnlyTheReportsThatMoveATaskToReviewOrBlocked(t *testing.T) {
	g := newEventRig(t, nil)
	task, _ := g.core.CreateTask(bg, domain.CreateTaskInput{Title: "Inbox thing"})
	id := task.ID
	g.report(domain.ProgressInput{TaskID: &id, Agent: "claude-code", Summary: "working", Status: status(domain.StatusDoing)})
	g.report(domain.ProgressInput{TaskID: &id, Agent: "claude-code", Summary: "a note"})
	g.report(domain.ProgressInput{TaskID: &id, Agent: domain.HousekeepingAgent, Summary: "closed", Status: status(domain.StatusReview)})
	g.report(domain.ProgressInput{TaskID: &id, Agent: domain.LegacyHousekeepingAgent, Summary: "closed", Status: status(domain.StatusReview)})
	// What the user does in the app writes no report, so it tells nothing.
	if _, err := g.r.UpdateTask(int(id), Patch{Status: StatusPtr(Blocked)}); err != nil {
		t.Fatal(err)
	}
	g.settled()
}

func TestEventsAreNotToldForWhatHappenedBeforeStart(t *testing.T) {
	g := newEventRig(t, func(core *domain.Core) {
		_, err := core.ReportProgress(bg, domain.ProgressInput{Agent: "claude-code", Summary: "old", Title: "old", Status: status(domain.StatusReview), Branch: "old"})
		if err != nil {
			t.Fatal(err)
		}
	})
	g.settled()
}

func TestEventsEachReportIsToldOnce(t *testing.T) {
	g := newEventRig(t, nil)
	g.report(domain.ProgressInput{Agent: "claude-code", Summary: "one", Title: "one", Status: status(domain.StatusReview), Branch: "one"})
	g.next()
	// Another change to the database tells nothing again.
	if _, err := g.r.CreateTask(CreateTaskInput{Title: "unrelated"}); err != nil {
		t.Fatal(err)
	}
	g.settled()
}

func TestEventsGitHubChecksFailingAndMergedByTheirKind(t *testing.T) {
	g := newEventRig(t, nil)
	task, _ := g.core.CreateTask(bg, domain.CreateTaskInput{Title: "Ship it", Source: domain.SourceAgent, Status: domain.StatusReview, PRURL: strp(prLink)})
	record := func(snap domain.PRSnapshot, done bool, reports ...domain.PRReport) {
		t.Helper()
		if _, err := g.core.RecordPR(bg, task.ID, domain.PRUpdate{URL: prLink, Snapshot: snap, Done: done, Reports: reports}); err != nil {
			t.Fatal(err)
		}
	}
	open := domain.PRSnapshot{State: domain.PROpen, Checks: domain.PRChecksFailing, Review: domain.PRReviewChangesRequested}
	record(open, false,
		domain.PRReport{Kind: domain.KindChecksFailing, Summary: "Checks are failing on PR #12"},
		domain.PRReport{Kind: domain.KindChangesRequested, Summary: "Changes were requested on PR #12"})
	if e := g.next(); e.Kind != EventChecksFailing || e.PRNumber != 12 || e.Title != "Ship it" || e.Agent != "" {
		t.Errorf("checks failing = %+v", e)
	}
	record(domain.PRSnapshot{State: domain.PRMerged}, true, domain.PRReport{Kind: domain.KindPRMerged, Summary: "PR #12 merged"})
	if e := g.next(); e.Kind != EventMerged || e.PRNumber != 12 {
		t.Errorf("merged = %+v", e)
	}
	g.settled() // changes requested told nothing
}

// A deleted report's id is reused by the next one. The reports are read by time, so the new one is
// told even when the watcher never saw the gap, and with the clock standing still. Driven by hand,
// not by the watcher, so that it does not depend on its timing.
func TestEventsAReportThatReusesADeletedOnesIdIsStillTold(t *testing.T) {
	now := func() time.Time { return time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC) }
	r, core := realPair(t, now)
	var got []Event
	r.SubscribeEvents(func(e Event) { got = append(got, e) })
	r.markEvents()
	report := func(title string) int64 {
		t.Helper()
		res, err := core.ReportProgress(bg, domain.ProgressInput{Agent: "claude-code", Summary: "done", Title: title, Status: status(domain.StatusReview), Branch: title})
		if err != nil {
			t.Fatal(err)
		}
		return res.Report.ID
	}

	first := report("first")
	r.deliverEvents()
	if len(got) != 1 || got[0].Title != "first" {
		t.Fatalf("events = %+v", got)
	}
	r.deliverEvents() // nothing new
	if len(got) != 1 {
		t.Fatalf("told again: %+v", got)
	}
	tasks, _ := r.ListTasks(Filter{})
	if err := r.DeleteTask(tasks[0].ID); err != nil {
		t.Fatal(err)
	}
	if again := report("second"); again != first {
		t.Fatalf("the new report has id %d, want the deleted one's %d to test the reuse", again, first)
	}
	r.deliverEvents()
	if len(got) != 2 || got[1].Title != "second" {
		t.Errorf("events = %+v, want the report that reused the id", got)
	}
	r.deliverEvents()
	if len(got) != 2 {
		t.Errorf("told again: %+v", got)
	}
}

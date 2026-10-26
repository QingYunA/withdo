package views

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/QingYunA/adhd-todo/app/theme"
)

// idleApp is an app with no agent task: one task of mine that Codex reported on three hours ago
// and Claude Code twelve minutes ago.
func idleApp(t *testing.T) *App {
	t.Helper()
	app, _ := newEmptyApp(t)
	if err := app.AddTask(data.CreateTaskInput{Title: "Old chore", Source: data.SourceUser}); err != nil {
		t.Fatal(err)
	}
	store := app.Store.(*data.Real)
	id := app.snap.tasks[0].ID
	store.SeedReportAs(id, "codex", "older", fixedNow().Add(-3*time.Hour))
	store.SeedReportAs(id, "claude-code", "newer", fixedNow().Add(-12*time.Minute))
	app.Refresh()
	return app
}

// The idle tray names who reported last across every task, and how long ago.
func TestTheIdleCardNamesTheAgentThatReportedLast(t *testing.T) {
	tt, _, _ := newTesterFor(idleApp(t), 1100)
	has(t, tt, "Idle", "All resting", "Claude Code · 12m")
	hasNot(t, tt, "Codex · 3h")
}

func TestTheIdleCardGivesWayToTheDropTargetWhileARowIsDragged(t *testing.T) {
	r := newHandRig(t, true)
	r.m.goTo(nav{})
	r.tt.Frame()
	r.tt.Frame()
	r.dragOnto(t, "Book dentist", "Native UI with mygo", true)
	r.tt.Frame()
	has(t, r.tt, panelDrop)
	hasNot(t, r.tt, "All resting")
}

func TestTheZsOfTheSleepingRobotRiseInStepsAndStandStillWithReduceMotion(t *testing.T) {
	base := time.UnixMilli(0)
	var seen [2][]theme.SleepZ
	for step := range 4 {
		f := theme.SleepFrame(base.Add(time.Duration(step)*theme.SleepStep), false)
		seen[0], seen[1] = append(seen[0], f[0]), append(seen[1], f[1])
	}
	for step := range 4 {
		if seen[0][step] != seen[1][(step+2)%4] {
			t.Errorf("the second z is not half a loop behind at step %d: %v %v", step, seen[0], seen[1])
		}
	}
	if seen[0][0].Shown || !seen[0][1].Shown || seen[0][1].Rise >= seen[0][2].Rise {
		t.Errorf("a z should appear, then rise: %v", seen[0])
	}
	if f := theme.SleepFrame(base.Add(time.Second), true); !f[0].Shown || !f[1].Shown || f != theme.SleepFrame(base, true) {
		t.Errorf("Reduce Motion should show both z's still: %v", f)
	}
}

func TestTheSidebarSummaryRowOpensSummaryAndIsSelectedThere(t *testing.T) {
	tt, m, _ := newMainTester(t, 1100)
	tt.Frame()
	if m.shows(nav{screen: screenSummary}) {
		t.Fatal("Summary is selected before it is opened")
	}
	m.sum.week, m.sum.back = true, 2
	if err := tt.Click("Summary"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !m.shows(nav{screen: screenSummary}) || !m.ShowingSummary() {
		t.Fatalf("the row went to %+v", m.nav)
	}
	if m.sum.week || m.sum.back != 0 {
		t.Errorf("Summary should open on today, got week=%v back=%d", m.sum.week, m.sum.back)
	}
}

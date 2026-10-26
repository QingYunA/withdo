package views

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
)

// Every task is on Today, Agents or its project's page: the rows and folds that keep it so.

func visibilityApp(t *testing.T) (*App, *data.Real) {
	t.Helper()
	store := demoStore(t, fixedNow, false)
	app := NewApp(store, &recorder{}, testConfig())
	app.Now = fixedNow
	return app, store
}

func TestMyTaskInReviewIsInTodayWithAToReviewTag(t *testing.T) {
	app, store := visibilityApp(t)
	store.CreateTask(data.CreateTaskInput{Title: "Check the draft", Status: data.Review})
	store.CreateTask(data.CreateTaskInput{Title: "Plain one", Status: data.Doing})
	app.Refresh()
	tt, _, _ := newTesterFor(app, 1100)
	tt.Frame()
	has(t, tt, "Check the draft", "To review")
	if n := app.buildMine().Count(); n != 2 {
		t.Errorf("today counts %d, want both", n)
	}
	if tt.HasText("All clear") {
		t.Error("a task waits for me and Today says all clear")
	}
}

func TestTasksDatedBeyondAWeekAreOnTheCalendarNotOnHome(t *testing.T) {
	app, store := visibilityApp(t)
	far := func(days int) *time.Time { return data.TimePtr(fixedNow().Add(time.Duration(days) * 24 * time.Hour)) }
	store.CreateTask(data.CreateTaskInput{Title: "Renew passport", DueAt: far(40)})
	store.CreateTask(data.CreateTaskInput{Title: "Book flights", DueAt: far(15)})
	app.Refresh()
	tt, _, _ := newTesterFor(app, 1100)
	tt.Frame()
	hasNot(t, tt, "Renew passport", "Book flights", "After next week", "No date")
}

func TestAgentsListsQueuedTasksAndFoldsTheBacklogOfTasksWithNoProject(t *testing.T) {
	app, store := visibilityApp(t)
	store.CreateTask(data.CreateTaskInput{Title: "Write the changelog", Status: data.Todo, Source: data.SourceAgent})
	store.CreateTask(data.CreateTaskInput{Title: "Maybe later idea", Status: data.Backlog, Source: data.SourceAgent})
	app.Refresh()
	tt, m, _ := newTesterFor(app, 1100)
	m.goTo(nav{screen: screenAgents})
	tt.Frame()
	has(t, tt, "Queued", "Write the changelog", "Backlog 1")
	hasNot(t, tt, "Maybe later idea", "No agents at work")
	click(t, tt, "Backlog 1")
	has(t, tt, "Maybe later idea")
}

func TestARepositoryTaskShowsNoWorkChipUnlessWorkWasSetOnIt(t *testing.T) {
	app, store := visibilityApp(t)
	p := store.SeedProject("scratch", "me/scratch")
	tomorrow := data.TimePtr(fixedNow().Add(24 * time.Hour))
	store.CreateTask(data.CreateTaskInput{Title: "Implied work", ProjectID: data.IntPtr(p.ID), DueAt: tomorrow})
	store.CreateTask(data.CreateTaskInput{Title: "Set work", ProjectID: data.IntPtr(p.ID), Category: workID, DueAt: tomorrow})
	store.CreateTask(data.CreateTaskInput{Title: "Set life", Category: lifeID, DueAt: tomorrow})
	app.Refresh()
	tt, _, _ := newTesterFor(app, 1100)
	tt.Frame()
	has(t, tt, "Implied work", "Set work", "Set life")
	if n := countText(tt, "Show only Work"); n != 1 {
		t.Errorf("%d Work chips, want one: the explicit one", n)
	}
	has(t, tt, "Show only Life")
}

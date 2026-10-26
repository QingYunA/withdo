package views

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// The M6 screens: Mine, Agents with subtasks, the inspector of a parent and of a subtask, the badge
// and the hop in flight. Render with
//
//	SCREENS_OUT=/tmp/m6-p2 go test ./app/views -run TestScreensMine

// mineApp is the seeded workspace with one more subtask handed back to its parent (the seed has
// subtasks that are done and doing only), so every status pill is on screen.
func mineApp(t *testing.T) (*App, *data.Real) {
	t.Helper()
	store := demoStore(t, fixedNow, true)
	app := NewApp(store, &recorder{}, testConfig())
	app.Now = fixedNow
	sub, err := store.CreateTask(data.CreateTaskInput{
		Title: "Vendor icons and one-click install", ProjectID: data.IntPtr(1), ParentID: data.IntPtr(1),
		Source: data.SourceAgent, Status: data.Review,
	})
	if err != nil {
		t.Fatal(err)
	}
	store.SeedReportAs(sub.ID, "claude-code", "Handed back for review", fixedNow().Add(-time.Minute))
	app.Refresh()
	return app, store
}

func TestScreensMine(t *testing.T) {
	screensTest(t)
	for _, a := range appearances() {
		app, _ := mineApp(t)
		mainShot(t, "mine-"+a.suffix+".png", app, a.dark, nil)
		mainShot(t, "agents-"+a.suffix+".png", app, a.dark, func(tt *ui.Tester, m *Main) { m.goTo(nav{screen: screenAgents}) })
		mainShot(t, "inspector-parent-"+a.suffix+".png", app, a.dark, func(tt *ui.Tester, m *Main) {
			m.goTo(nav{screen: screenAgents})
			m.OpenTask(1)
		})
		mainShot(t, "inspector-subtask-"+a.suffix+".png", app, a.dark, func(tt *ui.Tester, m *Main) {
			m.goTo(nav{screen: screenAgents})
			m.OpenTask(23)
		})
		zhShot(t, "mine-zh-"+a.suffix+".png", app, a.dark, nil)
		// Narrower than the lists and the Agents panel need: the panel is gone, its section is inline.
		mainShot(t, "mine-narrow-"+a.suffix+".png", app, a.dark, func(tt *ui.Tester, m *Main) { tt.SetSize(860, 720) })
		zhShot(t, "mine-narrow-zh-"+a.suffix+".png", app, a.dark, func(tt *ui.Tester, m *Main) { tt.SetSize(860, 720) })
		zhShot(t, "agents-zh-"+a.suffix+".png", app, a.dark, func(tt *ui.Tester, m *Main) { m.goTo(nav{screen: screenAgents}) })
		zhShot(t, "inspector-parent-zh-"+a.suffix+".png", app, a.dark, func(tt *ui.Tester, m *Main) {
			m.goTo(nav{screen: screenAgents})
			m.OpenTask(1)
		})
		empty, _ := newEmptyApp(t)
		mainShot(t, "agents-empty-"+a.suffix+".png", empty, a.dark, func(tt *ui.Tester, m *Main) { m.goTo(nav{screen: screenAgents}) })
	}
}

// TestScreensTheHop renders the sidebar with the hop in flight and after it has landed: the clock is
// the test's, so the frame is the one at 40 % of the flight.
func TestScreensTheHop(t *testing.T) {
	screensTest(t)
	for _, a := range appearances() {
		for _, at := range []struct {
			name string
			in   time.Duration
		}{{"hop-rise", 200 * time.Millisecond}, {"hop-peak", 330 * time.Millisecond}, {"hop-landed", 700 * time.Millisecond}, {"hop-settled", 2 * time.Second}} {
			app, _ := mineApp(t)
			clock := fixedNow()
			app.Now = func() time.Time { return clock }
			m := NewMain(app)
			m.nav = nav{screen: screenAgents}
			tt := ui.NewTester(m.View, 1100, 720)
			tt.SetScale(2)
			tt.SetDark(a.dark)
			tt.SetTitleBar(ui.TitleBar{Height: 52, Left: 78})
			tt.SetPreferences(ui.Preferences{})
			tt.Frame() // the sidebar rows are laid out: the next frame launches the hops
			tt.Frame()
			clock = clock.Add(at.in)
			settle(tt)
			shot(t, at.name+"-"+a.suffix+".png", tt.Image(), false, a.dark)
		}
	}
}

// TestScreensFilterAndGroups renders Today narrowed to Work by a chip (the filter bar), Agents grouped
// by agent (Codex works on one task, so there are two groups), and the Category row's strip open in
// the inspector.
func TestScreensFilterAndGroups(t *testing.T) {
	screensTest(t)
	for _, a := range appearances() {
		app, store := mineApp(t)
		store.SeedReportAs(15, "codex", "Writing tests for the reply parser", fixedNow().Add(-time.Minute))
		app.Refresh()
		mainShot(t, "mine-filter-"+a.suffix+".png", app, a.dark, func(tt *ui.Tester, m *Main) { m.catFilter = workID })
		mainShot(t, "mine-filter-life-"+a.suffix+".png", app, a.dark, func(tt *ui.Tester, m *Main) { m.catFilter = lifeID })
		mainShot(t, "agents-codex-"+a.suffix+".png", app, a.dark, func(tt *ui.Tester, m *Main) { m.goTo(nav{screen: screenAgents}) })
		zhShot(t, "mine-filter-zh-"+a.suffix+".png", app, a.dark, func(tt *ui.Tester, m *Main) { m.catFilter = workID })
		mainShot(t, "inspector-category-"+a.suffix+".png", app, a.dark, func(tt *ui.Tester, m *Main) {
			m.OpenTask(18)
			tt.Frame()
			click(t, tt, "Category: None")
		})
	}
}

// TestScreensVisibilityGaps renders the rows and folds that keep every task on Today or Agents: a
// task of mine in review, the No date fold open, and the queued tasks and Backlog fold of
// Agents.
func TestScreensVisibilityGaps(t *testing.T) {
	screensTest(t)
	for _, a := range appearances() {
		app, store := mineApp(t)
		far := func(days int) *time.Time { return data.TimePtr(fixedNow().Add(time.Duration(days) * 24 * time.Hour)) }
		store.CreateTask(data.CreateTaskInput{Title: "Check the draft of the launch post", Status: data.Review, Priority: data.P1})
		store.CreateTask(data.CreateTaskInput{Title: "Renew the passport", DueAt: far(40)})
		store.CreateTask(data.CreateTaskInput{Title: "Book flights for the conference", DueAt: far(15)})
		store.CreateTask(data.CreateTaskInput{Title: "Draft the changelog", Status: data.Todo, Source: data.SourceAgent})
		store.CreateTask(data.CreateTaskInput{Title: "Try a faster diff algorithm", Status: data.Backlog, Source: data.SourceAgent})
		app.Refresh()
		mainShot(t, "mine-review-"+a.suffix+".png", app, a.dark, nil)
		mainShot(t, "mine-no-date-open-"+a.suffix+".png", app, a.dark, func(tt *ui.Tester, m *Main) {
			app.noDateOpen = true
		})
		app.noDateOpen = false
		mainShot(t, "agents-queued-"+a.suffix+".png", app, a.dark, func(tt *ui.Tester, m *Main) { m.goTo(nav{screen: screenAgents}) })
		zhShot(t, "agents-queued-zh-"+a.suffix+".png", app, a.dark, func(tt *ui.Tester, m *Main) { m.goTo(nav{screen: screenAgents}) })
	}
}

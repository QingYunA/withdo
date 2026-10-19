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
func mineApp(t *testing.T) (*App, *data.Fake) {
	t.Helper()
	store := data.NewFake(fixedNow)
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

// TestScreensSubRows renders the sidebar's sub-rows: the Work page of Mine, an agent's page of Agents
// (Codex works on one task, so there are two agents to list), and the Category row's strip open in the
// inspector.
func TestScreensSubRows(t *testing.T) {
	screensTest(t)
	for _, a := range appearances() {
		app, store := mineApp(t)
		store.SeedReportAs(15, "codex", "Writing tests for the reply parser", fixedNow().Add(-time.Minute))
		app.Refresh()
		mainShot(t, "mine-work-"+a.suffix+".png", app, a.dark, func(tt *ui.Tester, m *Main) { m.goTo(nav{screen: screenMine, cat: workID}) })
		mainShot(t, "mine-life-"+a.suffix+".png", app, a.dark, func(tt *ui.Tester, m *Main) { m.goTo(nav{screen: screenMine, cat: lifeID}) })
		mainShot(t, "agents-codex-"+a.suffix+".png", app, a.dark, func(tt *ui.Tester, m *Main) { m.goTo(nav{screen: screenAgents, sub: "Codex"}) })
		zhShot(t, "mine-work-zh-"+a.suffix+".png", app, a.dark, func(tt *ui.Tester, m *Main) { m.goTo(nav{screen: screenMine, cat: workID}) })
		mainShot(t, "inspector-category-"+a.suffix+".png", app, a.dark, func(tt *ui.Tester, m *Main) {
			m.OpenTask(18)
			tt.Frame()
			click(t, tt, "Category: None")
		})
	}
}

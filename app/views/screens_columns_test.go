package views

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// The two-column project screens (docs/design/mockups/v7.html and v7b.html), in light and dark, in
// English and Simplified Chinese. Render with
//
//	SCREENS_OUT=$PWD/docs/design/screens go test ./app/views -run TestScreensColumns

// columnsShot renders a screen in the language: the Chinese ones carry -zh in their names.
func columnsShot(t *testing.T, name string, zh bool, app *App, dark bool, size [2]int, setup func(tt *ui.Tester, m *Main)) {
	t.Helper()
	m := NewMain(app)
	tt := ui.NewTester(m.View, size[0], size[1])
	tt.SetScale(2)
	tt.SetDark(dark)
	tt.SetTitleBar(ui.TitleBar{Height: 52, Left: 78})
	still(tt)
	if zh {
		inChinese(tt)
	}
	if setup != nil {
		setup(tt, m)
		tt.Frame()
	}
	settle(tt)
	shot(t, name, tt.Image(), false, dark)
}

// handedApp is the seeded workspace after the user handed "Write the release notes" to Claude Code.
func handedApp(t *testing.T) (*App, *data.Real) {
	t.Helper()
	app, store, _ := newTestApp(t)
	if _, err := store.HandOff(5); err != nil {
		t.Fatal(err)
	}
	app.Refresh()
	return app, store
}

// onlyMineApp is a project in which no agent has reported, with a few tasks of the user's.
func onlyMineApp(t *testing.T) (*App, data.Project) {
	t.Helper()
	f := demoStore(t, fixedNow, false)
	p := f.SeedProject("answer-me-with-html", "me/answer-me-with-html")
	for i, task := range []struct {
		title  string
		status data.Status
	}{{"Split page.ts into three files", data.Doing}, {"Write the CLI help", data.Todo}, {"Add progress to the video export", data.Todo}, {"Ship v0.3", data.Done}} {
		_, _ = f.CreateTask(data.CreateTaskInput{Title: task.title, Status: task.status, ProjectID: data.IntPtr(p.ID), Source: data.SourceUser})
		_ = i
	}
	app := NewApp(f, &recorder{}, testConfig())
	app.Now = fixedNow
	return app, p
}

func TestScreensColumns(t *testing.T) {
	screensTest(t)
	full := [2]int{1100, 720}
	for _, a := range appearances() {
		for _, l := range languages {
			app, _ := handedApp(t)
			columnsShot(t, "project-two-columns"+l.suffix+"-"+a.suffix+".png", l.tag == "zh-Hans", app, a.dark, [2]int{1100, 900}, func(tt *ui.Tester, m *Main) {
				toProject(1)(tt, m)
			})
			columnsShot(t, "project-narrow"+l.suffix+"-"+a.suffix+".png", l.tag == "zh-Hans", app, a.dark, [2]int{1100, 1100}, func(tt *ui.Tester, m *Main) {
				toProject(1)(tt, m)
				m.openRow(6)
			})
			empty, p := onlyMineApp(t)
			columnsShot(t, "project-empty-agents"+l.suffix+"-"+a.suffix+".png", l.tag == "zh-Hans", empty, a.dark, full, toProject(p.ID))
		}
	}
}

// TestScreensHandOff renders the hand-off during the drag, on the test's clock: the row carried by
// the sprite over Mine (the sprite mid-step), the row over the tray with the agent mid-hop, and the
// row just landed in Waiting for an agent with the agent squashing and the wash at its strongest.
func TestScreensHandOff(t *testing.T) {
	screensTest(t)
	for _, a := range appearances() {
		for _, l := range languages {
			rig := func() *handRig {
				r := newHandRig(t, false)
				r.tt.SetScale(2)
				r.tt.SetDark(a.dark)
				if l.tag == "zh-Hans" {
					inChinese(r.tt)
				}
				r.tt.SetSize(1100, 820)
				r.tt.Frame()
				return r
			}
			name := func(what string) string { return "handoff-" + what + l.suffix + "-" + a.suffix + ".png" }

			r := rig()
			x, y := r.grab(t, "Write the release notes")
			for i := 1; i <= 3; i++ { // walking: the pointer keeps moving
				r.tt.Move(x+float32(30*i), y+float32(40*i))
				r.after(70 * time.Millisecond)
			}
			shot(t, name("drag"), r.tt.Image(), false, a.dark)

			r = rig()
			r.dragOnto(t, "Write the release notes", "Port core to Go", true)
			r.after(330 * time.Millisecond) // the top of the agent's hop
			shot(t, name("over"), r.tt.Image(), false, a.dark)

			r = rig()
			r.dragOnto(t, "Write the release notes", "Port core to Go", false)
			r.after(110 * time.Millisecond)
			settle(r.tt) // the toast fades in on the wall clock; the sprites and the wash are on the app's
			shot(t, name("landed"), r.tt.Image(), false, a.dark)
		}
	}
}

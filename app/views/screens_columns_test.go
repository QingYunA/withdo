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
func handedApp(t *testing.T) (*App, *data.Fake) {
	t.Helper()
	app, store, _ := newTestApp(t)
	if _, err := store.HandOff(5, "prompt"); err != nil {
		t.Fatal(err)
	}
	app.Refresh()
	return app, store
}

// onlyMineApp is a project in which no agent has reported, with a few tasks of the user's.
func onlyMineApp(t *testing.T) (*App, data.Project) {
	t.Helper()
	f := data.NewEmptyFake(fixedNow)
	p := f.AddProject("answer-me-with-html", "me/answer-me-with-html")
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
	full := [2]int{1100, 720}
	for _, a := range appearances() {
		for _, l := range languages {
			app, _ := handedApp(t)
			columnsShot(t, "project-two-columns"+l.suffix+"-"+a.suffix+".png", l.tag == "zh-Hans", app, a.dark, [2]int{1100, 900}, func(tt *ui.Tester, m *Main) {
				m.ho.how = map[int]bool{5: true}
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

// TestScreensHandOff renders the hand-off: a row lifted over the tray, then frames of the motion on
// the test's clock (the parcel carried, passed across, and opening into the new row).
func TestScreensHandOff(t *testing.T) {
	for _, a := range appearances() {
		for _, l := range languages {
			shotAt := func(name string, in time.Duration) {
				r := newHandRig(t, false, true)
				r.tt.SetScale(2)
				r.tt.SetDark(a.dark)
				if l.tag == "zh-Hans" {
					inChinese(r.tt)
				}
				r.tt.SetSize(1100, 820)
				r.tt.Frame()
				r.m.ho.how = map[int]bool{5: true}
				r.drag(t, "Write the release notes", "Port core to Go", in < 0)
				if in >= 0 {
					r.after(in)
					settleFrames(r.tt)
				} else {
					settle(r.tt)
				}
				shot(t, "handoff-"+name+l.suffix+"-"+a.suffix+".png", r.tt.Image(), false, a.dark)
			}
			shotAt("lift", -1)
			shotAt("carry", 700*time.Millisecond)
			shotAt("catch", 850*time.Millisecond)
			shotAt("open", 1100*time.Millisecond)
			shotAt("wash", 1500*time.Millisecond)
		}
	}
}

// settleFrames draws a few frames without waiting, so the clock the test set stays where it is.
func settleFrames(tt *ui.Tester) {
	for i := 0; i < 3; i++ {
		tt.Frame()
	}
}

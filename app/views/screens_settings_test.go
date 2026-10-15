package views

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// The Settings window, a pane at a time, in light and dark, English and Chinese (-zh). Render with
//
//	SCREENS_OUT=$PWD/docs/design/screens go test ./app/views -run TestScreensSettings

// settingsShot renders the window at 2x, as tall as its pane needs, on a pane.
func settingsShot(t *testing.T, name string, app *App, dark, zh bool, pane Pane, setup func(r *settingsRig)) {
	t.Helper()
	s := NewSettings(app)
	var height float32
	s.OnLayout = func(_ string, h float32) { height = h }
	tt := ui.NewTester(s.View, int(settingsWidth), 900)
	tt.SetScale(2)
	tt.SetDark(dark)
	still(tt)
	if zh {
		inChinese(tt)
	}
	s.Show(pane)
	r := &settingsRig{tt: tt, s: s, app: app}
	r.fm, _ = app.Models.(*data.FakeModels)
	r.rec, _ = app.Act.(*recorder)
	tt.Frame()
	tt.Frame()
	if setup != nil {
		setup(r)
		tt.Frame()
	}
	settle(tt)
	tt.SetSize(int(settingsWidth), int(height))
	settle(tt)
	shot(t, name, tt.Image(), true, dark)
}

func settingsApp(t *testing.T, st data.GitHubStatus) *App {
	t.Helper()
	app, _, _ := newTestApp(t)
	app.Models = newFakeModels(t)
	app.GitHub = data.NewFakeGitHub(st)
	mustNil(t, app.Models.Save("anthropic", data.ProviderEdit{Model: "claude-haiku-5-5", Key: "sk-ant-demo"}))
	mustNil(t, app.Models.SetActive("anthropic"))
	return app
}

func TestScreensSettings(t *testing.T) {
	ready := data.GitHubStatus{State: data.GitHubReady, Account: "QingYunA", CheckedAt: fixedNow().Add(-2 * time.Minute)}
	for _, a := range appearances() {
		for _, zh := range []bool{false, true} {
			sfx := "-" + a.suffix
			if zh {
				sfx = "-zh" + sfx
			}
			general := settingsApp(t, ready)
			mustNil(t, general.ArchiveProject(3))
			settingsShot(t, "settings-general"+sfx+".png", general, a.dark, zh, PaneGeneral, nil)
			assistant := settingsApp(t, ready)
			settingsShot(t, "settings-assistant"+sfx+".png", assistant, a.dark, zh, PaneAssistant, func(r *settingsRig) {
				r.s.runTest(currentProvider(assistant.Models.Providers())) // the status row says how it went
				r.rec.drain(t)
			})
			agents := settingsApp(t, ready)
			settingsShot(t, "settings-agents"+sfx+".png", agents, a.dark, zh, PaneAgents, nil)
			for _, s := range githubStates() {
				gh := settingsApp(t, s.st)
				settingsShot(t, "settings-github"+s.name+sfx+".png", gh, a.dark, zh, PaneGitHub, nil)
			}
		}
	}
}

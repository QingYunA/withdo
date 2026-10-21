package views

import (
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// The Recent changes page: all repositories for 7 days, one item unfolded, the repository pop-up,
// a single repository, the page with nothing in it, gh signed out, and the Settings row of the
// switch with gh missing. Light and dark, English and Chinese (-zh). Render with
//
//	SCREENS_OUT=$PWD/docs/design/screens go test ./app/views -run TestScreensRecent

// recentShot renders the page at 2x after GitHub and the assistant have answered.
func recentShot(t *testing.T, name string, zh bool, app *App, rec *recorder, dark bool, setup func(tt *ui.Tester, m *Main)) {
	t.Helper()
	shoot := mainShot
	if zh {
		shoot = zhShot
	}
	shoot(t, name, app, dark, func(tt *ui.Tester, m *Main) {
		m.goTo(nav{screen: screenRecent})
		tt.Frame()
		settleRecent(t, tt, rec)
		if setup != nil {
			setup(tt, m)
			settleRecent(t, tt, rec)
		}
	})
}

func TestScreensRecent(t *testing.T) {
	screensTest(t)
	for _, a := range appearances() {
		for _, zh := range []bool{false, true} {
			sfx := "-" + a.suffix
			if zh {
				sfx = "-zh" + sfx
			}
			shot := func(name string, app *App, rec *recorder, setup func(tt *ui.Tester, m *Main)) {
				recentShot(t, "recent"+name+sfx+".png", zh, app, rec, a.dark, setup)
			}
			// the unfolded item is PR 14, which task 20 holds
			unfold := func(tt *ui.Tester, m *Main) { m.recent.open["https://github.com/QingYunA/adhd-todo/pull/14"] = true }
			app, _, _, _, rec := recentApp(t, true)
			shot("", app, rec, nil)
			app, _, _, _, rec = recentApp(t, true)
			shot("-open", app, rec, unfold)
			app, _, _, _, rec = recentApp(t, true)
			shot("-repos", app, rec, func(tt *ui.Tester, m *Main) { _ = tt.Click(m.say.RecentRepoPopUp() + ": " + m.say.RecentAllRepos()) })
			app, _, _, _, rec = recentApp(t, true)
			app.setRecentChoice("QingYunA/picpocket", 7)
			shot("-single", app, rec, nil)

			app, _, _, _, rec = recentApp(t, true)
			app.Recent = data.NewFakeRecent()
			shot("-empty", app, rec, nil)

			app, _, read, _, rec := recentApp(t, true)
			for _, p := range app.snap.projects {
				read.Fail(p.RepoSlug, &data.RecentError{Problem: data.RecentSignedOut, Repo: p.RepoSlug, Text: "gh auth login"})
			}
			shot("-signedout", app, rec, nil)
		}
	}
}

// The switch of the page in Settings > General, on while gh is signed out: its line says what to run.
func TestScreensRecentSettings(t *testing.T) {
	screensTest(t)
	signedOut := data.GitHubStatus{State: data.GitHubSignedOut, CheckedAt: fixedNow()}
	for _, a := range appearances() {
		for _, zh := range []bool{false, true} {
			sfx := "-" + a.suffix
			if zh {
				sfx = "-zh" + sfx
			}
			app := settingsApp(t, signedOut)
			mustNil(t, app.ArchiveProject(3))
			app.SetRecentChangesOn(true)
			settingsShot(t, "recent-settings"+sfx+".png", app, a.dark, zh, PaneAdvanced, func(r *settingsRig) { r.tt.Scroll(340, 300, 0, 3000) })
		}
	}
}

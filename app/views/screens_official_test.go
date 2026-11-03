package views

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// The official model (M15, v44 section 3): Settings > Model in its three states, and the composer
// with the model chip and the quiet lines where it does not answer. Rendered at 2x in light and
// dark, in English and Chinese:
//
//	scripts/screens.sh TestScreensOfficial
func TestScreensOfficial(t *testing.T) {
	screensTest(t)
	ghReady := data.GitHubStatus{State: data.GitHubReady, Account: "QingYunA", CheckedAt: fixedNow().Add(-2 * time.Minute)}
	ready := data.OfficialInfo{Phase: data.OfficialReady, Left: 86, Limit: 100, ResetsAt: fixedNow().Add(14 * time.Hour)}
	signedOut := data.OfficialInfo{Phase: data.OfficialSignedOut, Left: -1}
	usedUp := data.OfficialInfo{Phase: data.OfficialUsedUp, Left: 0, Limit: 100, ResetsAt: fixedNow().Add(14 * time.Hour)}
	for _, a := range appearances() {
		for _, zh := range []bool{false, true} {
			sfx := "-" + a.suffix
			if zh {
				sfx = "-zh" + sfx
			}
			// Settings: the official card in use (no provider of one's own chosen), signed out, and
			// a provider in use with the official model as the row under it.
			settingsWith := func(info data.OfficialInfo, own bool) *App {
				app := settingsApp(t, ghReady)
				app.Models.(*data.FakeModels).Info = info
				if !own {
					mustNil(t, app.Models.UseOfficial())
				}
				return app
			}
			settingsShot(t, "settings-model-official"+sfx+".png", settingsWith(ready, false), a.dark, zh, PaneModel, nil)
			settingsShot(t, "settings-model-official-signed-out"+sfx+".png", settingsWith(signedOut, false), a.dark, zh, PaneModel, nil)
			settingsShot(t, "settings-model-own-with-official"+sfx+".png", settingsWith(ready, true), a.dark, zh, PaneModel, nil)

			shotOf := func(name string, app *App, setup func(tt *ui.Tester, m *Main)) {
				if zh {
					zhShot(t, name+sfx+".png", app, a.dark, setup)
					return
				}
				mainShot(t, name+sfx+".png", app, a.dark, setup)
			}
			// The composer: the chip names the official model (a project page, where the chip is), and
			// the quiet line where it does not answer.
			text := "remind me to call the landlord tomorrow at 9"
			if zh {
				text = "明天上午九点提醒我给房东打电话"
			}
			withChip, _, _ := assistApp(t, &data.FakeAssistant{Ready: true})
			withChip.Models.(*data.FakeModels).Info = ready
			shotOf("project-composer-official", withChip, func(tt *ui.Tester, m *Main) {
				m.goTo(nav{screen: screenProject, projectID: 1})
				tt.Frame()
				tt.Key(ui.Cmd, ui.KeyK)
				tt.Type(text)
			})
			out, _, _ := assistApp(t, &data.FakeAssistant{Ready: false})
			out.Models.(*data.FakeModels).Info = signedOut
			shotOf("mine-composer-official-signed-out", out, func(tt *ui.Tester, m *Main) { tt.Key(ui.Cmd, ui.KeyK) })
			spent, _, _ := assistApp(t, &data.FakeAssistant{Ready: false})
			spent.Models.(*data.FakeModels).Info = usedUp
			shotOf("mine-composer-official-used-up", spent, func(tt *ui.Tester, m *Main) { tt.Key(ui.Cmd, ui.KeyK) })
		}
	}
}

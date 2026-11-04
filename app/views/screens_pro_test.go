package views

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// Withdo Pro on the desktop (M16, v45 section 1): the official card with its plan, the upgrade sheet
// and its waiting state, a Pro account, and the composer line that offers Pro. Rendered at 2x in
// light and dark, in English and Chinese:
//
//	scripts/screens.sh TestScreensPro
func TestScreensPro(t *testing.T) {
	screensTest(t)
	ghReady := data.GitHubStatus{State: data.GitHubReady, Account: "QingYunA", CheckedAt: fixedNow().Add(-2 * time.Minute)}
	free := data.OfficialInfo{Phase: data.OfficialReady, Left: 14, Limit: 20, ResetsAt: fixedNow().Add(14 * time.Hour)}
	spent := data.OfficialInfo{Phase: data.OfficialUsedUp, Left: 0, Limit: 20, ResetsAt: fixedNow().Add(14 * time.Hour)}
	pro := data.OfficialInfo{Phase: data.OfficialReady, Left: 86, Limit: 100, ResetsAt: fixedNow().Add(14 * time.Hour)}
	onFree := data.BillingInfo{Offered: true, Known: true, Monthly: "$6.99", Yearly: "$58.72"}
	onPro := data.BillingInfo{
		Offered: true, Known: true, Pro: true, Status: "trialing", PeriodEnd: fixedNow().Add(7 * 24 * time.Hour),
		Interval: "yearly", TrialUsed: true, Monthly: "$6.99", Yearly: "$58.72",
	}
	onFreePayPal := onFree
	onFreePayPal.PayPal = true
	onProPayPal := onPro
	onProPayPal.PayPal, onProPayPal.Provider = true, data.ProviderPayPal
	for _, a := range appearances() {
		for _, zh := range []bool{false, true} {
			sfx := "-" + a.suffix
			if zh {
				sfx = "-zh" + sfx
			}
			settingsWith := func(info data.OfficialInfo, bill data.BillingInfo) *App {
				app := settingsApp(t, ghReady)
				fm := app.Models.(*data.FakeModels)
				fm.Info, fm.Bill = info, bill
				mustNil(t, app.Models.UseOfficial())
				return app
			}
			settingsShot(t, "settings-model-pro-free"+sfx+".png", settingsWith(free, onFree), a.dark, zh, PaneModel, nil)
			settingsShot(t, "settings-model-pro-sheet"+sfx+".png", settingsWith(free, onFree), a.dark, zh, PaneModel,
				func(r *settingsRig) { r.s.openUpgrade() })
			settingsShot(t, "settings-model-pro-waiting"+sfx+".png", settingsWith(free, onFree), a.dark, zh, PaneModel,
				func(r *settingsRig) {
					r.s.openUpgrade()
					r.tt.Frame()
					mustNil(t, r.tt.Click(r.s.say.ProStartTrial()))
					r.rec.drain(t)
				})
			settingsShot(t, "settings-model-pro"+sfx+".png", settingsWith(pro, onPro), a.dark, zh, PaneModel, nil)

			// PayPal as a second way to pay (M17, v46): the button under the main one, the wait for
			// PayPal, and a Pro account billed by PayPal.
			settingsShot(t, "settings-model-pro-sheet-paypal"+sfx+".png", settingsWith(free, onFreePayPal), a.dark, zh, PaneModel,
				func(r *settingsRig) { r.s.openUpgrade() })
			settingsShot(t, "settings-model-pro-waiting-paypal"+sfx+".png", settingsWith(free, onFreePayPal), a.dark, zh, PaneModel,
				func(r *settingsRig) {
					r.s.openUpgrade()
					r.tt.Frame()
					mustNil(t, r.tt.Click(r.s.say.ProPayPal()))
					r.rec.drain(t)
				})
			settingsShot(t, "settings-model-pro-paypal"+sfx+".png", settingsWith(pro, onProPayPal), a.dark, zh, PaneModel, nil)

			// The composer's quiet line past the day's turns: it ends with the way to Pro.
			app, _, _ := assistApp(t, &data.FakeAssistant{Ready: false})
			fm := app.Models.(*data.FakeModels)
			fm.Info, fm.Bill = spent, onFree
			setup := func(tt *ui.Tester, m *Main) { tt.Key(ui.Cmd, ui.KeyK) }
			if zh {
				zhShot(t, "mine-composer-official-used-up-upgrade"+sfx+".png", app, a.dark, setup)
			} else {
				mainShot(t, "mine-composer-official-used-up-upgrade"+sfx+".png", app, a.dark, setup)
			}
		}
	}
}

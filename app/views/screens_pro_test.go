package views

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// Withdo Pro on the desktop (M16, v45; the plan moved to Account in v47): the plan group of Settings >
// Account in each status, the upgrade sheet over it and its waiting state, the official card in Model,
// and the composer line that offers Pro. Rendered at 2x in light and dark, in English and Chinese:
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
	onProCanceling := onPro
	onProCanceling.Status, onProCanceling.PeriodEnd = "canceling", fixedNow().Add(14*24*time.Hour)
	onProCanceling.PayPal, onProCanceling.Provider = true, data.ProviderPayPal
	onProActive := onPro
	onProActive.Status, onProActive.PeriodEnd = "active", fixedNow().Add(365*24*time.Hour)
	onProPastDue := onPro
	onProPastDue.Status, onProPastDue.PeriodEnd = "past_due", fixedNow().Add(14*24*time.Hour)
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
			// The plan in Account (v47): Free, a trial, canceled at PayPal, paying, a failed renewal; the upgrade
			// sheet over it, and its wait, with and without PayPal's button.
			account := func(info data.OfficialInfo, bill data.BillingInfo) *App {
				app := settingsWith(info, bill)
				app.Sync = &fakeSync{snap: syncedSync()}
				return app
			}
			settingsShot(t, "settings-account-plan-free"+sfx+".png", account(free, onFree), a.dark, zh, PaneAccount, nil)
			settingsShot(t, "settings-account-plan-trial"+sfx+".png", account(pro, onPro), a.dark, zh, PaneAccount, nil)
			settingsShot(t, "settings-account-plan-canceling-paypal"+sfx+".png", account(pro, onProCanceling), a.dark, zh, PaneAccount, nil)
			settingsShot(t, "settings-account-plan-active"+sfx+".png", account(pro, onProActive), a.dark, zh, PaneAccount, nil)
			settingsShot(t, "settings-account-plan-past-due"+sfx+".png", account(pro, onProPastDue), a.dark, zh, PaneAccount, nil)
			settingsShot(t, "settings-account-plan-sheet"+sfx+".png", account(free, onFree), a.dark, zh, PaneAccount,
				func(r *settingsRig) { r.s.openUpgrade() })
			settingsShot(t, "settings-account-plan-waiting"+sfx+".png", account(free, onFree), a.dark, zh, PaneAccount,
				func(r *settingsRig) {
					r.s.openUpgrade()
					r.tt.Frame()
					mustNil(t, r.tt.Click(r.s.say.ProStartTrial()))
					r.rec.drain(t)
				})
			// PayPal as a second way to pay (M17, v46): the button under the main one and the wait for PayPal.
			settingsShot(t, "settings-account-plan-sheet-paypal"+sfx+".png", account(free, onFreePayPal), a.dark, zh, PaneAccount,
				func(r *settingsRig) { r.s.openUpgrade() })
			settingsShot(t, "settings-account-plan-waiting-paypal"+sfx+".png", account(free, onFreePayPal), a.dark, zh, PaneAccount,
				func(r *settingsRig) {
					r.s.openUpgrade()
					r.tt.Frame()
					mustNil(t, r.tt.Click(r.s.say.ProPayPal()))
					r.rec.drain(t)
				})

			// The official card in Model keeps the plan pill and the quota; Free adds the link to Pro.
			settingsShot(t, "settings-model-pro-free"+sfx+".png", settingsWith(free, onFree), a.dark, zh, PaneModel, nil)
			settingsShot(t, "settings-model-pro"+sfx+".png", settingsWith(pro, onPro), a.dark, zh, PaneModel, nil)

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

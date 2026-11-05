package views

import (
	"errors"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// Withdo Pro on the desktop (M16, v45; v47 moves the plan to Account): the plan on Settings > Account,
// the upgrade sheet over it and its waiting state, the plan pill and link on the official card, and
// the composer line that offers Pro.

var (
	freeInfo = data.OfficialInfo{Phase: data.OfficialReady, Left: 14, Limit: 20, ResetsAt: fixedNow().Add(14 * time.Hour)}
	freeBill = data.BillingInfo{Offered: true, Known: true, Monthly: "$6.99", Yearly: "$58.72"}
	proInfo  = data.OfficialInfo{Phase: data.OfficialReady, Left: 86, Limit: 100, ResetsAt: fixedNow().Add(14 * time.Hour)}
)

// proRig is Settings > Model with the official model in use and billing as set.
func proRig(t *testing.T, info data.OfficialInfo, bill data.BillingInfo) *settingsRig {
	t.Helper()
	r := newSettingsRig(t, func(a *App) { a.Sync = &fakeSync{snap: syncedSync()} })
	r.fm.Info, r.fm.Bill = info, bill
	mustNil(t, r.fm.UseOfficial())
	r.assistant(t)
	return r
}

// accountRig is Settings > Account of a signed-in account with billing as set.
func accountRig(t *testing.T, info data.OfficialInfo, bill data.BillingInfo) *settingsRig {
	t.Helper()
	r := newSettingsRig(t, func(a *App) { a.Sync = &fakeSync{snap: syncedSync()} })
	r.fm.Info, r.fm.Bill = info, bill
	mustNil(t, r.fm.UseOfficial())
	r.s.Show(PaneAccount)
	r.tt.Frame()
	r.tt.Frame()
	return r
}

func TestTheFreeCardShowsThePlanTheQuotaAgainstItsLimitAndTheLinkToPro(t *testing.T) {
	r := proRig(t, freeInfo, freeBill)
	has(t, r.tt, "Free", "Available", "· 14 / 20 left today", "Upgrade to Pro")
	hasNot(t, r.tt, "Included with your account", "Manage subscription", "Pro: 100 a day", "Try it free for 7 days")
	if r.s.pro.open {
		t.Fatal("the sheet opened by itself")
	}
}

func TestTheUpgradeLinkOnTheCardOpensAccountWithTheSheet(t *testing.T) {
	r := proRig(t, freeInfo, freeBill)
	link, _ := r.tt.Find("Upgrade to Pro")
	lx, ly := center(link)
	r.tt.Move(lx, ly)
	mustNil(t, r.tt.Click("Upgrade to Pro"))
	r.tt.Frame()
	r.tt.Frame()
	if r.s.pane != PaneAccount {
		t.Fatalf("pane %v", r.s.pane)
	}
	has(t, r.tt, "Upgrade to Withdo Pro", "Start 7-day free trial")
}

func TestWithoutBillingTheCardIsAsInV44(t *testing.T) {
	r := proRig(t, proInfo, data.BillingInfo{})
	has(t, r.tt, "Included with your account", "· 86 left today")
	hasNot(t, r.tt, "Free", "Upgrade to Pro", "Manage subscription")

	// Offered, but the server has not said more: the plan pill is there, the link waits.
	r.fm.Bill = data.BillingInfo{Offered: true}
	r.tt.Frame()
	has(t, r.tt, "Free")
	hasNot(t, r.tt, "Upgrade to Pro", "Included with your account")
}

func TestTheProCardKeepsThePillAndTheQuotaOnly(t *testing.T) {
	pro := data.BillingInfo{
		Offered: true, Known: true, Pro: true, Status: "active", Interval: "yearly",
		PeriodEnd: fixedNow().Add(7 * 24 * time.Hour), TrialUsed: true,
	}
	r := proRig(t, proInfo, pro)
	has(t, r.tt, "Pro", "· 86 / 100 left today")
	hasNot(t, r.tt, "Upgrade to Pro", "Free", "Manage subscription", "Withdo Pro", "Yearly")
}

func TestAccountShowsThePlanAsTheLastRowOfTheAccountGroup(t *testing.T) {
	r := accountRig(t, freeInfo, freeBill)
	has(t, r.tt, "Free", "20 a day on the official model. Pro: 100 a day, 7-day free trial.", "Upgrade to Pro", "Devices")
	hasNot(t, r.tt, "Manage subscription", "Withdo Pro", "Plan")
	google, _ := r.tt.Find("Google")
	plan, _ := r.tt.Find("Upgrade to Pro")
	devices, _ := r.tt.Find("Devices")
	if !(google.Y < plan.Y && plan.Y < devices.Y) {
		t.Errorf("order: google %v, plan %v, devices %v", google.Y, plan.Y, devices.Y)
	}

	used := freeBill
	used.TrialUsed = true
	r.fm.Bill = used
	r.tt.Frame()
	has(t, r.tt, "20 a day on the official model. Pro: 100 a day on every device.")
	hasNot(t, r.tt, "7-day free trial.")
}

func TestThePlanRowIsNotThereWithoutAPlanToSell(t *testing.T) {
	for name, bill := range map[string]data.BillingInfo{
		"no billing": {},
		"unknown":    {Offered: true},
	} {
		r := accountRig(t, freeInfo, bill)
		if r.tt.HasText("Free") || r.tt.HasText("Upgrade to Pro") {
			t.Errorf("%s: the plan row is shown", name)
		}
		has(t, r.tt, "Devices")
	}

	r := newSettingsRig(t, func(a *App) { a.Sync = &fakeSync{snap: signedOutCodeSync()} })
	r.fm.Info, r.fm.Bill = freeInfo, freeBill
	r.s.Show(PaneAccount)
	r.tt.Frame()
	hasNot(t, r.tt, "Upgrade to Pro", "20 a day on the official model.")
}

func TestThePlanRowTagsTheStatusAndTheButtonManagesTheSubscription(t *testing.T) {
	for _, c := range []struct{ status, tag string }{
		{"trialing", "Free trial"},
		{"active", "Pro"},
		{"canceling", "Canceled"},
		{"past_due", "Payment failed"},
	} {
		pro := data.BillingInfo{
			Offered: true, Known: true, Pro: true, Status: c.status, Interval: "yearly", Yearly: "$58.72",
			PeriodEnd: fixedNow().Add(7 * 24 * time.Hour), TrialUsed: true,
		}
		r := accountRig(t, proInfo, pro)
		has(t, r.tt, "Withdo Pro", c.tag, "Manage subscription")
		hasNot(t, r.tt, "Upgrade to Pro", "20 a day")
	}
}

// The plan row's line for a Pro account, by status, interval and provider, in both languages.
// fixedNow is in Oct 2026.
func TestThePlanLineByStatusIntervalAndProvider(t *testing.T) {
	soon, later := fixedNow().Add(7*24*time.Hour), time.Date(2027, 10, 21, 0, 0, 0, 0, time.Local)
	bill := func(status, interval, provider string, end time.Time) data.BillingInfo {
		return data.BillingInfo{
			Offered: true, Known: true, Pro: true, Status: status, Interval: interval, Provider: provider, PeriodEnd: end,
			Monthly: "$6.99", Yearly: "$58.72", TrialUsed: true,
		}
	}
	noPrice := bill("active", "yearly", "", soon)
	noPrice.Yearly = ""
	for _, c := range []struct {
		name   string
		bill   data.BillingInfo
		en, zh string
	}{
		{"trialing yearly", bill("trialing", "yearly", "", soon),
			"Yearly $58.72 · First charge Oct 21", "年付 $58.72 · 10月21日开始扣费，之前随时可以取消"},
		{"trialing monthly paypal", bill("trialing", "monthly", data.ProviderPayPal, soon),
			"Monthly $6.99 · via PayPal · First charge Oct 21", "月付 $6.99 · 通过 PayPal · 10月21日开始扣费，之前随时可以取消"},
		{"active yearly", bill("active", "yearly", data.ProviderWaffo, later),
			"Yearly $58.72 · Renews Oct 21, 2027", "年付 $58.72 · 下次续费 2027年10月21日"},
		{"active monthly", bill("active", "monthly", "", soon),
			"Monthly $6.99 · Renews Oct 21", "月付 $6.99 · 下次续费 10月21日"},
		{"canceling yearly", bill("canceling", "yearly", "", soon),
			"Yearly · Pro until Oct 21; no more charges", "年付 · Pro 用到 10月21日，之后回到免费版，不会再扣费"},
		{"canceling paypal", bill("canceling", "yearly", data.ProviderPayPal, soon),
			"Yearly · via PayPal · Pro until Oct 21; no more charges", "年付 · 通过 PayPal · Pro 用到 10月21日，之后回到免费版，不会再扣费"},
		{"past due", bill("past_due", "monthly", "", soon),
			"Monthly · Renewal failed; update the payment method", "月付 · 续费没扣成功，请在订阅里更新付款方式"},
		{"past due paypal", bill("past_due", "yearly", data.ProviderPayPal, soon),
			"Yearly · via PayPal · Renewal failed; update the payment method", "年付 · 通过 PayPal · 续费没扣成功，请在订阅里更新付款方式"},
		{"no interval", bill("active", "", "", soon), "Renews Oct 21", "下次续费 10月21日"},
		{"no interval paypal", bill("trialing", "", data.ProviderPayPal, soon),
			"via PayPal · First charge Oct 21", "通过 PayPal · 10月21日开始扣费，之前随时可以取消"},
		{"no date", bill("active", "yearly", "", time.Time{}), "Yearly $58.72", "年付 $58.72"},
		{"no price", noPrice, "Yearly · Renews Oct 21", "年付 · 下次续费 10月21日"},
	} {
		r := accountRig(t, proInfo, c.bill)
		if got := r.s.planLine(c.bill); got != c.en {
			t.Errorf("%s: %q, want %q", c.name, got, c.en)
		}
		inChinese(r.tt)
		r.tt.Frame()
		if got := r.s.planLine(c.bill); got != c.zh {
			t.Errorf("%s: %q, want %q", c.name, got, c.zh)
		}
	}
}

func TestManageSubscriptionOpensAFreshPortalLinkEachTime(t *testing.T) {
	pro := data.BillingInfo{Offered: true, Known: true, Pro: true, Status: "active", PeriodEnd: fixedNow().Add(30 * 24 * time.Hour)}
	r := accountRig(t, proInfo, pro)
	n := 0
	r.fm.ManageFn = func() (string, error) { n++; return "https://portal.example.com/" + itoa(n), nil }
	for want := 1; want <= 2; want++ {
		mustNil(t, r.tt.Click("Manage subscription"))
		r.rec.drain(t)
		r.tt.Frame()
		if got := r.rec.urls; len(got) != want || got[want-1] != "https://portal.example.com/"+itoa(want) {
			t.Fatalf("opened %v after %d clicks", got, want)
		}
	}
	r.fm.ManageFn = func() (string, error) { return "", errors.New("down") }
	mustNil(t, r.tt.Click("Manage subscription"))
	r.rec.drain(t)
	r.tt.Frame()
	has(t, r.tt, "Could not open the subscription page. Try again.")
	if len(r.rec.urls) != 2 {
		t.Fatalf("a failed link opened %v", r.rec.urls)
	}
}

func TestUpgradeOpensTheSheetOnYearlyAndMonthlyChangesThePrice(t *testing.T) {
	r := accountRig(t, freeInfo, freeBill)
	mustNil(t, r.tt.Click("Upgrade to Pro"))
	r.tt.Frame()
	has(t, r.tt, "Upgrade to Withdo Pro", "The official model, 100 a day, on every device.", "Monthly", "Yearly", "Save 30%",
		"$58.72", "/ year, about $4.89 / month", "Official model 100 a day (20 on Free)", "Works on your phone and computer",
		"Sync stays free", "Cancel any time in the customer portal", "Start 7-day free trial",
		"Nothing is charged before the trial ends. Sold by Waffo, tax included.")
	mustNil(t, r.tt.Click("Monthly"))
	r.tt.Frame()
	has(t, r.tt, "$6.99", "/ month")
	hasNot(t, r.tt, "$58.72")
}

func TestTheSheetOffersSubscribeOnceTheTrialIsUsed(t *testing.T) {
	bill := freeBill
	bill.TrialUsed = true
	r := accountRig(t, freeInfo, bill)
	r.s.openUpgrade()
	r.tt.Frame()
	has(t, r.tt, "Subscribe", "Sold by Waffo, tax included.")
	hasNot(t, r.tt, "Start 7-day free trial", "Nothing is charged before the trial ends. Sold by Waffo, tax included.")
}

func TestStartingTheTrialOpensTheBrowserAndTheSheetWaitsUntilProArrives(t *testing.T) {
	r := accountRig(t, freeInfo, freeBill)
	var plans []string
	r.fm.CheckoutFn = func(plan string) (string, error) {
		plans = append(plans, plan)
		return "https://pay.example.com/c/" + plan, nil
	}
	r.s.openUpgrade()
	r.tt.Frame()
	mustNil(t, r.tt.Click("Start 7-day free trial"))
	r.rec.drain(t)
	r.tt.Frame()
	if len(plans) != 1 || plans[0] != data.PlanYearly || len(r.rec.urls) != 1 || r.rec.urls[0] != "https://pay.example.com/c/yearly" {
		t.Fatalf("plans %v, opened %v", plans, r.rec.urls)
	}
	has(t, r.tt, "Finish paying in your browser", "Reopen the payment page", "Cancel")
	hasNot(t, r.tt, "Start 7-day free trial")

	mustNil(t, r.tt.Click("Reopen the payment page"))
	if len(r.rec.urls) != 2 || r.rec.urls[1] != r.rec.urls[0] {
		t.Fatalf("Reopen opened %v", r.rec.urls)
	}

	// The server says the plan changed: the sheet closes by itself.
	r.fm.Bill = data.BillingInfo{Offered: true, Known: true, Pro: true, Status: "trialing", PeriodEnd: fixedNow().Add(7 * 24 * time.Hour), TrialUsed: true}
	r.fm.Info = proInfo
	r.tt.Frame()
	r.tt.Frame()
	hasNot(t, r.tt, "Finish paying in your browser")
	has(t, r.tt, "Manage subscription")
}

func TestTheMonthlySegmentAsksForTheMonthlyPlan(t *testing.T) {
	r := accountRig(t, freeInfo, freeBill)
	var plan string
	r.fm.CheckoutFn = func(p string) (string, error) { plan = p; return "https://pay.example.com", nil }
	r.s.openUpgrade()
	r.tt.Frame()
	mustNil(t, r.tt.Click("Monthly"))
	r.tt.Frame()
	mustNil(t, r.tt.Click("Start 7-day free trial"))
	r.rec.drain(t)
	if plan != data.PlanMonthly {
		t.Fatalf("asked for %q", plan)
	}
}

func TestCancelInTheWaitingSheetClosesIt(t *testing.T) {
	r := accountRig(t, freeInfo, freeBill)
	r.s.openUpgrade()
	r.tt.Frame()
	mustNil(t, r.tt.Click("Start 7-day free trial"))
	r.rec.drain(t)
	r.tt.Frame()
	mustNil(t, r.tt.Click("Cancel"))
	r.tt.Frame()
	hasNot(t, r.tt, "Finish paying in your browser", "Upgrade to Withdo Pro")
	if r.s.pro.open {
		t.Fatal("the sheet is still open")
	}
}

func TestCheckoutErrorsStayInTheSheet(t *testing.T) {
	for _, c := range []struct {
		err  error
		want string
	}{
		{errors.New("boom"), "Could not open the payment page. Try again."},
		{data.ErrCheckoutThrottled, "Too many tries. Wait a little and try again."},
	} {
		r := accountRig(t, freeInfo, freeBill)
		r.fm.CheckoutFn = func(string) (string, error) { return "", c.err }
		r.s.openUpgrade()
		r.tt.Frame()
		mustNil(t, r.tt.Click("Start 7-day free trial"))
		r.rec.drain(t)
		r.tt.Frame()
		has(t, r.tt, "Upgrade to Withdo Pro", c.want, "Start 7-day free trial")
		hasNot(t, r.tt, "Finish paying in your browser")
		if len(r.rec.urls) != 0 {
			t.Fatalf("a failed checkout opened %v", r.rec.urls)
		}
	}
}

func TestAnAccountThatIsProAlreadyClosesTheSheet(t *testing.T) {
	r := accountRig(t, freeInfo, freeBill)
	r.fm.CheckoutFn = func(string) (string, error) { return "", data.ErrAlreadySubscribed }
	r.s.openUpgrade()
	r.tt.Frame()
	mustNil(t, r.tt.Click("Start 7-day free trial"))
	r.rec.drain(t)
	r.tt.Frame()
	hasNot(t, r.tt, "Upgrade to Withdo Pro", "Finish paying in your browser")
}

func TestEscapeClosesTheSheet(t *testing.T) {
	r := accountRig(t, freeInfo, freeBill)
	r.s.openUpgrade()
	r.tt.Frame()
	r.tt.Key(0, ui.KeyEscape)
	r.tt.Frame()
	hasNot(t, r.tt, "Upgrade to Withdo Pro")
}

func TestShowUpgradeOpensAccountWithTheSheet(t *testing.T) {
	r := newSettingsRig(t, func(a *App) { a.Sync = &fakeSync{snap: syncedSync()} })
	r.fm.Info, r.fm.Bill = freeInfo, freeBill
	mustNil(t, r.fm.UseOfficial())
	r.s.ShowUpgrade()
	r.tt.Frame()
	if r.s.pane != PaneAccount {
		t.Fatalf("pane %v", r.s.pane)
	}
	has(t, r.tt, "Upgrade to Withdo Pro")
}

func TestShowUpgradeDoesNothingWhereThereIsNothingToSell(t *testing.T) {
	r := newSettingsRig(t, func(a *App) { a.Sync = &fakeSync{snap: syncedSync()} })
	r.fm.Info = freeInfo
	mustNil(t, r.fm.UseOfficial())
	r.s.ShowUpgrade()
	r.tt.Frame()
	hasNot(t, r.tt, "Upgrade to Withdo Pro")
}

func TestUsedUpTheQuietLineOffersProToAFreeAccount(t *testing.T) {
	r := newAssistRig(t, &data.FakeAssistant{Ready: false})
	fm := r.app.Models.(*data.FakeModels)
	fm.Info, fm.Bill = data.OfficialInfo{Phase: data.OfficialUsedUp, Left: 0, Limit: 20}, freeBill
	r.tt.Key(ui.Cmd, ui.KeyK)
	has(t, r.tt, "Today's 20 turns are used up: dates and reminders are read on this computer.", "Upgrade to Pro", " for 100 a day.")
	link, _ := r.tt.Find("Upgrade to Pro")
	lx, ly := center(link)
	r.tt.Move(lx, ly)
	mustNil(t, r.tt.Click("Upgrade to Pro"))
	if r.rec.upgrades != 1 {
		t.Fatalf("OpenUpgrade asked %d times", r.rec.upgrades)
	}
}

func TestUsedUpTheQuietLineHasNoProLinkForProOrWhereNothingIsSold(t *testing.T) {
	for name, bill := range map[string]data.BillingInfo{
		"pro":      {Offered: true, Known: true, Pro: true, Status: "active"},
		"no sale":  {},
		"unknown":  {Offered: true},
		"free, ok": {Offered: true, Known: true},
	} {
		r := newAssistRig(t, &data.FakeAssistant{Ready: false})
		fm := r.app.Models.(*data.FakeModels)
		fm.Info, fm.Bill = data.OfficialInfo{Phase: data.OfficialUsedUp, Left: 0, Limit: 100}, bill
		r.tt.Key(ui.Cmd, ui.KeyK)
		want := name == "free, ok"
		if got := r.tt.HasText("Upgrade to Pro"); got != want {
			t.Errorf("%s: Upgrade to Pro shown = %v, want %v", name, got, want)
		}
	}
}

func TestComingBackToTheWindowAsksForThePlanAgain(t *testing.T) {
	r := proRig(t, freeInfo, freeBill)
	before := r.fm.BillingRefreshes
	r.s.Focused()
	if r.fm.BillingRefreshes != before+1 {
		t.Fatalf("refreshes %d -> %d", before, r.fm.BillingRefreshes)
	}
}

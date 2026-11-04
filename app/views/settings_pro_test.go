package views

import (
	"errors"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

// Withdo Pro on the desktop (M16, v45): the plan on the official card, the upgrade sheet and its
// waiting state, and the composer line that offers Pro.

var (
	freeInfo = data.OfficialInfo{Phase: data.OfficialReady, Left: 14, Limit: 20, ResetsAt: fixedNow().Add(14 * time.Hour)}
	freeBill = data.BillingInfo{Offered: true, Known: true, Monthly: "$6.99", Yearly: "$58.72"}
	proInfo  = data.OfficialInfo{Phase: data.OfficialReady, Left: 86, Limit: 100, ResetsAt: fixedNow().Add(14 * time.Hour)}
)

// proRig is Settings > Model with the official model in use and billing as set.
func proRig(t *testing.T, info data.OfficialInfo, bill data.BillingInfo) *settingsRig {
	t.Helper()
	r := newSettingsRig(t)
	r.fm.Info, r.fm.Bill = info, bill
	mustNil(t, r.fm.UseOfficial())
	r.assistant(t)
	return r
}

func TestTheFreeCardShowsThePlanTheQuotaAgainstItsLimitAndTheWayToPro(t *testing.T) {
	r := proRig(t, freeInfo, freeBill)
	has(t, r.tt, "Free", "Available", "· 14 / 20 left today", "Pro: 100 a day, on every device. Try it free for 7 days.", "Upgrade to Pro")
	hasNot(t, r.tt, "Included with your account", "Manage subscription")

	used := freeBill
	used.TrialUsed = true
	r.fm.Bill = used
	r.tt.Frame()
	has(t, r.tt, "Pro: 100 a day, on every device.", "Upgrade to Pro")
	hasNot(t, r.tt, "Pro: 100 a day, on every device. Try it free for 7 days.")
}

func TestWithoutBillingTheCardIsAsInV44(t *testing.T) {
	r := proRig(t, proInfo, data.BillingInfo{})
	has(t, r.tt, "Included with your account", "· 86 left today")
	hasNot(t, r.tt, "Free", "Upgrade to Pro", "Manage subscription")

	// Offered, but the server has not said more: the plan pill is there, the footer waits.
	r.fm.Bill = data.BillingInfo{Offered: true}
	r.tt.Frame()
	has(t, r.tt, "Free")
	hasNot(t, r.tt, "Upgrade to Pro", "Included with your account")
}

func TestTheProCardShowsTheTrialTheRenewalOrTheEnd(t *testing.T) {
	pro := data.BillingInfo{
		Offered: true, Known: true, Pro: true, Status: "trialing", PeriodEnd: fixedNow().Add(7 * 24 * time.Hour), TrialUsed: true,
	}
	r := proRig(t, proInfo, pro)
	has(t, r.tt, "Pro", "· 86 / 100 left today", "Withdo Pro · trial until Oct 21", "Manage subscription")
	hasNot(t, r.tt, "Upgrade to Pro", "Free")

	pro.Status, pro.PeriodEnd = "active", time.Date(2027, 10, 21, 0, 0, 0, 0, time.Local)
	r.fm.Bill = pro
	r.tt.Frame()
	has(t, r.tt, "Withdo Pro · renews Oct 21, 2027")

	pro.Status, pro.PeriodEnd = "canceling", fixedNow().Add(20*24*time.Hour)
	r.fm.Bill = pro
	r.tt.Frame()
	has(t, r.tt, "Withdo Pro · ends Nov 3")
}

func TestManageSubscriptionOpensAFreshPortalLinkEachTime(t *testing.T) {
	pro := data.BillingInfo{Offered: true, Known: true, Pro: true, Status: "active", PeriodEnd: fixedNow().Add(30 * 24 * time.Hour)}
	r := proRig(t, proInfo, pro)
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
	r := proRig(t, freeInfo, freeBill)
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
	r := proRig(t, freeInfo, bill)
	r.s.openUpgrade()
	r.tt.Frame()
	has(t, r.tt, "Subscribe", "Sold by Waffo, tax included.")
	hasNot(t, r.tt, "Start 7-day free trial", "Nothing is charged before the trial ends. Sold by Waffo, tax included.")
}

func TestStartingTheTrialOpensTheBrowserAndTheSheetWaitsUntilProArrives(t *testing.T) {
	r := proRig(t, freeInfo, freeBill)
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
	r := proRig(t, freeInfo, freeBill)
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
	r := proRig(t, freeInfo, freeBill)
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
		r := proRig(t, freeInfo, freeBill)
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
	r := proRig(t, freeInfo, freeBill)
	r.fm.CheckoutFn = func(string) (string, error) { return "", data.ErrAlreadySubscribed }
	r.s.openUpgrade()
	r.tt.Frame()
	mustNil(t, r.tt.Click("Start 7-day free trial"))
	r.rec.drain(t)
	r.tt.Frame()
	hasNot(t, r.tt, "Upgrade to Withdo Pro", "Finish paying in your browser")
}

func TestEscapeClosesTheSheet(t *testing.T) {
	r := proRig(t, freeInfo, freeBill)
	r.s.openUpgrade()
	r.tt.Frame()
	r.tt.Key(0, ui.KeyEscape)
	r.tt.Frame()
	hasNot(t, r.tt, "Upgrade to Withdo Pro")
}

func TestShowUpgradeOpensModelWithTheSheet(t *testing.T) {
	r := newSettingsRig(t)
	r.fm.Info, r.fm.Bill = freeInfo, freeBill
	mustNil(t, r.fm.UseOfficial())
	r.s.ShowUpgrade()
	r.tt.Frame()
	if r.s.pane != PaneModel {
		t.Fatalf("pane %v", r.s.pane)
	}
	has(t, r.tt, "Upgrade to Withdo Pro")
}

func TestShowUpgradeDoesNothingWhereThereIsNothingToSell(t *testing.T) {
	r := newSettingsRig(t)
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

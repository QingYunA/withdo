package views

import (
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/app/data"
)

// PayPal (M17, v46): a second button under the main one, only when the server takes PayPal.

func TestTheSheetShowsPayPalOnlyWhenTheServerTakesIt(t *testing.T) {
	r := proRig(t, freeInfo, freeBill)
	r.s.openUpgrade()
	r.tt.Frame()
	hasNot(t, r.tt, "Subscribe with PayPal", "Cards are sold by Waffo, tax included.")
	has(t, r.tt, "Nothing is charged before the trial ends. Sold by Waffo, tax included.")

	bill := freeBill
	bill.PayPal = true
	r.fm.Bill = bill
	r.tt.Frame()
	has(t, r.tt, "Start 7-day free trial", "or", "Subscribe with PayPal",
		"Nothing is charged before the trial ends. Cards are sold by Waffo, tax included.")
	hasNot(t, r.tt, "Nothing is charged before the trial ends. Sold by Waffo, tax included.")

	bill.TrialUsed = true
	r.fm.Bill = bill
	r.tt.Frame()
	has(t, r.tt, "Subscribe with PayPal", "Cards are sold by Waffo, tax included.")
	hasNot(t, r.tt, "Nothing is charged before the trial ends.")
}

func TestThePayPalButtonChecksOutTheChosenPlanAtPayPalAndTheSheetWaitsForPayPal(t *testing.T) {
	bill := freeBill
	bill.PayPal = true
	r := proRig(t, freeInfo, bill)
	var plan string
	r.fm.CheckoutFn = func(p string) (string, error) { plan = p; return "https://paypal.example.com/c/" + p, nil }
	r.s.openUpgrade()
	r.tt.Frame()
	mustNil(t, r.tt.Click("Monthly"))
	r.tt.Frame()
	mustNil(t, r.tt.Click("Subscribe with PayPal"))
	r.rec.drain(t)
	r.tt.Frame()
	if plan != data.PlanMonthly || r.fm.CheckoutProvider != data.ProviderPayPal {
		t.Fatalf("asked for %q at %q", plan, r.fm.CheckoutProvider)
	}
	if len(r.rec.urls) != 1 || r.rec.urls[0] != "https://paypal.example.com/c/monthly" {
		t.Fatalf("opened %v", r.rec.urls)
	}
	has(t, r.tt, "Finish subscribing in PayPal", "Reopen PayPal", "Cancel")
	hasNot(t, r.tt, "Finish paying in your browser", "Reopen the payment page")
	mustNil(t, r.tt.Click("Reopen PayPal"))
	if len(r.rec.urls) != 2 || r.rec.urls[1] != r.rec.urls[0] {
		t.Fatalf("Reopen opened %v", r.rec.urls)
	}
}

func TestTheMainButtonStillChecksOutAtWaffoWhenPayPalIsOffered(t *testing.T) {
	bill := freeBill
	bill.PayPal = true
	r := proRig(t, freeInfo, bill)
	r.s.openUpgrade()
	r.tt.Frame()
	mustNil(t, r.tt.Click("Start 7-day free trial"))
	r.rec.drain(t)
	r.tt.Frame()
	if r.fm.CheckoutProvider != data.ProviderWaffo {
		t.Fatalf("the main button asked at %q", r.fm.CheckoutProvider)
	}
	has(t, r.tt, "Finish paying in your browser")
}

func TestAPayPalSubscriptionsProLineSaysSo(t *testing.T) {
	pro := data.BillingInfo{
		Offered: true, Known: true, Pro: true, Status: "trialing", Interval: "yearly", Provider: data.ProviderPayPal,
		PeriodEnd: fixedNow().Add(7 * 24 * time.Hour), TrialUsed: true,
	}
	r := proRig(t, proInfo, pro)
	has(t, r.tt, "Withdo Pro · Yearly · via PayPal · trial until Oct 21, then renews", "Manage subscription")

	pro.Status, pro.PeriodEnd = "active", time.Date(2027, 10, 21, 0, 0, 0, 0, time.Local)
	r.fm.Bill = pro
	r.tt.Frame()
	has(t, r.tt, "Withdo Pro · Yearly · via PayPal · renews Oct 21, 2027")

	pro.Status, pro.PeriodEnd = "canceling", fixedNow().Add(20*24*time.Hour)
	r.fm.Bill = pro
	r.tt.Frame()
	has(t, r.tt, "Withdo Pro · Yearly · via PayPal · ends Nov 3")

	pro.Interval = ""
	r.fm.Bill = pro
	r.tt.Frame()
	has(t, r.tt, "Withdo Pro · via PayPal · ends Nov 3")

	pro.Provider = data.ProviderWaffo
	r.fm.Bill = pro
	r.tt.Frame()
	has(t, r.tt, "Withdo Pro · ends Nov 3")
	hasNot(t, r.tt, "via PayPal")
}

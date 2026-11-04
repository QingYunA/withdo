package billing

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestPayPalTokenIsCachedUntilItExpires(t *testing.T) {
	p := newPayPalService(t, PayPalSandbox, 7)
	for i := 0; i < 3; i++ {
		if _, err := p.PayPalCheckout(context.Background(), 7, "", PlanMonthly); err != nil {
			t.Fatal(err)
		}
		p.advance(time.Minute) // the rate limit counts per hour, this stays well inside it
	}
	if p.paypal.tokens != 1 {
		t.Fatalf("%d token requests for three calls", p.paypal.tokens)
	}
	p.advance(9 * time.Hour) // past expires_in
	if _, err := p.PayPalCheckout(context.Background(), 7, "", PlanMonthly); err != nil {
		t.Fatal(err)
	}
	if p.paypal.tokens != 2 {
		t.Fatalf("%d token requests after the token expired", p.paypal.tokens)
	}
}

func TestPayPalRevokedTokenIsRenewedOnce(t *testing.T) {
	p := newPayPalService(t, PayPalSandbox, 7)
	if _, err := p.PayPalCheckout(context.Background(), 7, "", PlanMonthly); err != nil {
		t.Fatal(err)
	}
	p.paypal.mu.Lock()
	p.paypal.revoke = true
	p.paypal.mu.Unlock()
	if _, err := p.PayPalCheckout(context.Background(), 7, "", PlanMonthly); err != nil {
		t.Fatalf("a revoked token was not renewed: %v", err)
	}
	if p.paypal.tokens != 2 {
		t.Fatalf("%d token requests", p.paypal.tokens)
	}
}

func TestPayPalBadCredentialsAreUpstreamErrors(t *testing.T) {
	p := newPayPalService(t, PayPalSandbox, 7)
	p.Service.paypal.cfg.Secret = "wrong"
	_, err := p.PayPalCheckout(context.Background(), 7, "", PlanMonthly)
	if !errors.Is(err, ErrUpstream) {
		t.Fatalf("err = %v", err)
	}
	if strings.Contains(err.Error(), "wrong") || strings.Contains(err.Error(), ppClient) {
		t.Fatalf("the error carries credentials: %v", err)
	}
}

func TestPayPalCheckoutCreatesASubscription(t *testing.T) {
	p := newPayPalService(t, PayPalSandbox, 7)
	url, err := p.PayPalCheckout(context.Background(), 7, "buyer@example.com", PlanYearly)
	if err != nil || url != "https://paypal.test/webapps/billing/subscriptions?ba_token=BA-1" {
		t.Fatalf("PayPalCheckout = %q, %v (the approve link is not the first)", url, err)
	}
	if len(p.paypal.created) != 1 {
		t.Fatalf("%d subscriptions created", len(p.paypal.created))
	}
	body := p.paypal.created[0]
	if body["plan_id"] != ppYearlyTrial || body["custom_id"] != "7" {
		t.Fatalf("body = %v", body)
	}
	if sub, _ := body["subscriber"].(map[string]any); sub["email_address"] != "buyer@example.com" {
		t.Fatalf("subscriber = %v", body["subscriber"])
	}
	ctx, _ := body["application_context"].(map[string]any)
	want := map[string]string{
		"brand_name": "Withdo", "shipping_preference": "NO_SHIPPING", "user_action": "SUBSCRIBE_NOW",
		"return_url": "https://withdo.si/pro/", "cancel_url": "https://withdo.si/pro/",
	}
	for k, v := range want {
		if ctx[k] != v {
			t.Errorf("application_context.%s = %v, want %s", k, ctx[k], v)
		}
	}
	if len(p.waffo.calls()) != 0 {
		t.Fatal("Waffo was called")
	}
}

func TestPayPalCheckoutHasNoTrialOnceUsedOnEitherChannel(t *testing.T) {
	for _, provider := range []string{"", ProviderWaffo, ProviderPayPal} {
		p := newPayPalService(t, PayPalSandbox, 7)
		p.store.subs[7] = Subscription{
			Account: 7, Provider: provider, Status: StatusCanceled, TrialUsed: true, Mode: ModeTest,
			PeriodEnd: testNow.Add(-time.Hour),
		}
		if _, err := p.PayPalCheckout(context.Background(), 7, "", PlanMonthly); err != nil {
			t.Fatal(err)
		}
		body := p.paypal.created[0]
		if body["plan_id"] != ppMonthly {
			t.Errorf("provider %q: plan_id = %v, want the plan without a trial", provider, body["plan_id"])
		}
		if _, has := body["subscriber"]; has {
			t.Errorf("provider %q: an unknown email was sent", provider)
		}
	}
}

func TestPayPalCheckoutRefusals(t *testing.T) {
	ctx := context.Background()
	p := newPayPalService(t, PayPalSandbox, 7)
	if _, err := p.PayPalCheckout(ctx, 7, "", "weekly"); !errors.Is(err, ErrUnknownPlan) {
		t.Fatalf("weekly = %v", err)
	}
	p.store.subs[7] = Subscription{Account: 7, Provider: ProviderWaffo, Status: StatusActive, Mode: ModeTest, PeriodEnd: testNow.Add(time.Hour)}
	if _, err := p.PayPalCheckout(ctx, 7, "", PlanMonthly); !errors.Is(err, ErrAlreadySubscribed) {
		t.Fatalf("a Pro account = %v", err)
	}
	delete(p.store.subs, 7)
	p.paypal.links = `[{"href":"x","rel":"self"}]`
	if _, err := p.PayPalCheckout(ctx, 7, "", PlanMonthly); !errors.Is(err, ErrUpstream) {
		t.Fatalf("no approve link = %v", err)
	}
	for i := 0; i < checkoutsPerHour; i++ {
		_, _ = p.PayPalCheckout(ctx, 7, "", PlanMonthly)
	}
	if _, err := p.PayPalCheckout(ctx, 7, "", PlanMonthly); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("too many = %v", err)
	}

	// PayPal off: a Service without it.
	off := newTestService(t, 7)
	if off.PayPalEnabled() {
		t.Fatal("PayPal is on without a config")
	}
	if _, err := off.PayPalCheckout(ctx, 7, "", PlanMonthly); !errors.Is(err, ErrPayPalOff) {
		t.Fatalf("PayPal off = %v", err)
	}
	if _, _, err := off.HandlePayPalWebhook(ctx, ppHeaders, ppEvent("e", "BILLING.SUBSCRIPTION.ACTIVATED", `{"id":"I-1"}`)); !errors.Is(err, ErrPayPalOff) {
		t.Fatalf("webhook with PayPal off = %v", err)
	}
}

func TestPayPalConfigIsChecked(t *testing.T) {
	good := PayPalConfig{
		ClientID: "c", Secret: "s", WebhookID: "w",
		PlanMonthly: "a", PlanMonthlyTrial: "b", PlanYearly: "c", PlanYearlyTrial: "d",
	}
	if _, err := newPayPalAPI(good, nil, time.Now); err != nil {
		t.Fatal(err)
	}
	api, _ := newPayPalAPI(good, nil, time.Now)
	if api.cfg.Env != PayPalSandbox || api.cfg.BaseURL != "https://api-m.sandbox.paypal.com" {
		t.Fatalf("defaults = %+v", api.cfg)
	}
	live := good
	live.Env = "LIVE"
	if api, _ := newPayPalAPI(live, nil, time.Now); api.cfg.BaseURL != "https://api-m.paypal.com" {
		t.Fatalf("live base = %s", api.cfg.BaseURL)
	}
	for name, mutate := range map[string]func(*PayPalConfig){
		"env": func(c *PayPalConfig) { c.Env = "prod" }, "client": func(c *PayPalConfig) { c.ClientID = "" },
		"secret": func(c *PayPalConfig) { c.Secret = "" }, "webhook": func(c *PayPalConfig) { c.WebhookID = "" },
		"plan": func(c *PayPalConfig) { c.PlanYearlyTrial = " " },
	} {
		bad := good
		mutate(&bad)
		if _, err := newPayPalAPI(bad, nil, time.Now); err == nil {
			t.Errorf("%s: a bad config was accepted", name)
		}
	}
}

func TestPayPalWebhookVerification(t *testing.T) {
	ctx := context.Background()
	p := newPayPalService(t, PayPalSandbox, 7)
	p.paypal.setSub("I-1", ppTrialing("I-1", "7", testNow).json())

	// The event is sent to PayPal byte for byte, with the webhook id and the five headers.
	event := []byte(`{"id":"WH-A", "event_type":"BILLING.SUBSCRIPTION.ACTIVATED","resource":{"id":"I-1","note":"a<b & c"}}`)
	ev, outcome, err := p.HandlePayPalWebhook(ctx, ppHeaders, event)
	if err != nil || outcome != OutcomeApplied || ev.Account != 7 {
		t.Fatalf("genuine: %+v %s %v", ev, outcome, err)
	}
	if p.paypal.verifyCount() != 1 {
		t.Fatalf("%d verifications", p.paypal.verifyCount())
	}
	v := p.paypal.verifications[0]
	for field, want := range map[string]string{
		"auth_algo": ppHeaders.AuthAlgo, "cert_url": ppHeaders.CertURL, "transmission_id": ppHeaders.TransmissionID,
		"transmission_sig": ppHeaders.TransmissionSig, "transmission_time": ppHeaders.TransmissionTime, "webhook_id": ppWebhook,
	} {
		if string(v[field]) != `"`+want+`"` {
			t.Errorf("%s = %s", field, v[field])
		}
	}
	if !strings.Contains(string(v["_raw"]), `"webhook_event":`+string(event)+`}`) {
		t.Fatalf("the event was not sent as received: %s", v["_raw"])
	}

	// PayPal says no: nothing is read or stored.
	p.paypal.verifyResult = "FAILURE"
	before := p.paypal.gets("I-1")
	if _, _, err := p.HandlePayPalWebhook(ctx, ppHeaders, ppEvent("WH-B", "BILLING.SUBSCRIPTION.CANCELLED", `{"id":"I-1"}`)); !errors.Is(err, ErrBadSignature) {
		t.Fatalf("FAILURE = %v", err)
	}
	if p.paypal.gets("I-1") != before {
		t.Fatal("the subscription was read for a delivery PayPal did not vouch for")
	}
	// A 400 from the verify call (a certificate URL it does not accept) is a no as well.
	p.paypal.verifyResult, p.paypal.verifyHTTP = "SUCCESS", 400
	if _, _, err := p.HandlePayPalWebhook(ctx, ppHeaders, ppEvent("WH-C", "BILLING.SUBSCRIPTION.CANCELLED", `{"id":"I-1"}`)); !errors.Is(err, ErrBadSignature) {
		t.Fatalf("400 = %v", err)
	}
	// PayPal being down is not a no: the delivery is retried.
	p.paypal.verifyHTTP = 500
	if _, _, err := p.HandlePayPalWebhook(ctx, ppHeaders, ppEvent("WH-D", "BILLING.SUBSCRIPTION.CANCELLED", `{"id":"I-1"}`)); err == nil || errors.Is(err, ErrBadSignature) {
		t.Fatalf("500 = %v", err)
	}
	// Missing headers and a body that is not JSON never reach PayPal.
	p.paypal.verifyHTTP = 200
	n := p.paypal.verifyCount()
	for _, h := range []PayPalHeaders{{}, {AuthAlgo: "a"}, {CertURL: "c", TransmissionID: "t", TransmissionSig: "s", TransmissionTime: "x"}} {
		if _, _, err := p.HandlePayPalWebhook(ctx, h, event); !errors.Is(err, ErrBadSignature) {
			t.Errorf("headers %+v = %v", h, err)
		}
	}
	if _, _, err := p.HandlePayPalWebhook(ctx, ppHeaders, []byte("not json")); !errors.Is(err, ErrBadSignature) {
		t.Errorf("junk = %v", err)
	}
	if p.paypal.verifyCount() != n {
		t.Fatal("an incomplete delivery was sent to PayPal")
	}
	if sub := p.store.subs[7]; sub.Status != StatusTrialing {
		t.Fatalf("a refused delivery changed the row: %+v", sub)
	}
}

// The states of a subscription, read from PayPal and never from the event.
func TestPayPalWebhookMapsTheSubscriptionState(t *testing.T) {
	period := testNow.AddDate(0, 1, 0)
	cases := []struct {
		name       string
		sub        ppSubOpts
		wantStatus string
		wantPro    bool
		wantEnd    time.Time
	}{
		{"trial", ppTrialing("I-1", "7", testNow), StatusTrialing, true, testNow.Add(7 * 24 * time.Hour)},
		{"trial counted done but not yet charged", ppSubOpts{
			id: "I-1", status: "ACTIVE", plan: ppMonthlyTrial, customID: "7", updated: testNow, nextBilling: testNow.Add(time.Hour),
			cycles: []ppCycle{{trial: true, completed: 1, total: 1}, {completed: 0, total: 0}},
		}, StatusTrialing, true, testNow.Add(time.Hour)},
		{"active after the first charge", ppPaid("I-1", "7", testNow), StatusActive, true, period},
		{"active without a trial plan", ppSubOpts{
			id: "I-1", status: "ACTIVE", plan: ppMonthly, customID: "7", updated: testNow, nextBilling: period,
			lastPayment: testNow, lastAmount: "6.99", cycles: []ppCycle{{completed: 1, total: 0}},
		}, StatusActive, true, period},
		{"a failed first charge is not a trial", ppSubOpts{
			id: "I-1", status: "ACTIVE", plan: ppMonthlyTrial, customID: "7", updated: testNow, nextBilling: testNow.Add(time.Hour), failed: 1,
			cycles: []ppCycle{{trial: true, completed: 1, total: 1}, {completed: 0, total: 0}},
		}, StatusActive, true, testNow.Add(time.Hour)},
		{"cancelled with the next billing time kept", ppSubOpts{
			id: "I-1", status: "CANCELLED", plan: ppMonthly, customID: "7", updated: testNow, nextBilling: period,
			lastPayment: testNow, lastAmount: "6.99", cycles: []ppCycle{{completed: 1, total: 0}},
		}, StatusCanceling, true, period},
		{"cancelled, paid through the last payment", ppSubOpts{
			id: "I-1", status: "CANCELLED", plan: ppMonthly, customID: "7", updated: testNow.Add(24 * time.Hour),
			lastPayment: testNow, lastAmount: "6.99", cycles: []ppCycle{{completed: 1, total: 0}},
		}, StatusCanceling, true, period},
		{"cancelled during the trial with nothing paid", ppSubOpts{
			id: "I-1", status: "CANCELLED", plan: ppMonthlyTrial, customID: "7", updated: testNow,
			cycles: []ppCycle{{trial: true, completed: 0, total: 1, rem: 1}, {total: 0}},
		}, StatusCanceled, false, testNow},
		{"cancelled after its period", ppSubOpts{
			id: "I-1", status: "CANCELLED", plan: ppMonthly, customID: "7", updated: testNow, nextBilling: testNow.Add(-time.Hour),
		}, StatusCanceled, false, testNow.Add(-time.Hour)},
		{"suspended", ppSubOpts{
			id: "I-1", status: "SUSPENDED", plan: ppMonthly, customID: "7", updated: testNow, nextBilling: testNow.Add(-time.Hour), failed: 1,
		}, StatusPastDue, true, testNow.Add(-time.Hour)},
		{"suspended long ago", ppSubOpts{
			id: "I-1", status: "SUSPENDED", plan: ppMonthly, customID: "7", updated: testNow, nextBilling: testNow.Add(-3 * 24 * time.Hour),
		}, StatusPastDue, false, testNow.Add(-3 * 24 * time.Hour)},
		{"expired", ppSubOpts{id: "I-1", status: "EXPIRED", plan: ppMonthly, customID: "7", updated: testNow}, StatusCanceled, false, testNow},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := newPayPalService(t, PayPalSandbox, 7)
			p.paypal.setSub("I-1", tc.sub.json())
			// The event says the opposite of what PayPal reports.
			ev, outcome, err := p.deliver("WH-1", "BILLING.SUBSCRIPTION.CANCELLED", "I-1")
			if err != nil || outcome != OutcomeApplied || ev.Account != 7 {
				t.Fatalf("deliver: %+v %s %v", ev, outcome, err)
			}
			sub := p.store.subs[7]
			if sub.Status != tc.wantStatus || sub.Provider != ProviderPayPal || sub.OrderID != "I-1" || sub.Mode != ModeTest {
				t.Fatalf("row = %+v", sub)
			}
			if sub.Pro(testNow) != tc.wantPro {
				t.Errorf("Pro = %v, want %v", sub.Pro(testNow), tc.wantPro)
			}
			if !sub.PeriodEnd.Equal(tc.wantEnd) {
				t.Errorf("PeriodEnd = %v, want %v", sub.PeriodEnd, tc.wantEnd)
			}
			if sub.Interval != "monthly" {
				t.Errorf("Interval = %q", sub.Interval)
			}
			if got := p.notifications(); len(got) != 1 || got[0] != 7 {
				t.Errorf("notified %v", got)
			}
		})
	}
}

func TestPayPalCancelDuringTheTrialKeepsProUntilItsEnd(t *testing.T) {
	p := newPayPalService(t, PayPalSandbox, 7)
	p.paypal.setSub("I-1", ppTrialing("I-1", "7", testNow).json())
	if _, _, err := p.deliver("WH-1", "BILLING.SUBSCRIPTION.ACTIVATED", "I-1"); err != nil {
		t.Fatal(err)
	}
	if sub := p.store.subs[7]; sub.Status != StatusTrialing || !sub.TrialUsed {
		t.Fatalf("after activation = %+v", sub)
	}
	// PayPal drops next_billing_time on a cancellation: the trial's end the row knew stays.
	cancelled := ppSubOpts{
		id: "I-1", status: "CANCELLED", plan: ppMonthlyTrial, customID: "7", updated: testNow.Add(time.Hour),
		cycles: []ppCycle{{trial: true, completed: 0, total: 1, rem: 1}, {total: 0}},
	}
	p.paypal.setSub("I-1", cancelled.json())
	if _, outcome, err := p.deliver("WH-2", "BILLING.SUBSCRIPTION.CANCELLED", "I-1"); err != nil || outcome != OutcomeApplied {
		t.Fatalf("cancel: %s %v", outcome, err)
	}
	sub := p.store.subs[7]
	if sub.Status != StatusCanceling || !sub.Pro(testNow.Add(24*time.Hour)) || sub.Pro(testNow.Add(8*24*time.Hour)) || !sub.TrialUsed {
		t.Fatalf("after cancel = %+v", sub)
	}
}

func TestPayPalWebhookDedupesAndIgnoresOutOfOrderReads(t *testing.T) {
	p := newPayPalService(t, PayPalSandbox, 7)
	paid := ppPaid("I-1", "7", testNow.Add(time.Hour))
	p.paypal.setSub("I-1", paid.json())
	if _, outcome, err := p.deliver("WH-1", "PAYMENT.SALE.COMPLETED", "I-1"); err != nil || outcome != OutcomeApplied {
		t.Fatalf("first: %s %v", outcome, err)
	}
	// The same event again (PayPal retries).
	if _, outcome, err := p.deliver("WH-1", "PAYMENT.SALE.COMPLETED", "I-1"); err != nil || outcome != OutcomeDuplicate {
		t.Fatalf("repeat: %s %v", outcome, err)
	}
	if got := p.notifications(); len(got) != 1 {
		t.Fatalf("notified %v", got)
	}
	// An older delivery processed late: its read of the subscription is older than the row.
	stale := ppTrialing("I-1", "7", testNow)
	p.paypal.setSub("I-1", stale.json())
	if _, outcome, err := p.deliver("WH-0", "BILLING.SUBSCRIPTION.ACTIVATED", "I-1"); err != nil || outcome != OutcomeIgnored {
		t.Fatalf("stale read: %s %v", outcome, err)
	}
	if sub := p.store.subs[7]; sub.Status != StatusActive {
		t.Fatalf("a stale read undid the row: %+v", sub)
	}
	// The same state again under a new event id changes nothing and says nothing.
	p.paypal.setSub("I-1", paid.json())
	if _, outcome, _ := p.deliver("WH-2", "BILLING.SUBSCRIPTION.UPDATED", "I-1"); outcome != OutcomeApplied {
		t.Fatalf("a new event: %s", outcome)
	}
	if got := p.notifications(); len(got) != 1 {
		t.Fatalf("an unchanged state notified %v", got)
	}
}

func TestPayPalWebhookIgnoresWhatIsNotOurs(t *testing.T) {
	ctx := context.Background()
	p := newPayPalService(t, PayPalSandbox, 7)
	p.paypal.setSub("I-nocustom", ppTrialing("I-nocustom", "", testNow).json())
	p.paypal.setSub("I-badcustom", ppTrialing("I-badcustom", "not-a-number", testNow).json())
	p.paypal.setSub("I-nobody", ppTrialing("I-nobody", "999", testNow).json())
	p.paypal.setSub("I-pending", ppSubOpts{id: "I-pending", status: "APPROVAL_PENDING", plan: ppMonthly, customID: "7", updated: testNow}.json())
	p.paypal.setSub("I-other", ppTrialing("I-mismatch", "7", testNow).json())
	for _, subID := range []string{"I-nocustom", "I-badcustom", "I-nobody", "I-pending", "I-unknown", "I-other"} {
		if _, outcome, err := p.deliver("WH-"+subID, "BILLING.SUBSCRIPTION.ACTIVATED", subID); err != nil || outcome != OutcomeIgnored {
			t.Errorf("%s: %s %v", subID, outcome, err)
		}
	}
	if len(p.store.subs) != 0 {
		t.Fatalf("rows = %+v", p.store.subs)
	}
	// An event type the webhook is not registered for, and a sale without a subscription, read nothing.
	before := p.paypal.gets("I-1")
	for _, body := range [][]byte{
		ppEvent("WH-x", "CUSTOMER.DISPUTE.CREATED", `{"id":"I-1"}`),
		ppEvent("WH-y", "PAYMENT.SALE.COMPLETED", `{"id":"SALE-1"}`),
		ppEvent("WH-z", "PAYMENT.SALE.REFUNDED", `{"id":"REF-1","sale_id":"SALE-1"}`),
	} {
		if _, outcome, err := p.HandlePayPalWebhook(ctx, ppHeaders, body); err != nil || outcome != OutcomeIgnored {
			t.Errorf("%s: %s %v", body, outcome, err)
		}
	}
	if p.paypal.gets("I-1") != before || p.paypal.gets("SALE-1") != 0 {
		t.Fatal("an event that names no subscription caused a read")
	}
	// PayPal failing to answer is retried, not acknowledged.
	p.paypal.showStatus = 500
	if _, _, err := p.deliver("WH-fail", "BILLING.SUBSCRIPTION.ACTIVATED", "I-1"); err == nil || !errors.Is(err, ErrUpstream) {
		t.Fatalf("PayPal 500 = %v", err)
	}
}

func TestPayPalSaleEventsFindTheirSubscription(t *testing.T) {
	p := newPayPalService(t, PayPalSandbox, 7)
	p.paypal.setSub("I-1", ppPaid("I-1", "7", testNow).json())
	body := ppEvent("WH-1", "PAYMENT.SALE.COMPLETED", `{"id":"SALE-1","billing_agreement_id":"I-1","state":"completed"}`)
	ev, outcome, err := p.HandlePayPalWebhook(context.Background(), ppHeaders, body)
	if err != nil || outcome != OutcomeApplied || ev.SubscriptionID != "I-1" || p.store.subs[7].Status != StatusActive {
		t.Fatalf("%+v %s %v %+v", ev, outcome, err, p.store.subs[7])
	}
}

func TestPayPalTakesTheRowOverAndOnlyItsOwnEventsFollow(t *testing.T) {
	p := newPayPalService(t, PayPalSandbox, 7)
	// The account was on Waffo and that ended.
	p.store.subs[7] = Subscription{
		Account: 7, Provider: ProviderWaffo, OrderID: "ORD_1", Status: StatusCanceled, TrialUsed: true, Mode: ModeTest,
		Interval: "yearly", PeriodEnd: testNow.Add(-24 * time.Hour), EventAt: testNow.Add(-24 * time.Hour),
	}
	p.paypal.setSub("I-2", ppSubOpts{
		id: "I-2", status: "ACTIVE", plan: ppYearly, customID: "7", updated: testNow, nextBilling: testNow.AddDate(1, 0, 0),
		lastPayment: testNow, lastAmount: "58.72", cycles: []ppCycle{{completed: 1}},
	}.json())
	if _, outcome, err := p.deliver("WH-1", "BILLING.SUBSCRIPTION.ACTIVATED", "I-2"); err != nil || outcome != OutcomeApplied {
		t.Fatalf("takeover: %s %v", outcome, err)
	}
	sub := p.store.subs[7]
	if sub.Provider != ProviderPayPal || sub.OrderID != "I-2" || sub.Status != StatusActive || sub.Interval != "yearly" || !sub.TrialUsed {
		t.Fatalf("row = %+v", sub)
	}

	// A late Waffo event of the old order does not touch it.
	late := webhookBody("w9", "subscription.canceled", 7, testNow.Add(time.Minute), nil)
	if outcome, err := p.deliverWaffo(t, late, testNow); err != nil || outcome != OutcomeIgnored {
		t.Fatalf("old Waffo event: %s %v", outcome, err)
	}
	// So does the end of another PayPal subscription.
	p.paypal.setSub("I-old", ppSubOpts{id: "I-old", status: "CANCELLED", plan: ppMonthly, customID: "7", updated: testNow.Add(time.Hour)}.json())
	if _, outcome, err := p.deliver("WH-2", "BILLING.SUBSCRIPTION.CANCELLED", "I-old"); err != nil || outcome != OutcomeIgnored {
		t.Fatalf("old PayPal subscription: %s %v", outcome, err)
	}
	if got := p.store.subs[7]; got.Status != StatusActive || got.OrderID != "I-2" {
		t.Fatalf("row = %+v", got)
	}
	// The row's own cancellation applies.
	p.paypal.setSub("I-2", ppSubOpts{
		id: "I-2", status: "CANCELLED", plan: ppYearly, customID: "7", updated: testNow.Add(2 * time.Hour), nextBilling: testNow.AddDate(1, 0, 0),
	}.json())
	if _, outcome, err := p.deliver("WH-3", "BILLING.SUBSCRIPTION.CANCELLED", "I-2"); err != nil || outcome != OutcomeApplied {
		t.Fatalf("cancel: %s %v", outcome, err)
	}
	if got := p.store.subs[7]; got.Status != StatusCanceling || !got.Pro(testNow.AddDate(0, 6, 0)) {
		t.Fatalf("row = %+v", got)
	}

	// And a Waffo activation takes the row back.
	p.advance(3 * time.Hour)
	act := webhookBody("w10", "subscription.activated", 7, testNow.Add(3*time.Hour), map[string]any{
		"orderId": "ORD_2", "billingPeriod": "monthly",
		"currentPeriodStart": testNow.Add(3 * time.Hour).Format(time.RFC3339), "currentPeriodEnd": testNow.AddDate(0, 1, 0).Format(time.RFC3339),
	})
	if outcome, err := p.deliverWaffo(t, act, testNow.Add(3*time.Hour)); err != nil || outcome != OutcomeApplied {
		t.Fatalf("Waffo activation: %s %v", outcome, err)
	}
	if got := p.store.subs[7]; got.Provider != ProviderWaffo || got.OrderID != "ORD_2" || got.Status != StatusActive {
		t.Fatalf("row = %+v", got)
	}
}

func (p *ppService) deliverWaffo(t *testing.T, body []byte, signedAt time.Time) (Outcome, error) {
	t.Helper()
	return p.testService.deliver(t, body, signedAt)
}

func TestPayPalModeFollowsTheEnvironment(t *testing.T) {
	ctx := context.Background()
	sandbox := newPayPalService(t, PayPalSandbox, 7)
	sandbox.store.subs[7] = Subscription{Account: 7, Provider: ProviderPayPal, OrderID: "I-1", Status: StatusActive, Mode: ModeProd, TrialUsed: true,
		PeriodEnd: testNow.AddDate(0, 1, 0)}
	if _, ok, _ := sandbox.Subscription(ctx, 7); ok {
		t.Fatal("a live subscription counts on the sandbox")
	}
	if sandbox.IsProSubject(ctx, "subject-7") {
		t.Fatal("a live subscription gives Pro on the sandbox")
	}
	sandbox.store.subs[7] = Subscription{Account: 7, Provider: ProviderPayPal, OrderID: "I-1", Status: StatusActive, Mode: ModeTest,
		PeriodEnd: testNow.AddDate(0, 1, 0)}
	if !sandbox.IsProSubject(ctx, "subject-7") {
		t.Fatal("a sandbox subscription gives no Pro on the sandbox")
	}

	live := newPayPalService(t, PayPalLive, 7)
	live.store.subs[7] = Subscription{Account: 7, Provider: ProviderPayPal, OrderID: "I-1", Status: StatusActive, Mode: ModeTest, TrialUsed: true,
		PeriodEnd: testNow.AddDate(0, 1, 0)}
	if _, ok, _ := live.Subscription(ctx, 7); ok {
		t.Fatal("a sandbox subscription counts on live")
	}
	// ... so its trial is not used up either.
	if _, err := live.PayPalCheckout(ctx, 7, "", PlanMonthly); err != nil {
		t.Fatal(err)
	}
	if live.paypal.created[0]["plan_id"] != ppMonthlyTrial {
		t.Fatalf("plan = %v", live.paypal.created[0]["plan_id"])
	}
	live.paypal.setSub("I-9", ppTrialing("I-9", "7", testNow).json())
	if _, _, err := live.deliver("WH-1", "BILLING.SUBSCRIPTION.ACTIVATED", "I-9"); err != nil {
		t.Fatal(err)
	}
	if sub := live.store.subs[7]; sub.Mode != ModeProd {
		t.Fatalf("mode = %s", sub.Mode)
	}

	// With PayPal off a PayPal row is no subscription: the server cannot say which mode it is.
	off := newTestService(t, 7)
	off.store.subs[7] = Subscription{Account: 7, Provider: ProviderPayPal, OrderID: "I-1", Status: StatusActive, Mode: ModeTest, PeriodEnd: testNow.AddDate(0, 1, 0)}
	if off.IsProSubject(ctx, "subject-7") {
		t.Fatal("a PayPal subscription gives Pro with PayPal off")
	}
}

func TestPayPalProKeepsTheBillingLagButCancelingDoesNot(t *testing.T) {
	end := testNow
	paypal := Subscription{Provider: ProviderPayPal, Status: StatusActive, PeriodEnd: end}
	if !paypal.Pro(end.Add(23*time.Hour)) || paypal.Pro(end.Add(25*time.Hour)) {
		t.Error("an active PayPal subscription does not get exactly the billing lag")
	}
	paypal.Status = StatusCanceling
	if paypal.Pro(end.Add(time.Minute)) {
		t.Error("a cancelled PayPal subscription outlives its period")
	}
	waffo := Subscription{Provider: ProviderWaffo, Status: StatusActive, PeriodEnd: end}
	if waffo.Pro(end.Add(time.Minute)) {
		t.Error("the PayPal lag leaked to Waffo")
	}
	if !(Subscription{Status: StatusActive}).Pro(end) {
		t.Error("an active subscription with no period end is not Pro")
	}
}

func TestPayPalStatus(t *testing.T) {
	ctx := context.Background()
	p := newPayPalService(t, PayPalSandbox, 7, 8)
	p.store.subs[7] = Subscription{Account: 7, Provider: ProviderPayPal, OrderID: "I-1", Status: StatusActive, Mode: ModeTest,
		Interval: "monthly", PeriodEnd: testNow.AddDate(0, 1, 0)}
	st, err := p.Status(ctx, 7, func() string { return "a@example.com" })
	if err != nil || st.Plan != "pro" || st.Provider != ProviderPayPal || st.ManageURL != "https://www.sandbox.paypal.com/myaccount/autopay/" {
		t.Fatalf("PayPal status = %+v, %v", st, err)
	}
	if len(p.waffo.calls()) != 0 {
		t.Fatal("a portal link was made for a PayPal subscription")
	}
	live := newPayPalService(t, PayPalLive, 7)
	live.store.subs[7] = Subscription{Account: 7, Provider: ProviderPayPal, OrderID: "I-1", Status: StatusActive, Mode: ModeProd, PeriodEnd: testNow.AddDate(0, 1, 0)}
	if st, _ := live.Status(ctx, 7, func() string { return "" }); st.ManageURL != "https://www.paypal.com/myaccount/autopay/" {
		t.Fatalf("live manage url = %s", st.ManageURL)
	}

	// A cancelled subscription whose period has passed reads canceled: PayPal sends no event for it.
	p.store.subs[7] = Subscription{Account: 7, Provider: ProviderPayPal, OrderID: "I-1", Status: StatusCanceling, Mode: ModeTest,
		TrialUsed: true, PeriodEnd: testNow.Add(-time.Hour)}
	if st, _ := p.Status(ctx, 7, func() string { return "" }); st.Plan != "free" || st.Status != StatusCanceled || !st.TrialUsed {
		t.Fatalf("lapsed = %+v", st)
	}

	// Waffo's keep their portal link and say so.
	p.store.subs[8] = Subscription{Account: 8, OrderID: "ORD_1", Status: StatusActive, Mode: ModeTest, PeriodEnd: testNow.AddDate(0, 1, 0)}
	st, _ = p.Status(ctx, 8, func() string { return "b@example.com" })
	if st.Provider != ProviderWaffo || st.ManageURL != "https://portal.example/store#token=tok-1" {
		t.Fatalf("Waffo status = %+v", st)
	}
	if st, _ := p.Status(ctx, 8, func() string { return "" }); st.ManageURL != PortalLoginURL {
		t.Fatalf("no email: %+v", st)
	}
	// An account that never subscribed has no provider.
	p2 := newPayPalService(t, PayPalSandbox, 9)
	if st, _ := p2.Status(ctx, 9, func() string { return "" }); st.Provider != "" || st.Plan != "free" {
		t.Fatalf("never subscribed = %+v", st)
	}
}

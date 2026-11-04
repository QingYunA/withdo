package billing

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestCheckoutSendsASignedSessionRequest(t *testing.T) {
	ts := newTestService(t, 7)
	url, err := ts.Checkout(context.Background(), 7, "buyer@example.com", PlanYearly, "zh-CN")
	if err != nil || url != "https://checkout.example/c/1" {
		t.Fatalf("Checkout = %q, %v", url, err)
	}
	calls := ts.waffo.calls()
	if len(calls) != 1 {
		t.Fatalf("%d calls", len(calls))
	}
	c := calls[0]
	if c.Path != "/v1/actions/checkout/create-session" || c.Merchant != "MER_test" || !c.SignatureOK {
		t.Fatalf("call = %+v", c)
	}
	if c.Timestamp != "1793793600" { // testNow in Unix seconds
		t.Fatalf("timestamp = %s", c.Timestamp)
	}
	wantFields := map[string]any{
		"productId": "PROD_yearly", "currency": "USD", "buyerEmail": "buyer@example.com",
		"withTrial": true, "successUrl": "https://withdo.si/pro/", "language": "zh-Hans",
	}
	for k, want := range wantFields {
		if c.Body[k] != want {
			t.Errorf("%s = %v, want %v", k, c.Body[k], want)
		}
	}
	if meta, _ := c.Body["metadata"].(map[string]any); meta["account"] != "7" {
		t.Errorf("metadata = %v", c.Body["metadata"])
	}
	if id, _ := c.Body["orderMerchantExternalId"].(string); !strings.HasPrefix(id, "withdo-7-") || len(id) > 128 {
		t.Errorf("orderMerchantExternalId = %q", id)
	}
}

func TestCheckoutDropsTheTrialOnceUsed(t *testing.T) {
	ts := newTestService(t, 7)
	ts.store.subs[7] = Subscription{Account: 7, Status: StatusCanceled, TrialUsed: true, PeriodEnd: testNow.Add(-time.Hour)}
	if _, err := ts.Checkout(context.Background(), 7, "", PlanMonthly, ""); err != nil {
		t.Fatal(err)
	}
	body := ts.waffo.calls()[0].Body
	if body["withTrial"] != false {
		t.Fatalf("withTrial = %v", body["withTrial"])
	}
	if _, has := body["buyerEmail"]; has {
		t.Fatal("an unknown email was sent")
	}
	if _, has := body["language"]; has {
		t.Fatal("an empty language was sent")
	}
}

func TestCheckoutRefusals(t *testing.T) {
	ts := newTestService(t, 7)
	if _, err := ts.Checkout(context.Background(), 7, "", "weekly", ""); !errors.Is(err, ErrUnknownPlan) {
		t.Fatalf("plan weekly = %v", err)
	}
	ts.store.subs[7] = Subscription{Account: 7, Status: StatusActive, PeriodEnd: testNow.Add(24 * time.Hour)}
	if _, err := ts.Checkout(context.Background(), 7, "", PlanMonthly, ""); !errors.Is(err, ErrAlreadySubscribed) {
		t.Fatalf("a Pro account = %v", err)
	}
	if n := len(ts.waffo.calls()); n != 0 {
		t.Fatalf("%d calls to Waffo for refused checkouts", n)
	}
}

func TestCheckoutIsRateLimitedPerAccount(t *testing.T) {
	ts := newTestService(t, 7, 8)
	for i := 0; i < checkoutsPerHour; i++ {
		if _, err := ts.Checkout(context.Background(), 7, "", PlanMonthly, ""); err != nil {
			t.Fatalf("checkout %d: %v", i, err)
		}
	}
	if _, err := ts.Checkout(context.Background(), 7, "", PlanMonthly, ""); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("one too many = %v", err)
	}
	if _, err := ts.Checkout(context.Background(), 8, "", PlanMonthly, ""); err != nil {
		t.Fatalf("another account = %v", err)
	}
	ts.advance(61 * time.Minute)
	if _, err := ts.Checkout(context.Background(), 7, "", PlanMonthly, ""); err != nil {
		t.Fatalf("an hour later = %v", err)
	}
}

func TestCheckoutUpstreamFailures(t *testing.T) {
	ts := newTestService(t, 7)
	ts.waffo.reply = func(fakeCall) (int, string) {
		return 401, `{"data":null,"errors":[{"message":"Invalid signature","layer":"gateway"}]}`
	}
	_, err := ts.Checkout(context.Background(), 7, "secret@example.com", PlanMonthly, "")
	if !errors.Is(err, ErrUpstream) || !strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "secret@example.com") {
		t.Fatalf("err = %v", err)
	}
	ts.waffo.Close()
	if _, err := ts.Checkout(context.Background(), 7, "", PlanMonthly, ""); !errors.Is(err, ErrUpstream) {
		t.Fatalf("unreachable = %v", err)
	}
}

func TestCheckoutRetriesWithoutALanguageWaffoRefuses(t *testing.T) {
	ts := newTestService(t, 7)
	ts.waffo.reply = func(call fakeCall) (int, string) {
		if call.Body["language"] != nil {
			return 400, `{"data":null,"errors":[{"message":"Invalid language"}]}`
		}
		return 200, `{"data":{"checkoutUrl":"https://checkout.example/c/2"}}`
	}
	url, err := ts.Checkout(context.Background(), 7, "", PlanMonthly, "ja")
	if err != nil || url != "https://checkout.example/c/2" {
		t.Fatalf("Checkout = %q, %v", url, err)
	}
	if n := len(ts.waffo.calls()); n != 2 {
		t.Fatalf("%d calls", n)
	}
}

func TestStatusAndPortalLink(t *testing.T) {
	ts := newTestService(t, 7, 8)
	free, err := ts.Status(context.Background(), 8, func() string { t.Fatal("email asked for a free account"); return "" })
	if err != nil || free.Plan != "free" || free.ManageURL != "" || free.Status != "" {
		t.Fatalf("free = %+v, %v", free, err)
	}

	end := testNow.Add(30 * 24 * time.Hour)
	ts.store.subs[7] = Subscription{Account: 7, Status: StatusActive, PeriodEnd: end, TrialUsed: true}
	st, err := ts.Status(context.Background(), 7, func() string { return "buyer@example.com" })
	if err != nil || st.Plan != "pro" || st.Status != StatusActive || !st.PeriodEnd.Equal(end) || !st.TrialUsed {
		t.Fatalf("pro = %+v, %v", st, err)
	}
	if st.ManageURL != "https://portal.example/store#token=tok-1" {
		t.Fatalf("manage url = %q", st.ManageURL)
	}
	c := ts.waffo.calls()[0]
	if c.Path != "/v1/actions/auth/issue-session-token" || !c.SignatureOK ||
		c.Body["storeId"] != "STO_withdo" || c.Body["buyerIdentity"] != "buyer@example.com" || c.Body["purpose"] != "portal" {
		t.Fatalf("portal call = %+v", c)
	}

	// No email, or a portal that fails: the sign-in page.
	for name, email := range map[string]string{"no email": "", "failing portal": "buyer@example.com"} {
		if name == "failing portal" {
			ts.waffo.reply = func(fakeCall) (int, string) { return 500, `{}` }
		}
		got, err := ts.Status(context.Background(), 7, func() string { return email })
		if err != nil || got.ManageURL != PortalLoginURL {
			t.Fatalf("%s: manage url = %q, %v", name, got.ManageURL, err)
		}
	}

	// A subscription that ended is free again, with its status still reported.
	ts.advance(31 * 24 * time.Hour)
	got, _ := ts.Status(context.Background(), 7, func() string { return "" })
	if got.Plan != "free" || got.Status != StatusActive {
		t.Fatalf("expired = %+v", got)
	}
}

func TestCheckoutLanguage(t *testing.T) {
	for in, want := range map[string]string{
		"": "", "en": "en", "en-US": "en", "zh": "zh-Hans", "zh-CN": "zh-Hans", "zh_Hans": "zh-Hans",
		"zh-Hant": "zh-Hant", "zh-TW": "zh-Hant", "ja": "ja-JP", "ko": "ko-KR", "es": "es-MX", "pt-BR": "pt-BR", "xx": "",
	} {
		if got := checkoutLanguage(in); got != want {
			t.Errorf("checkoutLanguage(%q) = %q, want %q", in, got, want)
		}
	}
}

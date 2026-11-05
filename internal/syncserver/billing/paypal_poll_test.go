package billing

import (
	"context"
	"testing"
	"time"
)

func noEmail() string { return "" }

// pendingCheckoutFor makes a PayPal checkout for the account, which leaves I-NEW pending.
func pendingCheckoutFor(t *testing.T, p *ppService, account int64) {
	t.Helper()
	if _, err := p.PayPalCheckout(context.Background(), account, "a@example.com", PlanMonthly); err != nil {
		t.Fatal(err)
	}
}

func TestPayPalCheckoutRemembersItsSubscription(t *testing.T) {
	p := newPayPalService(t, PayPalSandbox, 7)
	pendingCheckoutFor(t, p, 7)
	if got := p.store.pending[7]; got.id != "I-NEW" || !got.at.Equal(testNow) {
		t.Fatalf("pending = %+v", got)
	}
}

func TestStatusFindsAnActivationWhoseWebhookWasLost(t *testing.T) {
	p := newPayPalService(t, PayPalSandbox, 7)
	pendingCheckoutFor(t, p, 7)
	p.paypal.setSub("I-NEW", ppTrialing("I-NEW", "7", testNow).json())

	st, err := p.Status(context.Background(), 7, noEmail)
	if err != nil || st.Plan != "pro" || st.Status != StatusTrialing || st.Provider != ProviderPayPal {
		t.Fatalf("Status = %+v, %v", st, err)
	}
	if row := p.store.subs[7]; row.OrderID != "I-NEW" || row.Provider != ProviderPayPal || row.Mode != ModeTest || !row.TrialUsed {
		t.Fatalf("row = %+v", row)
	}
	if _, _, ok, _ := p.store.PendingPayPal(context.Background(), 7); ok {
		t.Error("the pending checkout was kept after the row held it")
	}
	if got := p.notifications(); len(got) != 1 || got[0] != 7 {
		t.Errorf("notified %v", got)
	}
	// The row holds it now: the next read asks PayPal for nothing more within the refresh interval.
	if _, err := p.Status(context.Background(), 7, noEmail); err != nil || p.paypal.gets("I-NEW") != 1 {
		t.Fatalf("second Status: %v, %d reads", err, p.paypal.gets("I-NEW"))
	}
}

func TestStatusIgnoresASubscriptionOfAnotherAccount(t *testing.T) {
	p := newPayPalService(t, PayPalSandbox, 7, 8)
	pendingCheckoutFor(t, p, 7)
	p.paypal.setSub("I-NEW", ppTrialing("I-NEW", "8", testNow).json())

	st, err := p.Status(context.Background(), 7, noEmail)
	if err != nil || st.Plan != "free" {
		t.Fatalf("Status = %+v, %v", st, err)
	}
	if len(p.store.subs) != 0 {
		t.Fatalf("rows = %+v", p.store.subs)
	}
	if _, _, ok, _ := p.store.PendingPayPal(context.Background(), 7); ok {
		t.Error("a pending checkout that can never match was kept")
	}
	if len(p.notifications()) != 0 {
		t.Errorf("notified %v", p.notifications())
	}
}

func TestStatusPollsAPendingCheckoutAtMostEveryTenSeconds(t *testing.T) {
	ctx := context.Background()
	p := newPayPalService(t, PayPalSandbox, 7)
	pendingCheckoutFor(t, p, 7)
	// The payer has not approved yet.
	p.paypal.setSub("I-NEW", ppSubOpts{id: "I-NEW", status: "APPROVAL_PENDING", plan: ppMonthlyTrial, customID: "7", updated: testNow}.json())

	for i := 0; i < 3; i++ {
		if st, err := p.Status(ctx, 7, noEmail); err != nil || st.Plan != "free" {
			t.Fatalf("Status = %+v, %v", st, err)
		}
	}
	if n := p.paypal.gets("I-NEW"); n != 1 {
		t.Fatalf("%d reads within 10 seconds, want 1", n)
	}
	if _, _, ok, _ := p.store.PendingPayPal(ctx, 7); !ok {
		t.Fatal("the pending checkout was dropped before the payer approved")
	}

	p.advance(11 * time.Second)
	p.paypal.setSub("I-NEW", ppTrialing("I-NEW", "7", p.now).json())
	if st, err := p.Status(ctx, 7, noEmail); err != nil || st.Plan != "pro" {
		t.Fatalf("Status after approval = %+v, %v", st, err)
	}
	if n := p.paypal.gets("I-NEW"); n != 2 {
		t.Fatalf("%d reads, want 2", n)
	}
}

func TestStatusSurvivesPayPalBeingDown(t *testing.T) {
	ctx := context.Background()
	p := newPayPalService(t, PayPalSandbox, 7)
	pendingCheckoutFor(t, p, 7)
	p.paypal.showStatus = 500

	st, err := p.Status(ctx, 7, noEmail)
	if err != nil || st.Plan != "free" {
		t.Fatalf("Status = %+v, %v", st, err)
	}
	if _, _, ok, _ := p.store.PendingPayPal(ctx, 7); !ok {
		t.Error("a PayPal outage dropped the pending checkout")
	}

	// The same for an existing PayPal row: the stored row answers.
	p.store.subs[7] = Subscription{Account: 7, Provider: ProviderPayPal, OrderID: "I-1", Status: StatusActive, Mode: ModeTest,
		PeriodEnd: testNow.AddDate(0, 1, 0), EventAt: testNow.Add(-time.Hour)}
	if st, err := p.Status(ctx, 7, noEmail); err != nil || st.Plan != "pro" {
		t.Fatalf("Status with a PayPal row = %+v, %v", st, err)
	}
}

func TestStatusDropsAPendingCheckoutAfterThreeDaysOrWhenPayPalDoesNotKnowIt(t *testing.T) {
	ctx := context.Background()
	p := newPayPalService(t, PayPalSandbox, 7)
	pendingCheckoutFor(t, p, 7)
	p.advance(3*24*time.Hour + time.Minute)
	p.paypal.setSub("I-NEW", ppTrialing("I-NEW", "7", testNow).json())
	if st, err := p.Status(ctx, 7, noEmail); err != nil || st.Plan != "free" {
		t.Fatalf("Status = %+v, %v", st, err)
	}
	if _, _, ok, _ := p.store.PendingPayPal(ctx, 7); ok || p.paypal.gets("I-NEW") != 0 {
		t.Errorf("an old pending checkout was kept or read (%d reads)", p.paypal.gets("I-NEW"))
	}

	q := newPayPalService(t, PayPalSandbox, 7)
	pendingCheckoutFor(t, q, 7) // PayPal answers 404 for I-NEW: nothing was set
	if _, err := q.Status(ctx, 7, noEmail); err != nil {
		t.Fatal(err)
	}
	if _, _, ok, _ := q.store.PendingPayPal(ctx, 7); ok {
		t.Error("a subscription PayPal does not know was kept")
	}
}

func TestStatusForgetsThePendingCheckoutOnceAWebhookDeliveredIt(t *testing.T) {
	ctx := context.Background()
	p := newPayPalService(t, PayPalSandbox, 7)
	pendingCheckoutFor(t, p, 7)
	p.paypal.setSub("I-NEW", ppTrialing("I-NEW", "7", testNow).json())
	if _, outcome, err := p.deliver("WH-1", "BILLING.SUBSCRIPTION.ACTIVATED", "I-NEW"); err != nil || outcome != OutcomeApplied {
		t.Fatalf("deliver = %s, %v", outcome, err)
	}
	reads := p.paypal.gets("I-NEW")
	if st, err := p.Status(ctx, 7, noEmail); err != nil || st.Plan != "pro" {
		t.Fatalf("Status = %+v, %v", st, err)
	}
	if _, _, ok, _ := p.store.PendingPayPal(ctx, 7); ok {
		t.Error("the pending checkout was kept")
	}
	if n := p.paypal.gets("I-NEW"); n != reads+1 {
		// The pending entry is dropped without a read; the row's own refresh is the one read.
		t.Errorf("%d reads, had %d", n, reads)
	}
}

func TestStatusRefreshesAPayPalRowAtMostEveryTenMinutes(t *testing.T) {
	ctx := context.Background()
	p := newPayPalService(t, PayPalSandbox, 7, 8)
	paid := ppPaid("I-1", "7", testNow.Add(-time.Hour))
	p.store.subs[7] = Subscription{Account: 7, Provider: ProviderPayPal, OrderID: "I-1", Status: StatusActive, Mode: ModeTest,
		Interval: "monthly", PeriodStart: paid.lastPayment, PeriodEnd: paid.nextBilling, EventAt: paid.updated}
	// The payer cancelled in PayPal and the webhook never came.
	cancelled := paid
	cancelled.status, cancelled.updated = "CANCELLED", testNow
	p.paypal.setSub("I-1", cancelled.json())

	st, err := p.Status(ctx, 7, noEmail)
	if err != nil || st.Status != StatusCanceling || st.Plan != "pro" {
		t.Fatalf("Status = %+v, %v", st, err)
	}
	p.advance(5 * time.Minute)
	if _, err := p.Status(ctx, 7, noEmail); err != nil || p.paypal.gets("I-1") != 1 {
		t.Fatalf("second Status within 10 minutes: %v, %d reads", err, p.paypal.gets("I-1"))
	}
	p.advance(6 * time.Minute)
	if _, err := p.Status(ctx, 7, noEmail); err != nil || p.paypal.gets("I-1") != 2 {
		t.Fatalf("Status after 10 minutes: %v, %d reads", err, p.paypal.gets("I-1"))
	}

	// Waffo rows are not PayPal's business.
	p.store.subs[8] = Subscription{Account: 8, OrderID: "ORD_1", Status: StatusActive, Mode: ModeTest, PeriodEnd: testNow.AddDate(0, 1, 0)}
	if _, err := p.Status(ctx, 8, noEmail); err != nil || p.paypal.gets("ORD_1") != 0 {
		t.Fatalf("a Waffo row was read from PayPal (%d reads, %v)", p.paypal.gets("ORD_1"), err)
	}
}

func TestIsProSubjectRecoversARenewalWhoseWebhookWasLost(t *testing.T) {
	ctx := context.Background()
	p := newPayPalService(t, PayPalSandbox, 7)
	// The period ended two days ago, past the billing lag, and no sale webhook followed.
	due := testNow.Add(-2 * 24 * time.Hour)
	p.store.subs[7] = Subscription{Account: 7, Provider: ProviderPayPal, OrderID: "I-1", Status: StatusActive, Mode: ModeTest,
		Interval: "monthly", PeriodEnd: due, EventAt: due.AddDate(0, -1, 0)}
	renewed := ppPaid("I-1", "7", testNow.Add(-time.Hour))
	renewed.nextBilling = due.AddDate(0, 1, 0)
	p.paypal.setSub("I-1", renewed.json())

	if !p.IsProSubject(ctx, "subject-7") {
		t.Fatal("a renewed subscription is not Pro")
	}
	if p.paypal.gets("I-1") != 1 {
		t.Fatalf("%d reads", p.paypal.gets("I-1"))
	}
	// Pro now: no more reads.
	if !p.IsProSubject(ctx, "subject-7") || p.paypal.gets("I-1") != 1 {
		t.Fatalf("a Pro subscription was read again (%d reads)", p.paypal.gets("I-1"))
	}
}

func TestIsProSubjectReadsALapsedSubscriptionAtMostEveryTenMinutes(t *testing.T) {
	ctx := context.Background()
	p := newPayPalService(t, PayPalSandbox, 7, 8)
	due := testNow.Add(-2 * 24 * time.Hour)
	p.store.subs[7] = Subscription{Account: 7, Provider: ProviderPayPal, OrderID: "I-1", Status: StatusActive, Mode: ModeTest, PeriodEnd: due, EventAt: due}
	p.paypal.setSub("I-1", ppSubOpts{id: "I-1", status: "SUSPENDED", plan: ppMonthly, customID: "7", updated: testNow, nextBilling: due, failed: 1}.json())
	for i := 0; i < 3; i++ {
		if p.IsProSubject(ctx, "subject-7") {
			t.Fatal("a suspended subscription past its grace is Pro")
		}
	}
	if n := p.paypal.gets("I-1"); n != 1 {
		t.Fatalf("%d reads, want 1", n)
	}
	// What ended for good is not asked about.
	p.store.subs[8] = Subscription{Account: 8, Provider: ProviderPayPal, OrderID: "I-2", Status: StatusCanceled, Mode: ModeTest, PeriodEnd: due, EventAt: due}
	if p.IsProSubject(ctx, "subject-8") || p.paypal.gets("I-2") != 0 {
		t.Fatalf("a cancelled subscription was read (%d reads)", p.paypal.gets("I-2"))
	}
}

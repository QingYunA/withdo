package billing

import (
	"context"
	"slices"
	"testing"
	"time"
)

func TestProEntitlement(t *testing.T) {
	end := testNow.Add(24 * time.Hour)
	for _, tc := range []struct {
		name string
		sub  Subscription
		now  time.Time
		want bool
	}{
		{"never subscribed", Subscription{}, testNow, false},
		{"trialing", Subscription{Status: StatusTrialing, PeriodEnd: end}, testNow, true},
		{"active", Subscription{Status: StatusActive, PeriodEnd: end}, testNow, true},
		{"canceling keeps access", Subscription{Status: StatusCanceling, PeriodEnd: end}, testNow, true},
		{"canceled", Subscription{Status: StatusCanceled, PeriodEnd: end}, testNow, false},
		{"period over", Subscription{Status: StatusActive, PeriodEnd: end}, end, false},
		{"canceling after the period", Subscription{Status: StatusCanceling, PeriodEnd: end}, end.Add(time.Second), false},
		{"past due within the period", Subscription{Status: StatusPastDue, PeriodEnd: end}, testNow, true},
		{"past due gets the retry day", Subscription{Status: StatusPastDue, PeriodEnd: end}, end.Add(23 * time.Hour), true},
		{"past due after the retry day", Subscription{Status: StatusPastDue, PeriodEnd: end}, end.Add(25 * time.Hour), false},
		{"active without a period end", Subscription{Status: StatusActive}, testNow, true},
		{"unknown status", Subscription{Status: "paused", PeriodEnd: end}, testNow, false},
	} {
		if got := tc.sub.Pro(tc.now); got != tc.want {
			t.Errorf("%s: Pro = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// send delivers an event for account 7 at testNow+at.
func (ts *testService) send(t *testing.T, id, eventType string, at time.Duration, extra map[string]any) {
	t.Helper()
	body := webhookBody(id, eventType, 7, testNow.Add(at), extra)
	if _, err := ts.deliver(t, body, testNow); err != nil {
		t.Fatalf("%s: %v", id, err)
	}
}

func (ts *testService) row(t *testing.T) Subscription {
	t.Helper()
	sub, ok, _ := ts.store.Get(context.Background(), 7)
	if !ok {
		t.Fatal("no row for the account")
	}
	return sub
}

func TestSubscriptionLifecycle(t *testing.T) {
	ts := newTestService(t, 7)
	trialEnd := testNow.Add(7 * 24 * time.Hour)
	period1End := trialEnd.Add(30 * 24 * time.Hour)
	period2End := period1End.Add(30 * 24 * time.Hour)

	if ts.IsProSubject(context.Background(), "subject-7") {
		t.Fatal("Pro before any event")
	}
	ts.send(t, "1", "subscription.activated", 0, map[string]any{
		"orderStatus": "active", "periodNumber": 0, "billingPeriod": "monthly",
		"currentPeriodStart": testNow.Format(time.RFC3339), "currentPeriodEnd": trialEnd.Format(time.RFC3339)})
	row := ts.row(t)
	if row.Status != StatusTrialing || !row.TrialUsed || row.OrderID != "ORD_1" || !row.PeriodEnd.Equal(trialEnd) || row.Mode != ModeTest {
		t.Fatalf("after activation: %+v", row)
	}
	if !ts.IsProSubject(context.Background(), "subject-7") {
		t.Fatal("not Pro while trialing")
	}

	// The trial converts: a real charge, then the new period.
	ts.send(t, "2", "subscription.payment_succeeded", time.Minute, map[string]any{"periodNumber": 1, "paymentStatus": "succeeded"})
	if got := ts.row(t); got.Status != StatusActive || !got.PeriodEnd.Equal(trialEnd) {
		t.Fatalf("after the first charge: %+v", got)
	}
	ts.send(t, "3", "subscription.renewed", 2*time.Minute, map[string]any{
		"orderStatus": "active", "currentPeriodStart": trialEnd.Format(time.RFC3339), "currentPeriodEnd": period1End.Format(time.RFC3339)})
	if got := ts.row(t); !got.PeriodEnd.Equal(period1End) {
		t.Fatalf("after renewal: %+v", got)
	}

	// A late retry of an older event must not undo the newer state.
	ts.send(t, "4", "subscription.past_due", time.Minute, nil)
	if got := ts.row(t); got.Status != StatusActive {
		t.Fatalf("an older event won: %+v", got)
	}

	ts.send(t, "5", "subscription.past_due", 3*time.Minute, nil)
	if got := ts.row(t); got.Status != StatusPastDue {
		t.Fatalf("past_due: %+v", got)
	}
	ts.send(t, "6", "subscription.recovered", 4*time.Minute, map[string]any{"currentPeriodEnd": period2End.Format(time.RFC3339)})
	ts.send(t, "7", "subscription.canceling", 5*time.Minute, map[string]any{"currentPeriodEnd": period2End.Format(time.RFC3339)})
	if got := ts.row(t); got.Status != StatusCanceling || !got.PeriodEnd.Equal(period2End) || !ts.IsProSubject(context.Background(), "subject-7") {
		t.Fatalf("canceling: %+v", got)
	}
	ts.send(t, "8", "subscription.uncanceled", 6*time.Minute, nil)
	if got := ts.row(t); got.Status != StatusActive {
		t.Fatalf("uncanceled: %+v", got)
	}
	ts.send(t, "9", "subscription.canceling", 7*time.Minute, nil)
	ts.send(t, "10", "subscription.canceled", 8*time.Minute, nil)
	if got := ts.row(t); got.Status != StatusCanceled || !got.TrialUsed || ts.IsProSubject(context.Background(), "subject-7") {
		t.Fatalf("canceled: %+v", got)
	}

	// Devices were told as the plan changed.
	if got := ts.notifications(); len(got) == 0 || !slices.Equal(got[:1], []int64{7}) {
		t.Fatalf("notifications = %v", got)
	}
}

func TestPlanChangeTakesOverTheOrderAndStaleOrdersAreIgnored(t *testing.T) {
	ts := newTestService(t, 7)
	end := testNow.Add(30 * 24 * time.Hour)
	ts.send(t, "1", "subscription.activated", 0, map[string]any{"periodNumber": 1, "currentPeriodEnd": end.Format(time.RFC3339)})
	if got := ts.row(t); got.Status != StatusActive {
		t.Fatalf("a charged first period is active: %+v", got)
	}

	// The buyer moves to the yearly plan: a new order.
	yearEnd := testNow.Add(365 * 24 * time.Hour)
	ts.send(t, "2", "subscription.plan_changed", time.Minute, map[string]any{
		"orderId": "ORD_2", "productName": "Withdo Pro - Yearly", "currentPeriodEnd": yearEnd.Format(time.RFC3339)})
	if got := ts.row(t); got.OrderID != "ORD_2" || !got.PeriodEnd.Equal(yearEnd) || got.ProductName != "Withdo Pro - Yearly" {
		t.Fatalf("after the plan change: %+v", got)
	}
	// The old order is closed afterwards: that must not revoke the new plan.
	ts.send(t, "3", "subscription.canceled", 2*time.Minute, map[string]any{"orderId": "ORD_1"})
	if got := ts.row(t); got.Status != StatusActive || got.OrderID != "ORD_2" {
		t.Fatalf("a stale order's event applied: %+v", got)
	}
	// A resubscription after cancellation starts a new row state, keeping the trial flag.
	ts.send(t, "4", "subscription.canceled", 3*time.Minute, map[string]any{"orderId": "ORD_2"})
	ts.send(t, "5", "subscription.activated", 4*time.Minute, map[string]any{"orderId": "ORD_3", "periodNumber": 1, "currentPeriodEnd": end.Format(time.RFC3339)})
	if got := ts.row(t); got.Status != StatusActive || got.OrderID != "ORD_3" || !got.TrialUsed {
		t.Fatalf("resubscribed: %+v", got)
	}
}

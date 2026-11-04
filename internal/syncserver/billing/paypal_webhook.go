package billing

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"
)

// PayPalEvent is the part of a PayPal webhook the log line carries. Account is 0 when the event
// is not about a Withdo account.
type PayPalEvent struct {
	ID, Type, SubscriptionID string
	Account                  int64
}

// paypalEnvelope is what is read from a delivery before PayPal's state is asked for. Nothing in it
// decides an entitlement: it only names the subscription to look up.
type paypalEnvelope struct {
	ID       string `json:"id"`
	Type     string `json:"event_type"`
	Resource struct {
		ID                 string `json:"id"`
		BillingAgreementID string `json:"billing_agreement_id"`
	} `json:"resource"`
}

// subscriptionID is the PayPal subscription an event is about: a BILLING.SUBSCRIPTION.* event's
// resource is the subscription; a PAYMENT.SALE.* event's resource is the sale, which names its
// subscription as billing_agreement_id. Other events (and sales without one) name none.
func (e paypalEnvelope) subscriptionID() string {
	switch {
	case strings.HasPrefix(e.Type, "BILLING.SUBSCRIPTION."):
		return e.Resource.ID
	case strings.HasPrefix(e.Type, "PAYMENT.SALE."):
		return e.Resource.BillingAgreementID
	}
	return ""
}

// PayPalEnabled says whether PayPal is configured.
func (s *Service) PayPalEnabled() bool { return s.paypal != nil }

// HandlePayPalWebhook verifies a PayPal webhook delivery with PayPal itself, reads the
// subscription it is about from PayPal and applies that state. The event's own fields are never
// the state: a delivery only says which subscription changed, so a replayed, reordered or
// repeated event cannot put a stale status in the row. Verification failures are ErrBadSignature;
// any other error is ours or PayPal's (the database, PayPal being down), and the caller answers
// 5xx so PayPal sends the event again. Events that are not about a Withdo account are ignored,
// not errors.
func (s *Service) HandlePayPalWebhook(ctx context.Context, h PayPalHeaders, body []byte) (PayPalEvent, Outcome, error) {
	if s.paypal == nil {
		return PayPalEvent{}, "", ErrPayPalOff
	}
	var env paypalEnvelope
	if !h.complete() || json.Unmarshal(body, &env) != nil {
		return PayPalEvent{}, "", ErrBadSignature
	}
	ev := PayPalEvent{ID: env.ID, Type: env.Type, SubscriptionID: env.subscriptionID()}
	genuine, err := s.paypal.verifyWebhook(ctx, h, body)
	if err != nil {
		return ev, "", err
	}
	if !genuine {
		return ev, "", ErrBadSignature
	}
	if ev.ID == "" || ev.SubscriptionID == "" || !paypalEventTypes[ev.Type] {
		return ev, OutcomeIgnored, nil
	}
	sub, found, err := s.paypal.getSubscription(ctx, ev.SubscriptionID)
	if err != nil {
		return ev, "", err
	}
	account, ok := paypalAccount(sub.CustomID)
	if !found || !ok || sub.ID != ev.SubscriptionID {
		return ev, OutcomeIgnored, nil
	}
	ev.Account = account
	now := s.cfg.Now()
	mode := paypalMode(s.paypal.cfg.Env)
	before, after, dup, saved, err := s.store.Apply(ctx, "paypal:"+ev.ID, ev.Type, account,
		func(cur Subscription, exists bool) (Subscription, bool) {
			return reducePayPal(cur, exists, account, sub, mode, s.paypal.intervalOf(sub.PlanID), now)
		})
	switch {
	case errors.Is(err, ErrNoAccount):
		return ev, OutcomeIgnored, nil
	case err != nil:
		return ev, "", err
	case dup:
		return ev, OutcomeDuplicate, nil
	case !saved:
		return ev, OutcomeIgnored, nil
	}
	if s.notify != nil && (before.Status != after.Status || !before.PeriodEnd.Equal(after.PeriodEnd) ||
		before.Pro(now) != after.Pro(now)) {
		s.notify(account)
	}
	return ev, OutcomeApplied, nil
}

// paypalEventTypes are the deliveries that make a subscription worth reading again: the ones the
// webhook is registered for (scripts/paypal-setup.py).
var paypalEventTypes = map[string]bool{
	"BILLING.SUBSCRIPTION.ACTIVATED":      true,
	"BILLING.SUBSCRIPTION.UPDATED":        true,
	"BILLING.SUBSCRIPTION.CANCELLED":      true,
	"BILLING.SUBSCRIPTION.SUSPENDED":      true,
	"BILLING.SUBSCRIPTION.EXPIRED":        true,
	"BILLING.SUBSCRIPTION.RE-ACTIVATED":   true,
	"BILLING.SUBSCRIPTION.PAYMENT.FAILED": true,
	"PAYMENT.SALE.COMPLETED":              true,
	"PAYMENT.SALE.REFUNDED":               true,
}

// paypalAccount reads the account out of a subscription's custom_id (set at checkout).
func paypalAccount(customID string) (int64, bool) {
	id, err := strconv.ParseInt(strings.TrimSpace(customID), 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

// inTrial says whether an ACTIVE subscription is still in its free trial, or in the gap between the
// trial's end and the first charge: its plan has a TRIAL cycle, and either that cycle has runs left
// or no REGULAR cycle has been charged yet. A failed payment is not a trial.
func inTrial(cycles []paypalCycle, failedPayments int) bool {
	var hasTrial, trialRunning, regularCharged bool
	for _, c := range cycles {
		switch c.TenureType {
		case "TRIAL":
			hasTrial = true
			if c.CyclesCompleted < c.TotalCycles {
				trialRunning = true
			}
		case "REGULAR":
			if c.CyclesCompleted > 0 {
				regularCharged = true
			}
		}
	}
	return hasTrial && failedPayments == 0 && (trialRunning || !regularCharged)
}

// hasTrialCycle says whether the subscription's plan had a free trial: the account has used its trial.
func hasTrialCycle(cycles []paypalCycle) bool {
	for _, c := range cycles {
		if c.TenureType == "TRIAL" {
			return true
		}
	}
	return false
}

// paidThrough is when the last money paid for runs out, the best of what PayPal and the row know:
// next_billing_time (after a cancellation PayPal usually drops it), the period the row already
// had, the last real payment plus one interval.
func paidThrough(sub paypalSub, cur Subscription, same bool, interval string) time.Time {
	b := sub.BillingInfo
	if !b.NextBillingTime.IsZero() {
		return b.NextBillingTime.Time
	}
	if same && !cur.PeriodEnd.IsZero() {
		return cur.PeriodEnd
	}
	if value, err := strconv.ParseFloat(b.LastPayment.Amount.Value, 64); err == nil && value > 0 && !b.LastPayment.Time.IsZero() {
		switch interval {
		case "monthly":
			return b.LastPayment.Time.AddDate(0, 1, 0)
		case "yearly":
			return b.LastPayment.Time.AddDate(1, 0, 0)
		}
	}
	return time.Time{}
}

// snapshotTime is when PayPal last changed the subscription: the newest timestamp it carries. It
// orders two reads of the same subscription, so that one taken earlier but applied later (two
// deliveries racing) does not undo the newer one. now stands in for a resource with none.
func snapshotTime(sub paypalSub, now time.Time) time.Time {
	at := sub.UpdateTime.Time
	for _, t := range []time.Time{sub.StatusUpdateTime.Time, sub.BillingInfo.LastPayment.Time.Time} {
		if t.After(at) {
			at = t
		}
	}
	if at.IsZero() {
		return now
	}
	return at
}

// reducePayPal puts what PayPal says about a subscription into the account's row. The PayPal
// subscription takes the row over when it is ACTIVE (like Waffo's activated); the later states of
// a subscription that is not the row's are ignored, as is a read older than the row.
func reducePayPal(cur Subscription, exists bool, account int64, sub paypalSub, mode, interval string, now time.Time) (Subscription, bool) {
	same := exists && cur.providerOrDefault() == ProviderPayPal && cur.OrderID == sub.ID
	at := snapshotTime(sub, now)
	if exists && at.Before(cur.EventAt) {
		return cur, false
	}
	b := sub.BillingInfo
	next := cur
	next.Account = account
	end := paidThrough(sub, cur, same, interval)
	switch sub.Status {
	case "ACTIVE":
		next.Status = StatusActive
		if inTrial(b.CycleExecutions, b.FailedPaymentsCount) {
			next.Status = StatusTrialing
		}
		next.PeriodEnd = end
		next.PeriodStart = sub.StartTime.Time
		if p := b.LastPayment.Time.Time; !p.IsZero() {
			next.PeriodStart = p
		}
	case "SUSPENDED":
		if exists && !same {
			return cur, false
		}
		// A failed renewal: Pro for the grace after the period the payments covered.
		next.Status = StatusPastDue
		if end.IsZero() {
			end = at
		}
		next.PeriodEnd = end
	case "CANCELLED":
		if exists && !same {
			return cur, false
		}
		// Cancelled by the payer (or us): what was paid for stays Pro until its end.
		if end.IsZero() {
			end = at
		}
		next.PeriodEnd = end
		next.Status = StatusCanceled
		if end.After(now) {
			next.Status = StatusCanceling
		}
	case "EXPIRED":
		if exists && !same {
			return cur, false
		}
		next.Status = StatusCanceled
		next.PeriodEnd = at
	default: // APPROVAL_PENDING and APPROVED: nothing has been bought yet
		return cur, false
	}
	next.Provider = ProviderPayPal
	next.OrderID = sub.ID
	next.ProductName = "Withdo Pro"
	if interval != "" {
		next.Interval = interval
	} else if !same {
		next.Interval = ""
	}
	// The trial is the account's, once, on either channel: a plan with a TRIAL cycle used it.
	next.TrialUsed = cur.TrialUsed || hasTrialCycle(b.CycleExecutions)
	next.Mode = mode
	next.EventAt = at
	return next, true
}

package billing

import (
	"strings"
	"time"
)

// Subscription statuses, as Withdo stores them. Waffo's own words are active, canceling, canceled
// and past_due; trialing is an active subscription whose first charge has not happened.
const (
	StatusTrialing  = "trialing"
	StatusActive    = "active"
	StatusPastDue   = "past_due"
	StatusCanceling = "canceling"
	StatusCanceled  = "canceled"
)

// The payment channels a subscription can be on (M17). Both write the same row.
const (
	ProviderWaffo  = "waffo"
	ProviderPayPal = "paypal"
)

// paypalBillingLag is how long after next_billing_time PayPal may take to charge a renewal (or the
// first charge after a trial) and tell us; an active PayPal subscription stays Pro for it, so the
// minutes between the due time and the PAYMENT.SALE.COMPLETED webhook are not a lapse.
const paypalBillingLag = 24 * time.Hour

// pastDueGrace is the one retry day Waffo's channel gives a failed renewal: a past_due
// subscription stays Pro for it after the period ended.
const pastDueGrace = 24 * time.Hour

// Subscription is an account's row: the latest state the newest subscription's events put it in.
type Subscription struct {
	Account int64
	// Provider is ProviderWaffo or ProviderPayPal; empty reads as Waffo (see providerOrDefault).
	Provider string
	// OrderID is Waffo's order id, or the PayPal subscription id (I-...) for a PayPal row.
	OrderID     string
	ProductName string
	// Interval is "monthly" or "yearly", or empty when no event has said (see normalizeInterval).
	Interval    string
	Status      string
	PeriodStart time.Time // zero when unknown
	PeriodEnd   time.Time // zero when unknown
	TrialUsed   bool
	Mode        string
	// EventAt is the timestamp of the newest event applied; an older one arriving late is dropped.
	EventAt time.Time
}

// providerOrDefault is the channel the row is on; rows from before M17 have none and are Waffo's.
func (s Subscription) providerOrDefault() string {
	if s.Provider == "" {
		return ProviderWaffo
	}
	return s.Provider
}

// Pro is the entitlement rule: Pro while the status is trialing, active, canceling or past_due and
// the period has not ended (a past_due one gets pastDueGrace more, a trialing or active PayPal one
// paypalBillingLag). A period end the provider did not send leaves the status to decide.
func (s Subscription) Pro(now time.Time) bool {
	switch s.Status {
	case StatusTrialing, StatusActive:
		end := s.PeriodEnd
		if s.providerOrDefault() == ProviderPayPal {
			end = end.Add(paypalBillingLag)
		}
		return s.PeriodEnd.IsZero() || now.Before(end)
	case StatusCanceling:
		return s.PeriodEnd.IsZero() || now.Before(s.PeriodEnd)
	case StatusPastDue:
		return s.PeriodEnd.IsZero() || now.Before(s.PeriodEnd.Add(pastDueGrace))
	}
	return false
}

// normalizeInterval maps Waffo's billingPeriod (monthly, yearly, and weekly or quarterly, which
// Withdo does not sell) to "monthly" or "yearly". A period that says neither falls back to the
// product's name ("Withdo Pro - Yearly"); anything else is empty.
func normalizeInterval(billingPeriod, productName string) string {
	for _, text := range []string{billingPeriod, productName} {
		t := strings.ToLower(text)
		switch {
		case strings.Contains(t, "year"), strings.Contains(t, "annual"):
			return "yearly"
		case strings.Contains(t, "month"):
			return "monthly"
		}
	}
	return ""
}

// reduce applies one event to an account's row (cur is the zero value with exists false before
// the first event) and says whether the row changed. Events that are not about the subscription,
// that belong to an order the account has left, or that are older than the row are ignored.
func reduce(cur Subscription, exists bool, ev Event, account int64) (next Subscription, changed bool) {
	next = cur
	next.Account = account
	at := ev.Timestamp.Time
	if at.IsZero() {
		at = cur.EventAt // an event without a readable timestamp is taken as the newest
	}
	if exists && at.Before(cur.EventAt) {
		return cur, false
	}
	d := ev.Data
	// A Waffo event follows only a Waffo order: a PayPal subscription took the row over if it is one.
	sameOrder := !exists || cur.OrderID == "" || (cur.providerOrDefault() == ProviderWaffo && cur.OrderID == d.OrderID)
	takeOrder := func() {
		next.Provider = ProviderWaffo
		next.OrderID = d.OrderID
		if d.ProductName != "" {
			next.ProductName = d.ProductName
		}
		if interval := normalizeInterval(d.BillingPeriod, d.ProductName); interval != "" {
			next.Interval = interval
		}
	}
	setPeriod := func() {
		if !d.CurrentPeriodStart.IsZero() {
			next.PeriodStart = d.CurrentPeriodStart.Time
		}
		if !d.CurrentPeriodEnd.IsZero() {
			next.PeriodEnd = d.CurrentPeriodEnd.Time
		}
	}
	switch ev.Type {
	case "subscription.activated":
		takeOrder()
		next.Status = StatusActive
		setPeriod()
		// Waffo's payload has no trial flag: a period of 0 or a first period far shorter than a month
		// (the 7-day trial) is the trial. The renewal at its end makes the subscription active.
		if (d.PeriodNumber != nil && *d.PeriodNumber == 0) || trialPeriod(next.PeriodStart, next.PeriodEnd) {
			next.Status = StatusTrialing
		}
		next.TrialUsed = true
	case "subscription.plan_changed":
		takeOrder()
		next.Status = StatusActive
		setPeriod()
	case "subscription.renewed", "subscription.recovered", "subscription.uncanceled":
		if !sameOrder {
			return cur, false
		}
		takeOrder()
		next.Status = StatusActive
		setPeriod()
		// Uncanceling during the trial goes back to the trial; a renewal always starts a paid period.
		if ev.Type == "subscription.uncanceled" && trialPeriod(next.PeriodStart, next.PeriodEnd) {
			next.Status = StatusTrialing
		}
	case "subscription.canceling":
		if !sameOrder {
			return cur, false
		}
		takeOrder()
		next.Status = StatusCanceling
		setPeriod()
	case "subscription.canceled":
		if !sameOrder {
			return cur, false
		}
		takeOrder()
		next.Status = StatusCanceled
		setPeriod()
	case "subscription.past_due":
		if !sameOrder {
			return cur, false
		}
		takeOrder()
		next.Status = StatusPastDue
		setPeriod()
	case "subscription.payment_succeeded":
		// A pure payment event with no period dates. It also comes for the trial's $0 card check, in
		// no fixed order with activated, so it changes nothing: subscription.renewed ends a trial.
		return cur, false
	default:
		return cur, false
	}
	next.Mode = ev.Mode
	next.EventAt = at
	return next, true
}

// trialPeriod reports a billing period shorter than any monthly or yearly plan's: the free trial.
func trialPeriod(start, end time.Time) bool {
	if start.IsZero() || end.IsZero() || !end.After(start) {
		return false
	}
	return end.Sub(start) < 20*24*time.Hour
}

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

// pastDueGrace is the one retry day Waffo's channel gives a failed renewal: a past_due
// subscription stays Pro for it after the period ended.
const pastDueGrace = 24 * time.Hour

// Subscription is an account's row: the latest state Waffo's events put it in.
type Subscription struct {
	Account     int64
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

// Pro is the entitlement rule: Pro while the status is trialing, active, canceling or past_due and
// the period has not ended (a past_due one gets pastDueGrace more). A period end Waffo did not
// send leaves the status to decide.
func (s Subscription) Pro(now time.Time) bool {
	switch s.Status {
	case StatusTrialing, StatusActive, StatusCanceling:
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
	sameOrder := !exists || cur.OrderID == "" || cur.OrderID == d.OrderID
	takeOrder := func() {
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
		if d.PeriodNumber != nil && *d.PeriodNumber == 0 {
			next.Status = StatusTrialing // authorised, not yet charged
		}
		next.TrialUsed = true
		setPeriod()
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
		// A pure payment event: it carries no period dates. The first real charge ends a trial.
		if !sameOrder || cur.Status != StatusTrialing || d.PeriodNumber == nil || *d.PeriodNumber < 1 {
			return cur, false
		}
		next.Status = StatusActive
	default:
		return cur, false
	}
	next.Mode = ev.Mode
	next.EventAt = at
	return next, true
}

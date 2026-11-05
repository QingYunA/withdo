package billing

import (
	"context"
	"log"
	"time"
)

// PayPal's webhooks are how a subscription normally reaches the account's row, but a delivery can be
// lost (it was, in the first sandbox test). These polls are the fallback: the app asks for the plan
// while it waits for the checkout, and the server then reads the subscription from PayPal itself.
const (
	// paypalPendingMax is how long a checkout's subscription is looked for. An approval that was
	// not finished in this time is not coming.
	paypalPendingMax = 3 * 24 * time.Hour
	// paypalPendingEvery spaces the reads of a pending subscription; the apps ask every few seconds.
	paypalPendingEvery = 10 * time.Second
	// paypalRefreshEvery spaces the reads of the subscription the row already holds, which catch a
	// lost cancellation or renewal.
	paypalRefreshEvery = 10 * time.Minute
	// paypalPollTimeout keeps a slow PayPal from holding up the request that polls it.
	paypalPollTimeout = 8 * time.Second
)

// pollDue says whether account may be polled now, and counts the poll when it may: last remembers
// when each account was polled for one purpose.
func (s *Service) pollDue(last map[int64]time.Time, account int64, now time.Time, every time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if at, ok := last[account]; ok && now.Sub(at) < every {
		return false
	}
	last[account] = now
	return true
}

// pollEventID names a read of a subscription by the state PayPal reports, so the same state is
// applied once however often it is read.
func pollEventID(now time.Time) func(paypalSub) string {
	return func(sub paypalSub) string {
		return "paypal-poll:" + sub.ID + ":" + snapshotTime(sub, now).UTC().Format(time.RFC3339Nano)
	}
}

// pollPayPalSubscription reads a subscription from PayPal and applies it to account, like a
// webhook. PayPal being down is no failure of the caller's: it is logged and ok is false.
func (s *Service) pollPayPalSubscription(ctx context.Context, account int64, id string, now time.Time) (applied paypalApplied, ok bool) {
	ctx, cancel := context.WithTimeout(ctx, paypalPollTimeout)
	defer cancel()
	applied, err := s.applyPayPalSubscription(ctx, id, account, "poll", pollEventID(now))
	if err != nil {
		log.Printf("billing: account=%d paypal poll failed: %v", account, err)
		return applied, false
	}
	return applied, true
}

// pollPayPal brings the account's row up to date with PayPal before Status reads it: first the
// subscription its last checkout made, which no webhook has brought yet, then the one the row holds.
func (s *Service) pollPayPal(ctx context.Context, account int64) {
	now := s.cfg.Now()
	row, exists, err := s.store.Get(ctx, account)
	if err != nil {
		return // Status reads the row again and reports it
	}
	id, created, pending, err := s.store.PendingPayPal(ctx, account)
	if err != nil {
		log.Printf("billing: account=%d paypal poll failed: %v", account, err)
	}
	if pending {
		holds := exists && row.providerOrDefault() == ProviderPayPal && row.OrderID == id
		switch {
		case holds || now.Sub(created) > paypalPendingMax:
			s.forgetPendingPayPal(ctx, account, id)
		case s.pollDue(s.pendingPolled, account, now, paypalPendingEvery):
			s.pollPending(ctx, account, id, now)
			return // what PayPal says was just applied; the row's own read can wait
		}
	}
	if exists && s.refreshable(row) {
		s.refreshPayPal(ctx, row, now)
	}
}

// pollPending reads the subscription of the account's checkout. It is forgotten once the row holds
// it, or when PayPal does not know it or it is not the account's; until the payer approves it
// (APPROVAL_PENDING) it stays.
func (s *Service) pollPending(ctx context.Context, account int64, id string, now time.Time) {
	applied, ok := s.pollPayPalSubscription(ctx, account, id, now)
	if !ok {
		return
	}
	if applied.account == 0 {
		s.forgetPendingPayPal(ctx, account, id)
		return
	}
	// PayPal was just asked about this subscription: the row's own refresh counts it as read.
	s.pollDue(s.refreshed, account, now, paypalRefreshEvery)
	if row, exists, err := s.store.Get(ctx, account); err == nil && exists &&
		row.providerOrDefault() == ProviderPayPal && row.OrderID == id {
		s.forgetPendingPayPal(ctx, account, id)
	}
}

func (s *Service) forgetPendingPayPal(ctx context.Context, account int64, id string) {
	if err := s.store.ClearPendingPayPal(ctx, account, id); err != nil {
		log.Printf("billing: account=%d %v", account, err)
	}
}

// refreshable says whether the row is a PayPal subscription of this mode that can still change. A
// canceled one cannot: PayPal does not bring a subscription back from the end.
func (s *Service) refreshable(row Subscription) bool {
	return row.providerOrDefault() == ProviderPayPal && s.thisMode(row) && row.Status != StatusCanceled
}

// refreshPayPal reads the subscription the row holds, at most once per paypalRefreshEvery for the
// account, and says whether that changed the row.
func (s *Service) refreshPayPal(ctx context.Context, row Subscription, now time.Time) bool {
	if !s.pollDue(s.refreshed, row.Account, now, paypalRefreshEvery) {
		return false
	}
	applied, ok := s.pollPayPalSubscription(ctx, row.Account, row.OrderID, now)
	return ok && applied.outcome == OutcomeApplied
}

// lapsedByTime says whether the row is a PayPal subscription that is not Pro now only because its
// period ran out: a renewal whose webhook was lost looks exactly like this.
func (s *Service) lapsedByTime(row Subscription, now time.Time) bool {
	if row.Pro(now) || !s.refreshable(row) {
		return false
	}
	switch row.Status {
	case StatusTrialing, StatusActive, StatusPastDue:
		return true
	}
	return false
}

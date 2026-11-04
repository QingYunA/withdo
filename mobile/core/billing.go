package core

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/QingYunA/adhd-todo/internal/syncclient"
)

// Withdo Pro on the phone (M16): the account's plan, kept like the official model's state: asked
// for in the background, held for a few minutes, and read again when the server says it changed.

const (
	billingRefreshEvery = 5 * time.Minute
	billingRetryAfter   = 30 * time.Second
	billingRefreshTO    = 15 * time.Second
	checkoutTO          = 30 * time.Second
)

// billingState is what Billing reads.
type billingState struct {
	mu        sync.Mutex
	server    string
	signedIn  bool
	known     bool // offered is the server's answer
	offered   bool
	have      bool // b is the account's plan
	b         syncclient.Billing
	rev       int64     // counts the changes the app should read again
	refreshAt time.Time // no refresh before this
	busy      bool
}

// billingJSON is the JSON of Billing.
type billingJSON struct {
	Offered   bool   `json:"offered"`
	Plan      string `json:"plan"`
	Status    string `json:"status"`
	PeriodEnd string `json:"period_end"`
	TrialUsed bool   `json:"trial_used"`
	Monthly   string `json:"monthly"`
	Yearly    string `json:"yearly"`
	ManageURL string `json:"manage_url"`
}

// Billing says what the server sells and the account's plan, as JSON, without touching the network
// (it starts a refresh in the background when its answer is old):
//
//	{"offered":true,"plan":"pro","status":"trialing","period_end":"2026-11-11T09:00:00Z",
//	 "trial_used":true,"monthly":"$6.99","yearly":"$58.72","manage_url":"https://..."}
//
// offered is whether the server sells Withdo Pro (false until known, and on a server that does not).
// plan is free or pro, empty while signed out or not yet read. status is the subscription's
// (trialing, active, past_due, canceling, canceled), empty for an account that never subscribed;
// period_end is when the period ends (RFC 3339 UTC), empty then too. manage_url opens the customer
// portal; it lives 15 minutes, so call RefreshBilling right before opening it. The listener's
// OnStatus fires when the plan changed, with a larger billing_rev: read Billing again then.
func (c *Core) Billing() (string, error) {
	c.mu.Lock()
	closed := c.closed
	c.mu.Unlock()
	if closed {
		return "", errors.New("the core is closed")
	}
	return marshal(c.billingInfo())
}

// RefreshBilling asks the server for the plan now and returns once it has answered; the app calls
// it on coming back from the browser and before opening manage_url. Billing then has the answer.
func (c *Core) RefreshBilling() error {
	ctx, cancel := context.WithTimeout(context.Background(), billingRefreshTO)
	defer cancel()
	s := &c.bill
	s.mu.Lock()
	s.busy = true
	s.mu.Unlock()
	return c.fetchBilling(ctx)
}

// Checkout asks for the checkout page of plan ("monthly" or "yearly") and returns its URL, which
// the app opens in the browser. lang is the app's language, "en" or "zh-Hans", and only picks the
// page's default language. A failure is one of these words: bad_plan, signed_out, off,
// already_subscribed (the account is Pro already: Billing says so after a refresh), rate_limited,
// upstream (the payment provider is down) or network.
func (c *Core) Checkout(plan, lang string) (string, error) {
	if plan != syncclient.PlanMonthly && plan != syncclient.PlanYearly {
		return "", errors.New("bad_plan")
	}
	client, err := c.newClient(false)
	if err != nil {
		return "", errors.New("signed_out")
	}
	ctx, cancel := context.WithTimeout(context.Background(), checkoutTO)
	defer cancel()
	url, err := client.Checkout(ctx, plan, lang)
	if err != nil {
		log.Printf("adhd-todo checkout: %v", err)
		if errors.Is(err, syncclient.ErrAlreadySubscribed) {
			go func() { _ = c.RefreshBilling() }()
		}
		return "", errors.New(checkoutWord(err))
	}
	return url, nil
}

func checkoutWord(err error) string {
	switch {
	case errors.Is(err, syncclient.ErrBillingNotSignedIn):
		return "signed_out"
	case errors.Is(err, syncclient.ErrBillingOff):
		return "off"
	case errors.Is(err, syncclient.ErrAlreadySubscribed):
		return "already_subscribed"
	case errors.Is(err, syncclient.ErrCheckoutThrottled):
		return "rate_limited"
	case errors.Is(err, syncclient.ErrBillingUpstream):
		return "upstream"
	}
	return "network"
}

// billingInfo is the state held, starting a refresh when it is old.
func (c *Core) billingInfo() billingJSON {
	server, signedIn, explicit := c.officialSession()
	s := &c.bill
	s.mu.Lock()
	s.follow(server, signedIn)
	due := (signedIn || explicit) && !s.busy && !time.Now().Before(s.refreshAt)
	s.busy = s.busy || due
	info := s.info()
	s.mu.Unlock()
	if due {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), billingRefreshTO)
			defer cancel()
			_ = c.fetchBilling(ctx)
		}()
	}
	return info
}

// follow forgets what was held when the session is another server or account. Call with mu held.
func (s *billingState) follow(server string, signedIn bool) {
	if s.server != server || s.signedIn != signedIn {
		s.server, s.signedIn, s.known, s.offered, s.have, s.b, s.refreshAt = server, signedIn, false, false, false, syncclient.Billing{}, time.Time{}
		s.rev++
	}
}

// info is the state held. Call with mu held.
func (s *billingState) info() billingJSON {
	if !s.known || !s.offered {
		return billingJSON{}
	}
	out := billingJSON{Offered: true}
	if s.have {
		b := s.b
		out.Plan, out.Status, out.TrialUsed = b.Plan, b.Status, b.TrialUsed
		out.Monthly, out.Yearly, out.ManageURL = b.Monthly, b.Yearly, b.ManageURL
		if !b.PeriodEnd.IsZero() {
			out.PeriodEnd = b.PeriodEnd.UTC().Format(time.RFC3339)
		}
	}
	return out
}

// plan is the plan held, "" when it is not known.
func (s *billingState) plan() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.known || !s.offered || !s.have {
		return ""
	}
	return s.b.Plan
}

// fetchBilling asks the server what it sells and, signed in, the plan. The caller has set busy,
// which this clears. It tells the listener when what the app shows changed.
func (c *Core) fetchBilling(ctx context.Context) error {
	s := &c.bill
	server, signedIn, _ := c.officialSession()
	s.mu.Lock()
	s.follow(server, signedIn)
	s.mu.Unlock()
	offered, plan, havePlan, err := c.askBilling(ctx, server, signedIn)
	s.mu.Lock()
	changed := false
	switch {
	case s.server != server || s.signedIn != signedIn: // the session changed while asking: that answer is old
		s.refreshAt = time.Time{}
	case err != nil:
		s.refreshAt = time.Now().Add(billingRetryAfter)
	default:
		changed = !s.known || s.offered != offered || s.have != havePlan || !sameBilling(s.b, plan)
		s.known, s.offered, s.have, s.b = true, offered, havePlan, plan
		s.refreshAt = time.Now().Add(billingRefreshEvery)
		if changed {
			s.rev++
		}
	}
	s.busy = false
	s.mu.Unlock()
	if changed {
		c.refreshModelSoon() // the day's turns follow the plan
		c.notifyStatus()
	}
	return err
}

// sameBilling compares what the app shows; manage_url is made fresh by every call, so it does not count.
func sameBilling(a, b syncclient.Billing) bool {
	a.ManageURL, b.ManageURL = "", ""
	end := a.PeriodEnd.Equal(b.PeriodEnd)
	a.PeriodEnd, b.PeriodEnd = time.Time{}, time.Time{}
	return end && a == b
}

func (c *Core) askBilling(ctx context.Context, server string, signedIn bool) (offered bool, plan syncclient.Billing, havePlan bool, err error) {
	cfg, err := syncclient.FetchConfig(ctx, server, nil)
	if err != nil {
		return false, plan, false, err
	}
	if !cfg.Billing || !signedIn {
		return cfg.Billing, plan, false, nil
	}
	client, err := c.newClient(false)
	if err != nil {
		return cfg.Billing, plan, false, err
	}
	plan, err = client.Billing(ctx)
	switch {
	case err == nil:
		return true, plan, true, nil
	case errors.Is(err, syncclient.ErrBillingOff):
		return false, syncclient.Billing{}, false, nil
	case errors.Is(err, syncclient.ErrBillingNotSignedIn):
		// The server refused the session: Pro is offered, but nothing is known of the plan.
		return true, syncclient.Billing{}, false, nil
	}
	return false, syncclient.Billing{}, false, err
}

// billingChanged is the runner's OnBillingChange: the server says the plan changed. It runs on the
// event stream's goroutine, so the asking happens on another.
func (c *Core) billingChanged() {
	go func() { _ = c.RefreshBilling() }()
}

// billingRev is the number of changes of the plan held, for the status JSON's billing_rev.
func (c *Core) billingRev() int64 {
	c.bill.mu.Lock()
	defer c.bill.mu.Unlock()
	return c.bill.rev
}

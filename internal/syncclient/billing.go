package syncclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/QingYunA/adhd-todo/internal/syncproto"
)

// Errors Billing and Checkout return for the answers the plan UI words for the person. Test them
// with errors.Is.
var (
	// ErrBillingNotSignedIn says there is no signed-in account: the token source has none, or the
	// server refused the token.
	ErrBillingNotSignedIn = errors.New("syncclient: sign in to see the plan")
	// ErrBillingOff says the server does not sell Withdo Pro (an older server, or billing off).
	ErrBillingOff = errors.New("syncclient: this server does not sell Withdo Pro")
	// ErrBillingNetwork says the server could not be reached.
	ErrBillingNetwork = errors.New("syncclient: the server could not be reached")
	// ErrBillingUpstream says the server could not reach the payment provider; try again later.
	ErrBillingUpstream = errors.New("syncclient: the payment provider is unavailable")
	// ErrAlreadySubscribed says the account is Pro already; it manages the plan at Billing.ManageURL.
	ErrAlreadySubscribed = errors.New("syncclient: the account already has Withdo Pro")
	// ErrCheckoutThrottled says the account asked for too many checkout pages; try again later.
	ErrCheckoutThrottled = errors.New("syncclient: too many checkout attempts")
)

// The plans Checkout takes.
const (
	PlanMonthly = "monthly"
	PlanYearly  = "yearly"
)

// Billing is the account's plan (GET /v1/billing). Plan is "free" or "pro". Status is the
// subscription's (trialing, active, past_due, canceling, canceled), empty for an account that
// never subscribed; Interval is "monthly" or "yearly", empty when the server does not know (or is
// older); PeriodEnd is zero then too. Monthly and Yearly are the prices to show.
// ManageURL opens the customer portal for the subscription, empty without one: it is made fresh by
// every Billing call, lives 15 minutes, and must be opened at once, not stored.
type Billing struct {
	Plan      string
	Status    string
	Interval  string
	PeriodEnd time.Time
	TrialUsed bool
	Monthly   string
	Yearly    string
	ManageURL string
}

// Pro says whether the account is on Withdo Pro now.
func (b Billing) Pro() bool { return b.Plan == "pro" }

// BillingOffered asks GET /v1/config whether the server sells Withdo Pro. It needs no session. An
// older server, one that cannot be reached, and one with billing off all say false.
func (c *Client) BillingOffered(ctx context.Context) bool {
	cfg, err := FetchConfig(ctx, c.base, c.http)
	return err == nil && cfg.Billing
}

// Billing reads GET /v1/billing: the account's plan.
func (c *Client) Billing(ctx context.Context) (Billing, error) {
	var out syncproto.BillingResponse
	if err := c.billingCall(ctx, http.MethodGet, "/v1/billing", nil, &out); err != nil {
		return Billing{}, err
	}
	b := Billing{
		Plan: out.Plan, Status: out.Status, Interval: out.Interval, TrialUsed: out.TrialUsed,
		Monthly: out.Monthly, Yearly: out.Yearly, ManageURL: out.ManageURL,
	}
	if out.PeriodEnd != "" {
		end, err := time.Parse(time.RFC3339, out.PeriodEnd)
		if err != nil {
			return Billing{}, fmt.Errorf("decode /v1/billing: bad period_end")
		}
		b.PeriodEnd = end
	}
	if b.Plan == "" {
		return Billing{}, errors.New("decode /v1/billing: no plan")
	}
	return b, nil
}

// Checkout asks for a checkout page for plan (PlanMonthly or PlanYearly) and returns its URL, which
// the app opens in the browser. lang is the app's language (such as "en" or "zh-Hans") and only
// picks the page's default language. The app learns that the purchase went through from a
// Billing call after the "billing" event (OnBillingChange), not from the browser.
func (c *Client) Checkout(ctx context.Context, plan, lang string) (string, error) {
	var out syncproto.CheckoutResponse
	err := c.billingCall(ctx, http.MethodPost, "/v1/billing/checkout", syncproto.CheckoutRequest{Plan: plan, Lang: lang}, &out)
	if err != nil {
		return "", err
	}
	if out.URL == "" {
		return "", errors.New("decode /v1/billing/checkout: no url")
	}
	return out.URL, nil
}

// billingCall is one billing request with the failures sorted into the package's sentinels.
func (c *Client) billingCall(ctx context.Context, method, path string, body, out any) error {
	err := c.do(ctx, method, path, body, out)
	if err == nil {
		return nil
	}
	var se *statusError
	if errors.As(err, &se) {
		switch {
		case se.status == http.StatusUnauthorized:
			return ErrBillingNotSignedIn
		case se.status == http.StatusServiceUnavailable && se.code == "billing_disabled", se.status == http.StatusNotFound:
			return ErrBillingOff // an older server has no route
		case se.status == http.StatusConflict && se.code == "already_subscribed":
			return ErrAlreadySubscribed
		case se.status == http.StatusTooManyRequests:
			return ErrCheckoutThrottled
		case se.status == http.StatusBadGateway && se.code == "billing_upstream":
			return ErrBillingUpstream
		}
		return err
	}
	if IsSignedOut(err) {
		return ErrBillingNotSignedIn
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return fmt.Errorf("%w: %v", ErrBillingNetwork, err)
}

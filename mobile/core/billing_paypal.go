package core

import (
	"context"
	"errors"

	"github.com/QingYunA/adhd-todo/internal/syncclient"
	"github.com/QingYunA/adhd-todo/internal/syncproto"
)

// The three places this package meets what M17 adds to the syncclient (`/v1/config` "paypal",
// `GET /v1/billing` "provider", `Checkout(ctx, plan, provider, lang)`). When the syncclient has
// them, change only these three bodies (each says its line); app/data/billing_paypal.go has the same
// three for the desktop.

// The providers Checkout takes, and Billing's "provider" says.
const (
	providerWaffo  = "waffo"
	providerPayPal = "paypal"
)

// errPayPalNotReady is what a PayPal checkout says until the syncclient can ask for one.
var errPayPalNotReady = errors.New("this build cannot ask the server for a PayPal checkout yet")

// configPayPal says /v1/config offers PayPal. Switch: `return cfg.PayPal`.
func configPayPal(syncproto.ConfigResponse) bool { return false }

// billProvider is the provider of the account's subscription. Switch: `return b.Provider`.
func billProvider(syncclient.Billing) string { return providerWaffo }

// checkoutAt asks for a checkout page at provider. Switch: `return c.Checkout(ctx, plan, provider, lang)`.
func checkoutAt(ctx context.Context, c *syncclient.Client, plan, provider, lang string) (string, error) {
	if provider == providerPayPal {
		return "", errPayPalNotReady
	}
	return c.Checkout(ctx, plan, lang)
}

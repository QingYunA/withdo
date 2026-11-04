package core

import (
	"context"

	"github.com/QingYunA/adhd-todo/internal/syncclient"
	"github.com/QingYunA/adhd-todo/internal/syncproto"
)

// The three places this package meets what M17 adds to the syncclient: /v1/config "paypal",
// GET /v1/billing "provider", and a checkout at a provider.

// The providers Checkout takes, and Billing's "provider" says.
const (
	providerWaffo  = "waffo"
	providerPayPal = "paypal"
)

// configPayPal says /v1/config offers PayPal.
func configPayPal(cfg syncproto.ConfigResponse) bool { return cfg.PayPal }

// billProvider is the provider of the account's subscription.
func billProvider(b syncclient.Billing) string { return b.Provider }

// checkoutAt asks for a checkout page at provider.
func checkoutAt(ctx context.Context, c *syncclient.Client, plan, provider, lang string) (string, error) {
	return c.Checkout(ctx, plan, provider, lang)
}

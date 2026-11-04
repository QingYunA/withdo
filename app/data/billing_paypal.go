package data

import (
	"context"
	"errors"

	"github.com/QingYunA/adhd-todo/internal/syncclient"
	"github.com/QingYunA/adhd-todo/internal/syncproto"
)

// The three places this package meets what M17 adds to the syncclient (`/v1/config` "paypal",
// `GET /v1/billing` "provider", `Checkout(ctx, plan, provider, lang)`). Everything else reads
// BillingInfo. When the syncclient has them, change only these three bodies (each says its line):
// the rest of the app, and its tests (which use FakeModels), stay as they are.

// errPayPalNotReady is what a PayPal checkout says until the syncclient can ask for one.
var errPayPalNotReady = errors.New("data: this build cannot ask the server for a PayPal checkout yet")

// configPayPal says /v1/config offers PayPal. Switch: `return cfg.PayPal`.
func configPayPal(syncproto.ConfigResponse) bool { return false }

// billProvider is the provider of the account's subscription. Switch: `return b.Provider`.
func billProvider(syncclient.Billing) string { return ProviderWaffo }

// checkoutAt asks for a checkout page at provider. Switch: `return c.Checkout(ctx, plan, provider, lang)`.
func checkoutAt(ctx context.Context, c *syncclient.Client, plan, provider, lang string) (string, error) {
	if provider == ProviderPayPal {
		return "", errPayPalNotReady
	}
	return c.Checkout(ctx, plan, lang)
}

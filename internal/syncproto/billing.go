package syncproto

// The payment channels a checkout can be made on (CheckoutRequest.Provider, BillingResponse.Provider).
const (
	ProviderWaffo  = "waffo"
	ProviderPayPal = "paypal"
)

// CheckoutRequest is POST /v1/billing/checkout. Plan is "monthly" or "yearly"; Lang is the app's
// language (a BCP 47 code such as "en" or "zh-Hans"), which only picks Waffo's checkout page's
// default language and may be empty. Provider is ProviderWaffo (the default when empty) or
// ProviderPayPal; any other value is 400 bad_provider, and PayPal on a server without it is
// 503 paypal_disabled (GET /v1/config says "paypal": false).
type CheckoutRequest struct {
	Plan     string `json:"plan"`
	Lang     string `json:"lang,omitempty"`
	Provider string `json:"provider,omitempty"`
}

// CheckoutResponse is the answer: the hosted checkout page to open in a browser (Waffo's, or the
// PayPal page where the payer approves the subscription).
type CheckoutResponse struct {
	URL string `json:"url"`
}

// BillingResponse is GET /v1/billing: the account's plan. Plan is "free" or "pro". Status is the
// subscription's (trialing, active, past_due, canceling, canceled) and empty for an account that
// never subscribed; Interval is the plan's billing interval, "monthly" or "yearly", empty when
// unknown (and for an account that never subscribed); PeriodEnd is RFC 3339 or empty. Monthly and Yearly are the prices to show.
// Provider is the channel the subscription is on, "waffo" or "paypal" (empty without a subscription).
// ManageURL opens Waffo's customer portal for a Waffo subscription, or the payer's automatic-payments
// page at PayPal for a PayPal one (empty without a subscription); a portal link is made fresh by
// every request, is good for 15 minutes of inactivity and must not be stored.
type BillingResponse struct {
	Plan      string `json:"plan"`
	Status    string `json:"status"`
	Interval  string `json:"interval"`
	PeriodEnd string `json:"period_end"`
	TrialUsed bool   `json:"trial_used"`
	Monthly   string `json:"monthly"`
	Yearly    string `json:"yearly"`
	ManageURL string `json:"manage_url"`
	Provider  string `json:"provider"`
}

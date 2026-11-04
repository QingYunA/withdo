package syncproto

// CheckoutRequest is POST /v1/billing/checkout. Plan is "monthly" or "yearly"; Lang is the app's
// language (a BCP 47 code such as "en" or "zh-Hans"), which only picks the checkout page's
// default language and may be empty.
type CheckoutRequest struct {
	Plan string `json:"plan"`
	Lang string `json:"lang,omitempty"`
}

// CheckoutResponse is the answer: the hosted checkout page to open in a browser.
type CheckoutResponse struct {
	URL string `json:"url"`
}

// BillingResponse is GET /v1/billing: the account's plan. Plan is "free" or "pro". Status is the
// subscription's (trialing, active, past_due, canceling, canceled) and empty for an account that
// never subscribed; PeriodEnd is RFC 3339 or empty. Monthly and Yearly are the prices to show.
// ManageURL opens Waffo's customer portal for the subscription (empty without one); it is made
// fresh by every request, is good for 15 minutes of inactivity and must not be stored.
type BillingResponse struct {
	Plan      string `json:"plan"`
	Status    string `json:"status"`
	PeriodEnd string `json:"period_end"`
	TrialUsed bool   `json:"trial_used"`
	Monthly   string `json:"monthly"`
	Yearly    string `json:"yearly"`
	ManageURL string `json:"manage_url"`
}

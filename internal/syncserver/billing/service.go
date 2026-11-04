// Package billing is Withdo Pro (M16): Waffo Pancake is the merchant of record, this package makes
// its checkout sessions, verifies its webhooks and keeps which accounts are Pro. The sync server
// owns the routes; this package knows nothing of HTTP handlers or sessions.
package billing

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// What the plans cost, for display; Waffo charges what the products say.
const (
	PriceMonthly = "$6.99"
	PriceYearly  = "$58.72"
)

// Plans a checkout can be made for.
const (
	PlanMonthly = "monthly"
	PlanYearly  = "yearly"
)

// DefaultSuccessURL is where Waffo's checkout sends the buyer after paying.
const DefaultSuccessURL = "https://withdo.si/pro/"

// PortalLoginURL is Waffo's customer portal sign-in (an emailed magic link), the fallback when a
// signed-in portal link cannot be made.
const PortalLoginURL = "https://pancake.waffo.ai/consumer/portal/login"

const (
	checkoutsPerHour = 10
	externalIDMax    = 128
)

var (
	// ErrUnknownPlan says the plan is not monthly or yearly.
	ErrUnknownPlan = errors.New("billing: unknown plan")
	// ErrAlreadySubscribed says the account is Pro already: it manages the subscription in the portal.
	ErrAlreadySubscribed = errors.New("billing: the account already has a subscription")
	// ErrRateLimited says the account asked for too many checkout sessions.
	ErrRateLimited = errors.New("billing: too many checkouts")
)

// Config configures a Service. The first six fields are the deployment's.
type Config struct {
	MerchantID     string
	Key            *rsa.PrivateKey // the API key's private key
	StoreID        string
	ProductMonthly string
	ProductYearly  string
	// Mode is "test" or "prod": webhooks of the other mode are acknowledged and ignored.
	Mode string

	// PayPal, when not nil, adds PayPal as a second channel (M17); Waffo stays the main one.
	PayPal *PayPalConfig

	// The rest is for tests and defaults.
	BaseURL     string                    // default DefaultBaseURL
	SuccessURL  string                    // default DefaultSuccessURL
	HTTP        *http.Client              // default one with no own timeout (each call has 15 s)
	WebhookKeys map[string]*rsa.PublicKey // default the embedded keys
	Now         func() time.Time          // default time.Now
}

// Service is Withdo Pro's billing. It is safe for concurrent use.
type Service struct {
	cfg      Config
	api      *api
	verify   verifier
	store    Store
	notify   func(account int64)
	products map[string]string
	paypal   *paypalAPI // nil when PayPal is off

	mu     sync.Mutex
	recent map[int64][]time.Time // checkout times per account, for the rate limit
}

// New checks cfg and returns a Service over store. notify, when not nil, is called with an
// account whose plan or period changed, so its devices can refresh.
func New(cfg Config, store Store, notify func(account int64)) (*Service, error) {
	switch {
	case store == nil:
		return nil, errors.New("billing: no store")
	case cfg.MerchantID == "" || cfg.Key == nil:
		return nil, errors.New("billing: a merchant id and a private key are required")
	case cfg.StoreID == "" || cfg.ProductMonthly == "" || cfg.ProductYearly == "":
		return nil, errors.New("billing: a store id and both product ids are required")
	}
	cfg.Mode = strings.ToLower(strings.TrimSpace(cfg.Mode))
	if cfg.Mode == "" {
		cfg.Mode = ModeTest
	}
	if cfg.Mode != ModeTest && cfg.Mode != ModeProd {
		return nil, fmt.Errorf("billing: mode must be %q or %q", ModeTest, ModeProd)
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	if cfg.SuccessURL == "" {
		cfg.SuccessURL = DefaultSuccessURL
	}
	if cfg.HTTP == nil {
		cfg.HTTP = &http.Client{}
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.WebhookKeys == nil {
		keys, err := embeddedWebhookKeys()
		if err != nil {
			return nil, err
		}
		cfg.WebhookKeys = keys
	}
	var pp *paypalAPI
	if cfg.PayPal != nil {
		var err error
		if pp, err = newPayPalAPI(*cfg.PayPal, cfg.HTTP, cfg.Now); err != nil {
			return nil, err
		}
	}
	return &Service{
		cfg:      cfg,
		paypal:   pp,
		api:      &api{base: cfg.BaseURL, merchantID: cfg.MerchantID, key: cfg.Key, http: cfg.HTTP, now: cfg.Now},
		verify:   verifier{keys: cfg.WebhookKeys, now: cfg.Now},
		store:    store,
		notify:   notify,
		products: map[string]string{PlanMonthly: cfg.ProductMonthly, PlanYearly: cfg.ProductYearly},
		recent:   map[int64][]time.Time{},
	}, nil
}

// Status is what GET /v1/billing tells an account.
type Status struct {
	Plan      string // "free" or "pro"
	Status    string // the subscription's status, empty for an account that never subscribed
	Interval  string // "monthly" or "yearly", empty when unknown
	PeriodEnd time.Time
	TrialUsed bool
	ManageURL string
	// Provider is ProviderWaffo or ProviderPayPal, empty for an account that never subscribed.
	Provider string
}

// Subscription reads an account's row.
func (s *Service) Subscription(ctx context.Context, account int64) (Subscription, bool, error) {
	sub, ok, err := s.store.Get(ctx, account)
	return sub, ok && s.thisMode(sub), err
}

// thisMode says whether a stored subscription belongs to the mode the server runs in. A test-mode
// order left over after going live (or a live one on a test server) is no subscription: it gives no
// Pro and uses no trial. For a PayPal subscription the mode is the sandbox ("test") or live
// ("prod"), and with PayPal off it belongs to no mode the server can vouch for.
func (s *Service) thisMode(sub Subscription) bool {
	want := s.cfg.Mode
	if sub.providerOrDefault() == ProviderPayPal {
		if s.paypal == nil {
			return false
		}
		want = paypalMode(s.paypal.cfg.Env)
	}
	return sub.Mode == "" || sub.Mode == want
}

// IsProSubject says whether the account a sign-in subject maps to is Pro now. A failed read is
// not Pro: the free limit is the safe answer.
func (s *Service) IsProSubject(ctx context.Context, subject string) bool {
	sub, ok, err := s.store.GetBySubject(ctx, subject)
	if err != nil {
		log.Printf("billing: plan lookup failed: %v", err)
		return false
	}
	return ok && s.thisMode(sub) && sub.Pro(s.cfg.Now())
}

// Status reads the account's plan. For an account that has a subscription it also makes a fresh
// customer-portal link for email (the buyer identity of its checkouts), falling back to the
// portal's sign-in page when the link cannot be made; accounts that never subscribed get none.
func (s *Service) Status(ctx context.Context, account int64, email func() string) (Status, error) {
	sub, ok, err := s.Subscription(ctx, account)
	if err != nil {
		return Status{}, err
	}
	st := Status{Plan: "free"}
	if !ok {
		return st, nil
	}
	now := s.cfg.Now()
	st.Status, st.Interval, st.PeriodEnd, st.TrialUsed, st.Provider = sub.Status, sub.Interval, sub.PeriodEnd, sub.TrialUsed, sub.providerOrDefault()
	if st.Status == StatusCanceling && !sub.PeriodEnd.IsZero() && !now.Before(sub.PeriodEnd) {
		st.Status = StatusCanceled // PayPal sends no event when a cancelled subscription's period ends
	}
	if sub.Pro(now) {
		st.Plan = "pro"
	}
	if st.Provider == ProviderPayPal {
		st.ManageURL = paypalManageURL(s.paypal.cfg.Env)
		return st, nil
	}
	st.ManageURL = PortalLoginURL
	if addr := email(); addr != "" {
		if link, err := s.api.portalLink(ctx, s.cfg.StoreID, addr); err == nil {
			st.ManageURL = link
		} else {
			log.Printf("billing: account=%d portal link failed: %v", account, err)
		}
	}
	return st, nil
}

// checkoutLanguage maps an app language to one Waffo's checkout page offers, or "" for its default.
func checkoutLanguage(lang string) string {
	tag := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(lang), "_", "-"))
	switch {
	case tag == "zh-hant" || strings.HasPrefix(tag, "zh-tw") || strings.HasPrefix(tag, "zh-hk"):
		return "zh-Hant"
	case tag == "zh" || strings.HasPrefix(tag, "zh-"):
		return "zh-Hans"
	}
	base, _, _ := strings.Cut(tag, "-")
	switch base {
	case "en":
		return "en"
	case "ja":
		return "ja-JP"
	case "ko":
		return "ko-KR"
	case "es":
		return "es-MX"
	case "pt":
		return "pt-BR"
	}
	return ""
}

// checkoutGate is what both channels check before making a checkout for the account: it is not Pro
// already and has not asked for too many checkouts. exists and sub are its row (this mode's).
func (s *Service) checkoutGate(ctx context.Context, account int64) (sub Subscription, exists bool, now time.Time, err error) {
	sub, exists, err = s.Subscription(ctx, account)
	if err != nil {
		return sub, false, now, err
	}
	now = s.cfg.Now()
	if exists && sub.Pro(now) {
		return sub, exists, now, ErrAlreadySubscribed
	}
	if !s.allowCheckout(account, now) {
		return sub, exists, now, ErrRateLimited
	}
	return sub, exists, now, nil
}

// Checkout makes a Waffo checkout session for the account and returns its URL. The account's email
// pre-fills the page; the trial is offered once per account.
func (s *Service) Checkout(ctx context.Context, account int64, email, plan, language string) (string, error) {
	product, ok := s.products[plan]
	if !ok {
		return "", ErrUnknownPlan
	}
	sub, exists, now, err := s.checkoutGate(ctx, account)
	if err != nil {
		return "", err
	}
	external := "withdo-" + strconv.FormatInt(account, 10) + "-" + strconv.FormatInt(now.Unix(), 10)
	if len(external) > externalIDMax {
		external = external[:externalIDMax]
	}
	req := checkoutRequest{
		ProductID:               product,
		Currency:                "USD",
		BuyerEmail:              email,
		Metadata:                map[string]string{"account": strconv.FormatInt(account, 10)},
		OrderMerchantExternalID: external,
		WithTrial:               !(exists && sub.TrialUsed),
		SuccessURL:              s.cfg.SuccessURL,
		Language:                checkoutLanguage(language),
	}
	url, err := s.api.createCheckout(ctx, req)
	var refused *upstreamError
	if errors.As(err, &refused) && refused.status == http.StatusBadRequest && req.Language != "" {
		// The language list is Waffo's; the page's default is not worth a failed checkout.
		req.Language = ""
		url, err = s.api.createCheckout(ctx, req)
	}
	if err != nil {
		return "", err
	}
	log.Printf("billing: account=%d plan=%s checkout=created", account, plan)
	return url, nil
}

// PayPalCheckout creates a PayPal subscription for the account and returns the PayPal page the
// payer approves it at. The 7-day trial plan is used unless the account's row says it had a trial,
// on either channel. The subscription's custom_id is the account: that is how its webhooks find it.
func (s *Service) PayPalCheckout(ctx context.Context, account int64, email, plan string) (string, error) {
	if s.paypal == nil {
		return "", ErrPayPalOff
	}
	if s.paypal.planID(plan, false) == "" {
		return "", ErrUnknownPlan
	}
	sub, exists, _, err := s.checkoutGate(ctx, account)
	if err != nil {
		return "", err
	}
	_, approve, err := s.paypal.createSubscription(ctx, s.paypal.planID(plan, !(exists && sub.TrialUsed)),
		strconv.FormatInt(account, 10), email)
	if err != nil {
		return "", err
	}
	log.Printf("billing: account=%d plan=%s provider=paypal checkout=created", account, plan)
	return approve, nil
}

// allowCheckout counts a checkout against the account's hourly allowance.
func (s *Service) allowCheckout(account int64, now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	cutoff := now.Add(-time.Hour)
	kept := s.recent[account][:0]
	for _, at := range s.recent[account] {
		if at.After(cutoff) {
			kept = append(kept, at)
		}
	}
	if len(kept) >= checkoutsPerHour {
		s.recent[account] = kept
		return false
	}
	s.recent[account] = append(kept, now)
	return true
}

// Outcome says what a webhook did; it is what the log line carries.
type Outcome string

// The outcomes of HandleWebhook.
const (
	OutcomeApplied   Outcome = "applied"
	OutcomeDuplicate Outcome = "duplicate"
	OutcomeIgnored   Outcome = "ignored"
)

// HandleWebhook verifies a Waffo webhook (header is X-Waffo-Signature, body the raw bytes) and
// applies it. Verification failures are ErrBadSignature and ErrStaleSignature; any other error is
// ours (the database), and the caller answers 5xx so Waffo sends the event again. Events that are
// not about a Withdo account, or are of the other mode or store, are ignored, not errors.
func (s *Service) HandleWebhook(ctx context.Context, header string, body []byte) (Event, Outcome, error) {
	ev, err := s.verify.verify(header, body)
	if err != nil {
		return ev, "", err
	}
	if ev.Mode != s.cfg.Mode || (ev.StoreID != "" && ev.StoreID != s.cfg.StoreID) {
		return ev, OutcomeIgnored, nil
	}
	account, ok := ev.Data.Account()
	key := ev.dedupeKey()
	if !ok || key == "" {
		return ev, OutcomeIgnored, nil
	}
	before, after, dup, saved, err := s.store.Apply(ctx, key, ev.Type, account,
		func(cur Subscription, exists bool) (Subscription, bool) { return reduce(cur, exists, ev, account) })
	switch {
	case errors.Is(err, ErrNoAccount):
		return ev, OutcomeIgnored, nil
	case err != nil:
		return ev, "", err
	case dup:
		return ev, OutcomeDuplicate, nil
	case !saved:
		return ev, OutcomeIgnored, nil
	}
	now := s.cfg.Now()
	if s.notify != nil && (before.Status != after.Status || !before.PeriodEnd.Equal(after.PeriodEnd) ||
		before.Pro(now) != after.Pro(now)) {
		s.notify(account)
	}
	return ev, OutcomeApplied, nil
}

package billing

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// PayPal environments, as PAYPAL_ENV spells them.
const (
	PayPalSandbox = "sandbox"
	PayPalLive    = "live"
)

const (
	paypalSandboxBase = "https://api-m.sandbox.paypal.com"
	paypalLiveBase    = "https://api-m.paypal.com"
	// paypalTokenSlack is how long before its expiry a cached access token is renewed.
	paypalTokenSlack = time.Minute
)

// ErrPayPalOff says PayPal is not configured on this server.
var ErrPayPalOff = errors.New("billing: PayPal is off")

// PayPalConfig configures PayPal as a second channel (M17). Without one the Service sells through
// Waffo only.
type PayPalConfig struct {
	ClientID, Secret string
	// Env is PayPalSandbox (default) or PayPalLive.
	Env       string
	WebhookID string
	// The four billing plans (docs/plans/m17-paypal.md): with and without the 7-day trial.
	PlanMonthly, PlanMonthlyTrial, PlanYearly, PlanYearlyTrial string

	// For tests and defaults.
	BaseURL   string // default the environment's API root
	ReturnURL string // default DefaultSuccessURL; also where cancelling the approval returns
}

// paypalMode is the stored mode of a PayPal row: the sandbox is "test", live is "prod", the words
// the rows of Waffo use, so that the other-mode rule reads the same.
func paypalMode(env string) string {
	if env == PayPalLive {
		return ModeProd
	}
	return ModeTest
}

// paypalManageURL is the payer's automatic-payments page, where a PayPal subscription is cancelled.
func paypalManageURL(env string) string {
	if env == PayPalLive {
		return "https://www.paypal.com/myaccount/autopay/"
	}
	return "https://www.sandbox.paypal.com/myaccount/autopay/"
}

// paypalAPI makes the calls Withdo makes to PayPal's REST API, with a cached OAuth token.
type paypalAPI struct {
	cfg  PayPalConfig
	http *http.Client
	now  func() time.Time

	mu      sync.Mutex
	token   string
	expires time.Time
}

func newPayPalAPI(cfg PayPalConfig, client *http.Client, now func() time.Time) (*paypalAPI, error) {
	cfg.Env = strings.ToLower(strings.TrimSpace(cfg.Env))
	if cfg.Env == "" {
		cfg.Env = PayPalSandbox
	}
	if cfg.Env != PayPalSandbox && cfg.Env != PayPalLive {
		return nil, fmt.Errorf("billing: PayPal env must be %q or %q", PayPalSandbox, PayPalLive)
	}
	for name, value := range map[string]string{
		"client id": cfg.ClientID, "secret": cfg.Secret, "webhook id": cfg.WebhookID,
		"monthly plan": cfg.PlanMonthly, "monthly trial plan": cfg.PlanMonthlyTrial,
		"yearly plan": cfg.PlanYearly, "yearly trial plan": cfg.PlanYearlyTrial,
	} {
		if strings.TrimSpace(value) == "" {
			return nil, fmt.Errorf("billing: PayPal %s is required", name)
		}
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = paypalSandboxBase
		if cfg.Env == PayPalLive {
			cfg.BaseURL = paypalLiveBase
		}
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	if cfg.ReturnURL == "" {
		cfg.ReturnURL = DefaultSuccessURL
	}
	return &paypalAPI{cfg: cfg, http: client, now: now}, nil
}

// planID is the PayPal plan for a Withdo plan, with or without the trial; "" for an unknown plan.
func (a *paypalAPI) planID(plan string, trial bool) string {
	switch {
	case plan == PlanMonthly && trial:
		return a.cfg.PlanMonthlyTrial
	case plan == PlanMonthly:
		return a.cfg.PlanMonthly
	case plan == PlanYearly && trial:
		return a.cfg.PlanYearlyTrial
	case plan == PlanYearly:
		return a.cfg.PlanYearly
	}
	return ""
}

// intervalOf is the interval ("monthly", "yearly") of a PayPal plan id, "" when it is none of ours.
func (a *paypalAPI) intervalOf(planID string) string {
	switch planID {
	case "":
		return ""
	case a.cfg.PlanMonthly, a.cfg.PlanMonthlyTrial:
		return "monthly"
	case a.cfg.PlanYearly, a.cfg.PlanYearlyTrial:
		return "yearly"
	}
	return ""
}

// accessToken returns the cached client-credentials token, fetching a new one when it is missing
// or about to expire.
func (a *paypalAPI) accessToken(ctx context.Context) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.token != "" && a.now().Add(paypalTokenSlack).Before(a.expires) {
		return a.token, nil
	}
	ctx, cancel := context.WithTimeout(ctx, apiTimeout)
	defer cancel()
	form := url.Values{"grant_type": {"client_credentials"}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.cfg.BaseURL+"/v1/oauth2/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(a.cfg.ClientID, a.cfg.Secret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	raw, status, err := a.send(ctx, req)
	if err != nil {
		return "", err
	}
	if status/100 != 2 {
		return "", paypalError(status, raw)
	}
	var out struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(raw, &out); err != nil || out.AccessToken == "" {
		return "", fmt.Errorf("%w: unreadable token", ErrUpstream)
	}
	a.token = out.AccessToken
	a.expires = a.now().Add(time.Duration(out.ExpiresIn) * time.Second)
	return a.token, nil
}

func (a *paypalAPI) dropToken() {
	a.mu.Lock()
	a.token = ""
	a.mu.Unlock()
}

// send performs req and returns the (bounded) body and status. A transport failure is an
// ErrUpstream that carries no URL or credentials.
func (a *paypalAPI) send(ctx context.Context, req *http.Request) ([]byte, int, error) {
	resp, err := a.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, 0, fmt.Errorf("%w: %v", ErrUpstream, ctx.Err())
		}
		return nil, 0, fmt.Errorf("%w: unreachable", ErrUpstream)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, apiBodyMax))
	return raw, resp.StatusCode, nil
}

// paypalError is the ErrUpstream for a PayPal answer that is not 2xx. It carries PayPal's error
// name and debug id (what its support asks for), never the message, which can echo the request.
func paypalError(status int, raw []byte) error {
	var e struct {
		Name    string `json:"name"`
		DebugID string `json:"debug_id"`
	}
	_ = json.Unmarshal(raw, &e)
	msg := "no name"
	if e.Name != "" {
		msg = e.Name
		if e.DebugID != "" {
			msg += " debug_id=" + e.DebugID
		}
	}
	return &upstreamError{status: status, message: msg}
}

// call sends one authorised JSON request and decodes a 2xx answer into out (nil to ignore it). A
// 401 means the cached token was revoked: it is dropped and the call made once more.
func (a *paypalAPI) call(ctx context.Context, method, path string, in, out any) (int, error) {
	var body []byte
	switch v := in.(type) {
	case nil:
	case []byte: // already encoded
		body = v
	default:
		var err error
		if body, err = json.Marshal(in); err != nil {
			return 0, fmt.Errorf("billing: encode the request: %w", err)
		}
	}
	for attempt := 0; ; attempt++ {
		token, err := a.accessToken(ctx)
		if err != nil {
			return 0, err
		}
		cctx, cancel := context.WithTimeout(ctx, apiTimeout)
		req, err := http.NewRequestWithContext(cctx, method, a.cfg.BaseURL+path, bytes.NewReader(body))
		if err != nil {
			cancel()
			return 0, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/json")
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		raw, status, err := a.send(cctx, req)
		cancel()
		if err != nil {
			return 0, err
		}
		if status == http.StatusUnauthorized && attempt == 0 {
			a.dropToken()
			continue
		}
		if status/100 != 2 {
			return status, paypalError(status, raw)
		}
		if out != nil {
			if err := json.Unmarshal(raw, out); err != nil {
				return status, fmt.Errorf("%w: unreadable answer", ErrUpstream)
			}
		}
		return status, nil
	}
}

type paypalLink struct {
	Href string `json:"href"`
	Rel  string `json:"rel"`
}

// approveLink picks the link the payer approves the subscription at, by its rel.
func approveLink(links []paypalLink) string {
	for _, l := range links {
		if l.Rel == "approve" && l.Href != "" {
			return l.Href
		}
	}
	return ""
}

// createSubscription creates a subscription to planID for the account and returns its id and the
// PayPal page the payer approves it at. The subscription starts when the payer approves.
func (a *paypalAPI) createSubscription(ctx context.Context, planID, customID, email string) (id, approveURL string, err error) {
	subscriber := map[string]string{}
	if email != "" {
		subscriber["email_address"] = email
	}
	in := map[string]any{
		"plan_id":    planID,
		"custom_id":  customID,
		"subscriber": subscriber,
		"application_context": map[string]string{
			"brand_name":          "Withdo",
			"shipping_preference": "NO_SHIPPING",
			"user_action":         "SUBSCRIBE_NOW",
			"return_url":          a.cfg.ReturnURL,
			"cancel_url":          a.cfg.ReturnURL,
		},
	}
	if email == "" {
		delete(in, "subscriber")
	}
	var out struct {
		ID    string       `json:"id"`
		Links []paypalLink `json:"links"`
	}
	if _, err := a.call(ctx, http.MethodPost, "/v1/billing/subscriptions", in, &out); err != nil {
		return "", "", err
	}
	approve := approveLink(out.Links)
	if out.ID == "" || approve == "" {
		return "", "", fmt.Errorf("%w: no approval link", ErrUpstream)
	}
	return out.ID, approve, nil
}

// paypalSub is the part of PayPal's subscription resource the entitlement rule reads.
type paypalSub struct {
	ID               string   `json:"id"`
	Status           string   `json:"status"`
	PlanID           string   `json:"plan_id"`
	CustomID         string   `json:"custom_id"`
	StartTime        flexTime `json:"start_time"`
	StatusUpdateTime flexTime `json:"status_update_time"`
	UpdateTime       flexTime `json:"update_time"`
	BillingInfo      struct {
		CycleExecutions []paypalCycle `json:"cycle_executions"`
		LastPayment     struct {
			Amount struct {
				Value string `json:"value"`
			} `json:"amount"`
			Time flexTime `json:"time"`
		} `json:"last_payment"`
		NextBillingTime     flexTime `json:"next_billing_time"`
		FailedPaymentsCount int      `json:"failed_payments_count"`
	} `json:"billing_info"`
}

type paypalCycle struct {
	TenureType      string `json:"tenure_type"` // TRIAL or REGULAR
	CyclesCompleted int    `json:"cycles_completed"`
	TotalCycles     int    `json:"total_cycles"`
}

// getSubscription reads a subscription from PayPal. found is false for an id PayPal does not know
// (404), which is no error: a sandbox id on a live server, say.
func (a *paypalAPI) getSubscription(ctx context.Context, id string) (sub paypalSub, found bool, err error) {
	status, err := a.call(ctx, http.MethodGet, "/v1/billing/subscriptions/"+url.PathEscape(id), nil, &sub)
	var refused *upstreamError
	if errors.As(err, &refused) && status == http.StatusNotFound {
		return paypalSub{}, false, nil
	}
	if err != nil {
		return paypalSub{}, false, err
	}
	return sub, true, nil
}

// PayPalHeaders are the five PAYPAL-* headers of a webhook delivery.
type PayPalHeaders struct {
	AuthAlgo, CertURL, TransmissionID, TransmissionSig, TransmissionTime string
}

func (h PayPalHeaders) complete() bool {
	return h.AuthAlgo != "" && h.CertURL != "" && h.TransmissionID != "" && h.TransmissionSig != "" && h.TransmissionTime != ""
}

// verifyWebhook asks PayPal whether a delivery is genuine (POST /v1/notifications/verify-webhook-signature)
// with the raw event body. Only "SUCCESS" is a yes; a 400 (a certificate URL or algorithm PayPal does
// not accept) is a no too. Anything else is an error: the caller asks PayPal to deliver again.
func (a *paypalAPI) verifyWebhook(ctx context.Context, h PayPalHeaders, event []byte) (bool, error) {
	// The event goes in byte for byte: PayPal checks the signature over its exact text, so it must
	// not pass through json.Marshal, which would compact it and escape <, > and &.
	head, err := json.Marshal(map[string]string{
		"auth_algo": h.AuthAlgo, "cert_url": h.CertURL, "transmission_id": h.TransmissionID,
		"transmission_sig": h.TransmissionSig, "transmission_time": h.TransmissionTime, "webhook_id": a.cfg.WebhookID,
	})
	if err != nil {
		return false, err
	}
	in := append(append(head[:len(head)-1:len(head)-1], []byte(`,"webhook_event":`)...), event...)
	in = append(in, '}')
	var out struct {
		VerificationStatus string `json:"verification_status"`
	}
	status, err := a.call(ctx, http.MethodPost, "/v1/notifications/verify-webhook-signature", in, &out)
	if status == http.StatusBadRequest {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return out.VerificationStatus == "SUCCESS", nil
}

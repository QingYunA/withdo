package billing

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	ppClient  = "client-id"
	ppSecret  = "client-secret"
	ppWebhook = "WH-1"
)

// fakePayPal is PayPal's REST API: OAuth, create and show subscription, verify-webhook-signature.
type fakePayPal struct {
	*httptest.Server
	mu sync.Mutex
	// tokens counts the token requests; expiresIn is what they answer.
	tokens    int
	expiresIn int
	// revoke, when set, makes the next authorised request answer 401 once.
	revoke bool
	subs   map[string]string // subscription id -> JSON the show call answers
	// links is the "links" array of a created subscription.
	links         string
	created       []map[string]any
	verifications []map[string]json.RawMessage
	getCalls      map[string]int
	// verifyStatus is the answer of verify-webhook-signature: HTTP status and verification_status.
	verifyHTTP   int
	verifyResult string
	// showStatus overrides the HTTP status of the show call (0 is the stored JSON or a 404).
	showStatus int
}

func newFakePayPal(t *testing.T) *fakePayPal {
	t.Helper()
	f := &fakePayPal{
		expiresIn: 32400, subs: map[string]string{}, getCalls: map[string]int{},
		links:      `[{"href":"https://api.paypal.test/v1/billing/subscriptions/I-NEW","rel":"self","method":"GET"},{"href":"https://paypal.test/webapps/billing/subscriptions?ba_token=BA-1","rel":"approve","method":"GET"},{"href":"x","rel":"edit","method":"PATCH"}]`,
		verifyHTTP: 200, verifyResult: "SUCCESS",
	}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		f.mu.Lock()
		defer f.mu.Unlock()
		if r.URL.Path == "/v1/oauth2/token" {
			user, pass, ok := r.BasicAuth()
			if !ok || user != ppClient || pass != ppSecret || string(raw) != "grant_type=client_credentials" {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = io.WriteString(w, `{"error":"invalid_client"}`)
				return
			}
			f.tokens++
			_, _ = fmt.Fprintf(w, `{"access_token":"tok-%d","token_type":"Bearer","expires_in":%d}`, f.tokens, f.expiresIn)
			return
		}
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer tok-") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if f.revoke {
			f.revoke = false
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/billing/subscriptions":
			var body map[string]any
			_ = json.Unmarshal(raw, &body)
			f.created = append(f.created, body)
			w.WriteHeader(http.StatusCreated)
			_, _ = fmt.Fprintf(w, `{"id":"I-NEW","status":"APPROVAL_PENDING","links":%s}`, f.links)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v1/billing/subscriptions/"):
			id := strings.TrimPrefix(r.URL.Path, "/v1/billing/subscriptions/")
			f.getCalls[id]++
			if f.showStatus != 0 {
				w.WriteHeader(f.showStatus)
				_, _ = io.WriteString(w, `{"name":"INTERNAL_SERVER_ERROR","debug_id":"d1"}`)
				return
			}
			body, ok := f.subs[id]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				_, _ = io.WriteString(w, `{"name":"RESOURCE_NOT_FOUND","debug_id":"d2"}`)
				return
			}
			_, _ = io.WriteString(w, body)
		case r.Method == http.MethodPost && r.URL.Path == "/v1/notifications/verify-webhook-signature":
			var body map[string]json.RawMessage
			_ = json.Unmarshal(raw, &body)
			// The event is kept as sent, to compare bytes.
			body["_raw"] = raw
			f.verifications = append(f.verifications, body)
			w.WriteHeader(f.verifyHTTP)
			_, _ = fmt.Fprintf(w, `{"verification_status":%q}`, f.verifyResult)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(f.Close)
	return f
}

func (f *fakePayPal) setSub(id, body string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.subs[id] = body
}

func (f *fakePayPal) gets(id string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.getCalls[id]
}

func (f *fakePayPal) verifyCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.verifications)
}

const (
	ppMonthly, ppMonthlyTrial = "P-M", "P-MT"
	ppYearly, ppYearlyTrial   = "P-Y", "P-YT"
)

// ppService is a Service with Waffo and PayPal over fakes.
type ppService struct {
	*testService
	paypal *fakePayPal
}

func newPayPalService(t *testing.T, env string, accounts ...int64) *ppService {
	t.Helper()
	apiKey, webhookKey := genKey(t), genKey(t)
	ts := &testService{store: newMemStore(accounts...), waffo: newFakeWaffo(t, apiKey), webhook: webhookKey, now: testNow}
	fake := newFakePayPal(t)
	svc, err := New(Config{
		MerchantID: "MER_test", Key: apiKey, StoreID: "STO_withdo",
		ProductMonthly: "PROD_monthly", ProductYearly: "PROD_yearly", Mode: ModeTest,
		BaseURL:     ts.waffo.URL,
		WebhookKeys: map[string]*rsa.PublicKey{ModeTest: &webhookKey.PublicKey, ModeProd: &genKey(t).PublicKey},
		Now:         func() time.Time { ts.nowMu.Lock(); defer ts.nowMu.Unlock(); return ts.now },
		PayPal: &PayPalConfig{
			ClientID: ppClient, Secret: ppSecret, Env: env, WebhookID: ppWebhook, BaseURL: fake.URL,
			PlanMonthly: ppMonthly, PlanMonthlyTrial: ppMonthlyTrial, PlanYearly: ppYearly, PlanYearlyTrial: ppYearlyTrial,
		},
	}, ts.store, func(account int64) {
		ts.notifyMu.Lock()
		defer ts.notifyMu.Unlock()
		ts.notified = append(ts.notified, account)
	})
	if err != nil {
		t.Fatal(err)
	}
	ts.Service = svc
	return &ppService{testService: ts, paypal: fake}
}

var ppHeaders = PayPalHeaders{
	AuthAlgo: "SHA256withRSA", CertURL: "https://api.paypal.com/v1/notifications/certs/CERT-1",
	TransmissionID: "tx-1", TransmissionSig: "sig", TransmissionTime: "2026-11-04T12:00:00Z",
}

// event is a PayPal webhook body.
func ppEvent(id, eventType string, resource string) []byte {
	return []byte(fmt.Sprintf(`{"id":%q,"event_type":%q,"create_time":"2026-11-04T12:00:00Z","resource":%s}`, id, eventType, resource))
}

func (p *ppService) deliver(id, eventType, subID string) (PayPalEvent, Outcome, error) {
	resource := fmt.Sprintf(`{"id":%q}`, subID)
	if strings.HasPrefix(eventType, "PAYMENT.SALE.") { // a sale names its subscription as billing_agreement_id
		resource = fmt.Sprintf(`{"id":"SALE-1","billing_agreement_id":%q}`, subID)
	}
	return p.HandlePayPalWebhook(context.Background(), ppHeaders, ppEvent(id, eventType, resource))
}

type ppCycle struct {
	trial                 bool
	completed, total, rem int
}

// ppSubJSON is PayPal's show-subscription answer.
type ppSubOpts struct {
	id, status, plan, customID string
	updated                    time.Time
	nextBilling                time.Time
	lastPayment                time.Time
	lastAmount                 string
	cycles                     []ppCycle
	failed                     int
}

func (o ppSubOpts) json() string {
	ts := func(t time.Time) string {
		if t.IsZero() {
			return ""
		}
		return t.UTC().Format(time.RFC3339)
	}
	var cycles []string
	for i, c := range o.cycles {
		tenure := "REGULAR"
		if c.trial {
			tenure = "TRIAL"
		}
		cycles = append(cycles, fmt.Sprintf(`{"tenure_type":%q,"sequence":%d,"cycles_completed":%d,"cycles_remaining":%d,"total_cycles":%d}`,
			tenure, i+1, c.completed, c.rem, c.total))
	}
	billing := fmt.Sprintf(`{"outstanding_balance":{"currency_code":"USD","value":"0.0"},"cycle_executions":[%s],"failed_payments_count":%d`,
		strings.Join(cycles, ","), o.failed)
	if !o.lastPayment.IsZero() {
		billing += fmt.Sprintf(`,"last_payment":{"amount":{"currency_code":"USD","value":%q},"time":%q}`, o.lastAmount, ts(o.lastPayment))
	}
	if !o.nextBilling.IsZero() {
		billing += fmt.Sprintf(`,"next_billing_time":%q`, ts(o.nextBilling))
	}
	billing += "}"
	custom := ""
	if o.customID != "" {
		custom = fmt.Sprintf(`"custom_id":%q,`, o.customID)
	}
	return fmt.Sprintf(`{"id":%q,"status":%q,"status_update_time":%q,"update_time":%q,"plan_id":%q,%s"start_time":%q,"subscriber":{"email_address":"someone@example.com"},"billing_info":%s}`,
		o.id, o.status, ts(o.updated), ts(o.updated), o.plan, custom, ts(o.updated), billing)
}

// A subscription in its trial, paid and active, as PayPal reports them.
func ppTrialing(id, custom string, at time.Time) ppSubOpts {
	return ppSubOpts{
		id: id, status: "ACTIVE", plan: ppMonthlyTrial, customID: custom, updated: at,
		nextBilling: at.Add(7 * 24 * time.Hour),
		cycles:      []ppCycle{{trial: true, completed: 0, total: 1, rem: 1}, {completed: 0, total: 0}},
	}
}

func ppPaid(id, custom string, at time.Time) ppSubOpts {
	return ppSubOpts{
		id: id, status: "ACTIVE", plan: ppMonthlyTrial, customID: custom, updated: at,
		nextBilling: at.AddDate(0, 1, 0), lastPayment: at, lastAmount: "6.99",
		cycles: []ppCycle{{trial: true, completed: 1, total: 1}, {completed: 1, total: 0}},
	}
}

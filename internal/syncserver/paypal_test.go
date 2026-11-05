package syncserver_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/syncproto"
	"github.com/QingYunA/adhd-todo/internal/syncserver"
	"github.com/QingYunA/adhd-todo/internal/syncserver/billing"
)

// fakePayPalAPI is the part of PayPal's REST API the server calls.
type fakePayPalAPI struct {
	*httptest.Server
	mu      sync.Mutex
	created []map[string]any
	sub     string // the JSON GET /v1/billing/subscriptions/I-1 answers
	genuine bool
}

func newFakePayPalAPI(t *testing.T) *fakePayPalAPI {
	t.Helper()
	f := &fakePayPalAPI{genuine: true}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		f.mu.Lock()
		defer f.mu.Unlock()
		switch {
		case r.URL.Path == "/v1/oauth2/token":
			_, _ = io.WriteString(w, `{"access_token":"tok","expires_in":3600}`)
		case r.Method == http.MethodPost && r.URL.Path == "/v1/billing/subscriptions":
			var body map[string]any
			_ = json.Unmarshal(raw, &body)
			f.created = append(f.created, body)
			_, _ = io.WriteString(w, `{"id":"I-1","status":"APPROVAL_PENDING","links":[{"href":"https://paypal.test/approve/I-1","rel":"approve"}]}`)
		case r.Method == http.MethodGet && r.URL.Path == "/v1/billing/subscriptions/I-1":
			_, _ = io.WriteString(w, f.sub)
		case r.URL.Path == "/v1/notifications/verify-webhook-signature":
			status := "SUCCESS"
			if !f.genuine {
				status = "FAILURE"
			}
			_, _ = fmt.Fprintf(w, `{"verification_status":%q}`, status)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(f.Close)
	return f
}

func (f *fakePayPalAPI) setSub(status string, custom string, updated time.Time, extra string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	at := updated.UTC().Format(time.RFC3339)
	f.sub = fmt.Sprintf(`{"id":"I-1","status":%q,"plan_id":"P-MT","custom_id":%q,"update_time":%q,"status_update_time":%q,"billing_info":{%s}}`,
		status, custom, at, at, extra)
}

func paypalConfig(api *fakePayPalAPI) *billing.PayPalConfig {
	return &billing.PayPalConfig{
		ClientID: "c", Secret: "s", Env: billing.PayPalSandbox, WebhookID: "WH-1", BaseURL: api.URL,
		PlanMonthly: "P-M", PlanMonthlyTrial: "P-MT", PlanYearly: "P-Y", PlanYearlyTrial: "P-YT",
	}
}

func (r *billingRig) sendPayPal(t *testing.T, id, eventType string, headers bool) int {
	t.Helper()
	body := fmt.Sprintf(`{"id":%q,"event_type":%q,"resource":{"id":"I-1"}}`, id, eventType)
	req, _ := http.NewRequest(http.MethodPost, r.http.URL+"/v1/billing/paypal", bytes.NewReader([]byte(body)))
	if headers {
		req.Header.Set("PAYPAL-AUTH-ALGO", "SHA256withRSA")
		req.Header.Set("PAYPAL-CERT-URL", "https://api.paypal.com/v1/notifications/certs/CERT-1")
		req.Header.Set("PAYPAL-TRANSMISSION-ID", "tx")
		req.Header.Set("PAYPAL-TRANSMISSION-SIG", "sig")
		req.Header.Set("PAYPAL-TRANSMISSION-TIME", time.Now().UTC().Format(time.RFC3339))
	}
	resp, err := r.http.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	return resp.StatusCode
}

func TestPayPalIsOffWithoutCredentials(t *testing.T) {
	rig := newBillingRig(t)
	var cfg syncproto.ConfigResponse
	rig.get(t, "/v1/config", "", &cfg)
	if !cfg.Billing || cfg.PayPal {
		t.Fatalf("config = %+v", cfg)
	}
	resp, body := rig.do(t, http.MethodPost, "/v1/billing/checkout", aliceToken, syncproto.CheckoutRequest{Plan: "monthly", Provider: "paypal"})
	var out struct {
		Code string `json:"code"`
	}
	if resp.StatusCode != http.StatusServiceUnavailable || json.Unmarshal(body, &out) != nil || out.Code != syncserver.CodePayPalDisabled {
		t.Fatalf("paypal checkout = %d %s", resp.StatusCode, body)
	}
	if code := rig.sendPayPal(t, "WH-x", "BILLING.SUBSCRIPTION.ACTIVATED", true); code != http.StatusServiceUnavailable {
		t.Fatalf("paypal webhook = %d", code)
	}
	// Waffo still sells, by default and by name.
	requireStatus(t, "default", rig.post(t, "/v1/billing/checkout", aliceToken, syncproto.CheckoutRequest{Plan: "monthly"}, nil), http.StatusOK)
	requireStatus(t, "waffo", rig.post(t, "/v1/billing/checkout", aliceToken, syncproto.CheckoutRequest{Plan: "monthly", Provider: "waffo"}, nil), http.StatusOK)
	requireStatus(t, "bad provider", rig.post(t, "/v1/billing/checkout", aliceToken, syncproto.CheckoutRequest{Plan: "monthly", Provider: "stripe"}, nil), http.StatusBadRequest)
}

func TestPayPalCheckoutAndWebhook(t *testing.T) {
	api := newFakePayPalAPI(t)
	rig := newBillingRigWith(t, paypalConfig(api))
	var cfg syncproto.ConfigResponse
	rig.get(t, "/v1/config", "", &cfg)
	if !cfg.Billing || !cfg.PayPal {
		t.Fatalf("config = %+v", cfg)
	}
	device := rig.device(t, aliceToken, "phone")
	stream := rig.stream(t, aliceToken, device.DeviceID)
	stream.waitReady(t)

	requireStatus(t, "unknown plan", rig.post(t, "/v1/billing/checkout", aliceToken, syncproto.CheckoutRequest{Plan: "weekly", Provider: "paypal"}, nil), http.StatusBadRequest)
	requireStatus(t, "bad provider", rig.post(t, "/v1/billing/checkout", aliceToken, syncproto.CheckoutRequest{Plan: "monthly", Provider: "stripe"}, nil), http.StatusBadRequest)
	requireStatus(t, "no token", rig.post(t, "/v1/billing/checkout", "", syncproto.CheckoutRequest{Plan: "monthly", Provider: "paypal"}, nil), http.StatusUnauthorized)

	var out syncproto.CheckoutResponse
	requireStatus(t, "paypal checkout", rig.post(t, "/v1/billing/checkout", aliceToken, syncproto.CheckoutRequest{Plan: "monthly", Provider: "paypal"}, &out), http.StatusOK)
	if out.URL != "https://paypal.test/approve/I-1" {
		t.Fatalf("url = %q", out.URL)
	}
	created := api.created[0]
	account, _ := created["custom_id"].(string)
	sub, _ := created["subscriber"].(map[string]any)
	if created["plan_id"] != "P-MT" || account == "" || sub["email_address"] != aliceSubject+"@example.com" {
		t.Fatalf("created = %v", created)
	}

	quota := func(token string) (limit int, plan string) {
		var q struct {
			Limit int    `json:"limit"`
			Plan  string `json:"plan"`
		}
		requireStatus(t, "quota", rig.get(t, "/v1/llm/quota", token, &q), http.StatusOK)
		return q.Limit, q.Plan
	}
	if limit, plan := quota(aliceToken); limit != 2 || plan != "free" {
		t.Fatalf("before = %d %s", limit, plan)
	}

	// No PayPal headers, or PayPal not vouching for the delivery: nothing happens.
	now := time.Now()
	api.setSub("ACTIVE", account, now, `"cycle_executions":[{"tenure_type":"TRIAL","cycles_completed":0,"total_cycles":1},{"tenure_type":"REGULAR","cycles_completed":0,"total_cycles":0}],"failed_payments_count":0,"next_billing_time":"`+now.Add(7*24*time.Hour).UTC().Format(time.RFC3339)+`"`)
	requireStatus(t, "no headers", rig.sendPayPal(t, "WH-1", "BILLING.SUBSCRIPTION.ACTIVATED", false), http.StatusUnauthorized)
	api.genuine = false
	requireStatus(t, "forged", rig.sendPayPal(t, "WH-1", "BILLING.SUBSCRIPTION.ACTIVATED", true), http.StatusUnauthorized)
	api.genuine = true
	if limit, _ := quota(aliceToken); limit != 2 {
		t.Fatalf("a forged event changed the limit to %d", limit)
	}

	requireStatus(t, "activated", rig.sendPayPal(t, "WH-1", "BILLING.SUBSCRIPTION.ACTIVATED", true), http.StatusOK)
	requireStatus(t, "activated again", rig.sendPayPal(t, "WH-1", "BILLING.SUBSCRIPTION.ACTIVATED", true), http.StatusOK)
	stream.waitEvent(t, "billing", 2*time.Second)

	var st syncproto.BillingResponse
	requireStatus(t, "GET /v1/billing", rig.get(t, "/v1/billing", aliceToken, &st), http.StatusOK)
	if st.Plan != "pro" || st.Status != "trialing" || st.Provider != "paypal" || st.Interval != "monthly" || !st.TrialUsed ||
		st.ManageURL != "https://www.sandbox.paypal.com/myaccount/autopay/" || st.PeriodEnd == "" {
		t.Fatalf("pro = %+v", st)
	}
	if limit, plan := quota(aliceToken); limit != 5 || plan != "pro" {
		t.Fatalf("pro quota = %d %s", limit, plan)
	}
	if limit, plan := quota(bobToken); limit != 2 || plan != "free" {
		t.Fatalf("another account's quota = %d %s", limit, plan)
	}
	requireStatus(t, "checkout while Pro", rig.post(t, "/v1/billing/checkout", aliceToken, syncproto.CheckoutRequest{Plan: "monthly", Provider: "paypal"}, nil), http.StatusConflict)
	requireStatus(t, "waffo checkout while Pro", rig.post(t, "/v1/billing/checkout", aliceToken, syncproto.CheckoutRequest{Plan: "monthly"}, nil), http.StatusConflict)

	// Cancelled in the trial: still Pro until the trial's end, and the event says so again.
	api.setSub("CANCELLED", account, now.Add(time.Minute), `"cycle_executions":[{"tenure_type":"TRIAL","cycles_completed":0,"total_cycles":1},{"tenure_type":"REGULAR","cycles_completed":0,"total_cycles":0}],"failed_payments_count":0`)
	requireStatus(t, "cancelled", rig.sendPayPal(t, "WH-2", "BILLING.SUBSCRIPTION.CANCELLED", true), http.StatusOK)
	stream.waitEvent(t, "billing", 2*time.Second)
	requireStatus(t, "GET /v1/billing", rig.get(t, "/v1/billing", aliceToken, &st), http.StatusOK)
	if st.Plan != "pro" || st.Status != "canceling" || st.Provider != "paypal" {
		t.Fatalf("after cancel = %+v", st)
	}

	// An event for a subscription of nobody we know is acknowledged.
	api.setSub("ACTIVE", "424242", now, `"failed_payments_count":0`)
	requireStatus(t, "unknown account", rig.sendPayPal(t, "WH-3", "BILLING.SUBSCRIPTION.ACTIVATED", true), http.StatusOK)
	api.setSub("ACTIVE", "", now, `"failed_payments_count":0`)
	requireStatus(t, "no custom id", rig.sendPayPal(t, "WH-4", "BILLING.SUBSCRIPTION.ACTIVATED", true), http.StatusOK)
}

// A webhook that never arrives is made up for by GET /v1/billing, which the apps poll while they
// wait for the checkout: the server reads the subscription the checkout made from PayPal itself.
func TestPayPalLostWebhookIsMadeUpByPolling(t *testing.T) {
	api := newFakePayPalAPI(t)
	rig := newBillingRigWith(t, paypalConfig(api))

	var out syncproto.CheckoutResponse
	requireStatus(t, "paypal checkout", rig.post(t, "/v1/billing/checkout", aliceToken, syncproto.CheckoutRequest{Plan: "monthly", Provider: "paypal"}, &out), http.StatusOK)
	account, _ := api.created[0]["custom_id"].(string)

	var st syncproto.BillingResponse
	// Approved at PayPal, and no webhook is sent.
	now := time.Now()
	api.setSub("ACTIVE", account, now, `"cycle_executions":[{"tenure_type":"TRIAL","cycles_completed":0,"total_cycles":1},{"tenure_type":"REGULAR","cycles_completed":0,"total_cycles":0}],"failed_payments_count":0,"next_billing_time":"`+now.Add(7*24*time.Hour).UTC().Format(time.RFC3339)+`"`)
	requireStatus(t, "polled", rig.get(t, "/v1/billing", aliceToken, &st), http.StatusOK)
	if st.Plan != "pro" || st.Status != "trialing" || st.Provider != "paypal" || !st.TrialUsed {
		t.Fatalf("after polling = %+v", st)
	}
	var q struct {
		Plan string `json:"plan"`
	}
	requireStatus(t, "quota", rig.get(t, "/v1/llm/quota", aliceToken, &q), http.StatusOK)
	if q.Plan != "pro" {
		t.Fatalf("quota plan = %q", q.Plan)
	}
}

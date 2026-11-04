package syncserver_test

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/syncproto"
	"github.com/QingYunA/adhd-todo/internal/syncserver"
	"github.com/QingYunA/adhd-todo/internal/syncserver/billing"
)

// billingRig is a sync server with billing on, over a fake Waffo and a generated webhook key.
type billingRig struct {
	*harness
	webhook *rsa.PrivateKey
	mu      sync.Mutex
	calls   []map[string]any // bodies of the checkout and portal calls
}

func newBillingRig(t *testing.T) *billingRig {
	t.Helper()
	h := newHarness(t)
	apiKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	webhookKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	rig := &billingRig{harness: h, webhook: webhookKey}
	waffo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		body["_path"] = r.URL.Path
		rig.mu.Lock()
		rig.calls = append(rig.calls, body)
		rig.mu.Unlock()
		if strings.HasSuffix(r.URL.Path, "issue-session-token") {
			_, _ = io.WriteString(w, `{"data":{"token":"portal-token","portalUrl":"https://portal.example/s"}}`)
			return
		}
		_, _ = io.WriteString(w, `{"data":{"checkoutUrl":"https://checkout.example/c/1"}}`)
	}))
	t.Cleanup(waffo.Close)
	err = h.server.EnableBilling(billing.Config{
		MerchantID: "MER_test", Key: apiKey, StoreID: "STO_withdo",
		ProductMonthly: "PROD_m", ProductYearly: "PROD_y", Mode: billing.ModeTest,
		BaseURL: waffo.URL, WebhookKeys: map[string]*rsa.PublicKey{billing.ModeTest: &webhookKey.PublicKey},
	}, func(_ context.Context, subject string) (string, error) { return subject + "@example.com", nil })
	if err != nil {
		t.Fatal(err)
	}
	h.server.EnableLLM(syncserver.LLMOptions{APIKey: "k", DailyTurns: 2, DailyTurnsPro: 5})
	return rig
}

func (r *billingRig) lastCall(t *testing.T) map[string]any {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.calls) == 0 {
		t.Fatal("Waffo was not called")
	}
	return r.calls[len(r.calls)-1]
}

// webhook posts a signed event for account at time at and returns the status.
func (r *billingRig) sendWebhook(t *testing.T, key *rsa.PrivateKey, id, eventType, account string, at time.Time, extra map[string]any) int {
	t.Helper()
	data := map[string]any{
		"orderId": "ORD_1", "productName": "Withdo Pro", "orderMetadata": map[string]any{"account": account},
	}
	for k, v := range extra {
		data[k] = v
	}
	body, _ := json.Marshal(map[string]any{
		"id": id, "timestamp": at.UTC().Format(time.RFC3339), "eventType": eventType,
		"storeId": "STO_withdo", "mode": "test", "data": data,
	})
	ms := strconv.FormatInt(time.Now().UnixMilli(), 10)
	sum := sha256.Sum256([]byte(ms + "." + string(body)))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest(http.MethodPost, r.http.URL+"/v1/billing/waffo", bytes.NewReader(body))
	req.Header.Set("X-Waffo-Signature", "t="+ms+",v1="+base64.StdEncoding.EncodeToString(sig))
	resp, err := r.http.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	return resp.StatusCode
}

func TestBillingIsOffWithoutAKey(t *testing.T) {
	h := newHarness(t)
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/v1/billing"}, {http.MethodPost, "/v1/billing/checkout"}, {http.MethodPost, "/v1/billing/waffo"},
	} {
		resp, body := h.do(t, tc.method, tc.path, aliceToken, map[string]string{"plan": "monthly"})
		var out struct {
			Code string `json:"code"`
		}
		if resp.StatusCode != http.StatusServiceUnavailable || json.Unmarshal(body, &out) != nil || out.Code != syncserver.CodeBillingDisabled {
			t.Fatalf("%s %s = %d %s", tc.method, tc.path, resp.StatusCode, body)
		}
	}
	var cfg syncproto.ConfigResponse
	h.get(t, "/v1/config", "", &cfg)
	if cfg.Billing {
		t.Fatal("config says billing is on")
	}
}

func TestBillingRoutes(t *testing.T) {
	rig := newBillingRig(t)
	var cfg syncproto.ConfigResponse
	rig.get(t, "/v1/config", "", &cfg)
	if !cfg.Billing {
		t.Fatal("config says billing is off")
	}
	requireStatus(t, "no token", rig.get(t, "/v1/billing", "", nil), http.StatusUnauthorized)
	requireStatus(t, "checkout without a token", rig.post(t, "/v1/billing/checkout", "", syncproto.CheckoutRequest{Plan: "monthly"}, nil), http.StatusUnauthorized)

	var free syncproto.BillingResponse
	requireStatus(t, "GET /v1/billing", rig.get(t, "/v1/billing", aliceToken, &free), http.StatusOK)
	if free.Plan != "free" || free.Status != "" || free.TrialUsed || free.Monthly != "$6.99" || free.Yearly != "$58.72" || free.ManageURL != "" {
		t.Fatalf("free = %+v", free)
	}

	requireStatus(t, "unknown plan", rig.post(t, "/v1/billing/checkout", aliceToken, syncproto.CheckoutRequest{Plan: "weekly"}, nil), http.StatusBadRequest)
	var out syncproto.CheckoutResponse
	requireStatus(t, "checkout", rig.post(t, "/v1/billing/checkout", aliceToken, syncproto.CheckoutRequest{Plan: "yearly", Lang: "en"}, &out), http.StatusOK)
	if out.URL != "https://checkout.example/c/1" {
		t.Fatalf("checkout url = %q", out.URL)
	}
	call := rig.lastCall(t)
	meta, _ := call["metadata"].(map[string]any)
	if call["productId"] != "PROD_y" || call["buyerEmail"] != aliceSubject+"@example.com" || meta["account"] == "" || call["withTrial"] != true {
		t.Fatalf("checkout call = %v", call)
	}
}

func TestWebhookMakesAnAccountPro(t *testing.T) {
	rig := newBillingRig(t)
	device := rig.device(t, aliceToken, "phone")
	stream := rig.stream(t, aliceToken, device.DeviceID)
	stream.waitReady(t)

	var out syncproto.CheckoutResponse
	requireStatus(t, "checkout", rig.post(t, "/v1/billing/checkout", aliceToken, syncproto.CheckoutRequest{Plan: "monthly"}, &out), http.StatusOK)
	account := rig.lastCall(t)["metadata"].(map[string]any)["account"].(string)

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

	now := time.Now()
	end := now.Add(7 * 24 * time.Hour).UTC().Format(time.RFC3339)
	// Not signed by Waffo.
	stranger, _ := rsa.GenerateKey(rand.Reader, 2048)
	requireStatus(t, "forged", rig.sendWebhook(t, stranger, "w1", "subscription.activated", account, now, nil), http.StatusUnauthorized)
	if limit, _ := quota(aliceToken); limit != 2 {
		t.Fatalf("a forged event changed the limit to %d", limit)
	}

	activated := map[string]any{"periodNumber": 0, "billingPeriod": "yearly", "currentPeriodEnd": end}
	requireStatus(t, "activated", rig.sendWebhook(t, rig.webhook, "w2", "subscription.activated", account, now, activated), http.StatusOK)
	requireStatus(t, "activated again", rig.sendWebhook(t, rig.webhook, "w2", "subscription.activated", account, now, activated), http.StatusOK)
	stream.waitEvent(t, "billing", 2*time.Second)

	var st syncproto.BillingResponse
	requireStatus(t, "GET /v1/billing", rig.get(t, "/v1/billing", aliceToken, &st), http.StatusOK)
	if st.Plan != "pro" || st.Status != "trialing" || st.Interval != "yearly" || !st.TrialUsed || st.PeriodEnd != end || st.ManageURL != "https://portal.example/s#token=portal-token" {
		t.Fatalf("pro = %+v", st)
	}
	if call := rig.lastCall(t); call["buyerIdentity"] != aliceSubject+"@example.com" || call["storeId"] != "STO_withdo" {
		t.Fatalf("portal call = %v", call)
	}
	if limit, plan := quota(aliceToken); limit != 5 || plan != "pro" {
		t.Fatalf("pro quota = %d %s", limit, plan)
	}
	if limit, plan := quota(bobToken); limit != 2 || plan != "free" {
		t.Fatalf("another account's quota = %d %s", limit, plan)
	}

	// Already Pro: a second checkout is refused.
	requireStatus(t, "checkout while Pro", rig.post(t, "/v1/billing/checkout", aliceToken, syncproto.CheckoutRequest{Plan: "monthly"}, nil), http.StatusConflict)

	requireStatus(t, "canceled", rig.sendWebhook(t, rig.webhook, "w3", "subscription.canceled", account, now.Add(time.Minute), nil), http.StatusOK)
	stream.waitEvent(t, "billing", 2*time.Second)
	requireStatus(t, "GET /v1/billing", rig.get(t, "/v1/billing", aliceToken, &st), http.StatusOK)
	if st.Plan != "free" || st.Status != "canceled" || !st.TrialUsed {
		t.Fatalf("after cancel = %+v", st)
	}
	if limit, plan := quota(aliceToken); limit != 2 || plan != "free" {
		t.Fatalf("quota after cancel = %d %s", limit, plan)
	}

	// The trial is not offered again.
	requireStatus(t, "checkout again", rig.post(t, "/v1/billing/checkout", aliceToken, syncproto.CheckoutRequest{Plan: "monthly"}, nil), http.StatusOK)
	if rig.lastCall(t)["withTrial"] != false {
		t.Fatalf("checkout call = %v", rig.lastCall(t))
	}
}

func TestWebhookForAnUnknownAccountIsAcknowledged(t *testing.T) {
	rig := newBillingRig(t)
	requireStatus(t, "unknown account", rig.sendWebhook(t, rig.webhook, "u1", "subscription.activated", "424242", time.Now(), nil), http.StatusOK)
	requireStatus(t, "no account", rig.sendWebhook(t, rig.webhook, "u2", "subscription.activated", "", time.Now(), nil), http.StatusOK)
}

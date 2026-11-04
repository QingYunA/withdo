package core

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QingYunA/adhd-todo/internal/syncclient"
)

// fakeBilling is a sync server's /v1/config, /v1/billing, /v1/billing/checkout, /v1/llm/quota and
// /v1/events, the pieces the phone's plan reads.
type fakeBilling struct {
	*httptest.Server
	mu          sync.Mutex
	offered     bool
	plan        string
	status      string
	interval    string
	periodEnd   string
	trialUsed   bool
	checkout    func(plan string) (int, string) // status and body of POST /v1/billing/checkout
	checks      []string                        // the "plan" and "lang" of each checkout request
	billingGets int
	event       chan struct{} // a send makes the events stream say "billing"
}

func newFakeBilling(t *testing.T) *fakeBilling {
	t.Helper()
	f := &fakeBilling{offered: true, plan: "free", event: make(chan struct{}, 4)}
	f.Server = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.Close)
	return f
}

func (f *fakeBilling) serve(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	offered, plan, status, interval, end, trialUsed := f.offered, f.plan, f.status, f.interval, f.periodEnd, f.trialUsed
	f.mu.Unlock()
	switch r.URL.Path {
	case "/v1/config":
		writeTestJSON(w, map[string]any{"llm": true, "billing": offered})
	case "/v1/llm/quota":
		limit := 20
		if plan == "pro" {
			limit = 100
		}
		writeTestJSON(w, map[string]any{"limit": limit, "left": limit - 6, "resets_at": time.Now().Add(6 * time.Hour).UTC().Format(time.RFC3339)})
	case "/v1/billing":
		f.mu.Lock()
		f.billingGets++
		n := f.billingGets
		f.mu.Unlock()
		if !offered {
			http.Error(w, `{"error":"billing_disabled","code":"billing_disabled"}`, http.StatusServiceUnavailable)
			return
		}
		manage := ""
		if status != "" {
			manage = "https://portal.example/session-" + string(rune('0'+n%10))
		}
		writeTestJSON(w, map[string]any{
			"plan": plan, "status": status, "interval": interval, "period_end": end, "trial_used": trialUsed,
			"monthly": "$6.99", "yearly": "$58.72", "manage_url": manage,
		})
	case "/v1/billing/checkout":
		raw, _ := io.ReadAll(r.Body)
		var body struct {
			Plan string `json:"plan"`
			Lang string `json:"lang"`
		}
		_ = json.Unmarshal(raw, &body)
		f.mu.Lock()
		f.checks = append(f.checks, body.Plan+"/"+body.Lang)
		answer := f.checkout
		f.mu.Unlock()
		code, text := http.StatusOK, `{"url":"https://pay.example/checkout/abc"}`
		if answer != nil {
			code, text = answer(body.Plan)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_, _ = io.WriteString(w, text)
	case "/v1/events":
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		for {
			select {
			case <-f.event:
				_, _ = io.WriteString(w, "event: billing\ndata: {}\n\n")
				w.(http.Flusher).Flush()
			case <-r.Context().Done():
				return
			}
		}
	default:
		http.NotFound(w, r)
	}
}

func (f *fakeBilling) set(plan, status, end string, trialUsed bool) {
	f.mu.Lock()
	f.plan, f.status, f.periodEnd, f.trialUsed = plan, status, end, trialUsed
	f.mu.Unlock()
}

// billingPhone opens a phone signed in to the fake server with a development token.
func billingPhone(t *testing.T, f *fakeBilling) *Core {
	t.Helper()
	dir := t.TempDir()
	phone, err := open(dir, newMapSecrets(), func() time.Time { return demoNow })
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = phone.Close() })
	if err := syncclient.SaveConfigAt(filepath.Join(dir, "sync.json"), syncclient.Config{Server: f.URL, Token: "dev-token"}); err != nil {
		t.Fatalf("write sync.json: %v", err)
	}
	return phone
}

func readBilling(t *testing.T, phone *Core) billingJSON {
	t.Helper()
	raw, err := phone.Billing()
	if err != nil {
		t.Fatalf("Billing: %v", err)
	}
	var out billingJSON
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("decode Billing %q: %v", raw, err)
	}
	return out
}

// waitBilling reads Billing until check accepts it (the refresh runs in the background) or fails.
func waitBilling(t *testing.T, phone *Core, check func(billingJSON) bool) billingJSON {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		got := readBilling(t, phone)
		if check(got) {
			return got
		}
		if time.Now().After(deadline) {
			t.Fatalf("Billing never got there; last %+v", got)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestBillingIsEmptyUntilTheServerHasAnswered(t *testing.T) {
	f := newFakeBilling(t)
	phone := billingPhone(t, f)
	first := readBilling(t, phone)
	if first.Offered || first.Plan != "" {
		t.Errorf("before the answer = %+v, want nothing", first)
	}
	got := waitBilling(t, phone, func(b billingJSON) bool { return b.Plan != "" })
	if !got.Offered || got.Plan != "free" || got.Monthly != "$6.99" || got.Yearly != "$58.72" || got.TrialUsed || got.ManageURL != "" {
		t.Errorf("free plan = %+v", got)
	}
}

func TestBillingReportsAProPlanWithItsPeriodAndPortal(t *testing.T) {
	f := newFakeBilling(t)
	f.set("pro", "trialing", "2026-11-11T09:00:00Z", true)
	phone := billingPhone(t, f)
	got := waitBilling(t, phone, func(b billingJSON) bool { return b.Plan == "pro" })
	if got.Status != "trialing" || got.PeriodEnd != "2026-11-11T09:00:00Z" || !got.TrialUsed || !strings.HasPrefix(got.ManageURL, "https://portal.example/") {
		t.Errorf("pro plan = %+v", got)
	}
}

func TestBillingReportsTheIntervalWhenTheServerSaysIt(t *testing.T) {
	f := newFakeBilling(t)
	f.set("pro", "active", "2027-11-11T09:00:00Z", true)
	f.mu.Lock()
	f.interval = "yearly"
	f.mu.Unlock()
	phone := billingPhone(t, f)
	got := waitBilling(t, phone, func(b billingJSON) bool { return b.Plan == "pro" })
	if got.Interval != "yearly" {
		t.Errorf("interval = %q, want yearly", got.Interval)
	}
}

func TestBillingIsNotOfferedByAServerWithoutIt(t *testing.T) {
	f := newFakeBilling(t)
	f.mu.Lock()
	f.offered = false
	f.mu.Unlock()
	phone := billingPhone(t, f)
	if err := phone.RefreshBilling(); err != nil {
		t.Fatalf("RefreshBilling: %v", err)
	}
	if got := readBilling(t, phone); got.Offered || got.Plan != "" {
		t.Errorf("billing off = %+v", got)
	}
}

func TestBillingSignedOutKnowsOnlyThatItIsOffered(t *testing.T) {
	f := newFakeBilling(t)
	dir := t.TempDir()
	phone, err := open(dir, newMapSecrets(), func() time.Time { return demoNow })
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = phone.Close() }()
	// sync.json names the server, but there is no sign-in: nothing to ask the account.
	if err := syncclient.SaveConfigAt(filepath.Join(dir, "sync.json"), syncclient.Config{Server: f.URL, Auth: syncclient.AuthPocketBase}); err != nil {
		t.Fatalf("write sync.json: %v", err)
	}
	if err := phone.RefreshBilling(); err != nil {
		t.Fatalf("RefreshBilling: %v", err)
	}
	if got := readBilling(t, phone); !got.Offered || got.Plan != "" {
		t.Errorf("signed out = %+v, want offered without a plan", got)
	}
}

func TestRefreshBillingFetchesNowAndKeepsTheCache(t *testing.T) {
	f := newFakeBilling(t)
	phone := billingPhone(t, f)
	waitBilling(t, phone, func(b billingJSON) bool { return b.Plan == "free" })

	f.set("pro", "active", "2027-11-11T09:00:00Z", true)
	if got := readBilling(t, phone); got.Plan != "free" {
		t.Errorf("Billing asked the network: %+v", got)
	}
	if err := phone.RefreshBilling(); err != nil {
		t.Fatalf("RefreshBilling: %v", err)
	}
	got := readBilling(t, phone)
	if got.Plan != "pro" || got.Status != "active" {
		t.Errorf("after RefreshBilling = %+v", got)
	}
}

func TestRefreshBillingTellsTheListenerOnlyWhenThePlanChanged(t *testing.T) {
	f := newFakeBilling(t)
	phone := billingPhone(t, f)
	statuses := make(chan string, 16)
	phone.SetListener(&statusListener{statuses: statuses})
	waitBilling(t, phone, func(b billingJSON) bool { return b.Plan == "free" })
	drain := func() (rev int64) {
		for {
			select {
			case raw := <-statuses:
				var s statusJSON
				if err := json.Unmarshal([]byte(raw), &s); err != nil {
					t.Fatalf("decode status %q: %v", raw, err)
				}
				rev = s.BillingRev
			case <-time.After(300 * time.Millisecond):
				return rev
			}
		}
	}
	first := drain()
	if first == 0 {
		t.Fatalf("the first answer did not raise billing_rev")
	}

	if err := phone.RefreshBilling(); err != nil { // nothing changed
		t.Fatalf("RefreshBilling: %v", err)
	}
	if rev := drain(); rev != 0 {
		t.Errorf("an unchanged plan told the listener (billing_rev %d)", rev)
	}

	f.set("pro", "trialing", "2026-11-11T09:00:00Z", true)
	if err := phone.RefreshBilling(); err != nil {
		t.Fatalf("RefreshBilling: %v", err)
	}
	if rev := drain(); rev <= first {
		t.Errorf("a new plan left billing_rev at %d (was %d)", rev, first)
	}
}

func TestModelStateCarriesThePlanAndTheQuotasOwnLimit(t *testing.T) {
	f := newFakeBilling(t)
	phone := billingPhone(t, f)
	if err := phone.RefreshBilling(); err != nil {
		t.Fatalf("RefreshBilling: %v", err)
	}
	got := modelPhase(t, phone, phaseReady)
	if got.Plan != "free" || got.Limit != 20 {
		t.Errorf("free model state = %+v", got)
	}

	f.set("pro", "active", "2027-11-11T09:00:00Z", true)
	if err := phone.RefreshBilling(); err != nil { // the change also asks the quota again
		t.Fatalf("RefreshBilling: %v", err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		raw, _ := phone.ModelState()
		var m modelStateJSON
		_ = json.Unmarshal([]byte(raw), &m)
		if m.Plan == "pro" && m.Limit == 100 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("model state never followed the plan: %s", raw)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestCheckoutReturnsTheURLAndSendsPlanAndLanguage(t *testing.T) {
	f := newFakeBilling(t)
	phone := billingPhone(t, f)
	url, err := phone.Checkout("yearly", "zh-Hans")
	if err != nil {
		t.Fatalf("Checkout: %v", err)
	}
	if url != "https://pay.example/checkout/abc" {
		t.Errorf("url = %q", url)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.checks) != 1 || f.checks[0] != "yearly/zh-Hans" {
		t.Errorf("the server was asked %v", f.checks)
	}
}

func TestCheckoutNamesEachFailure(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		want   string
	}{
		{"already Pro", 409, `{"error":"already_subscribed","code":"already_subscribed"}`, "already_subscribed"},
		{"too many", 429, `{"error":"rate_limited","code":"rate_limited"}`, "rate_limited"},
		{"provider down", 502, `{"error":"billing_upstream","code":"billing_upstream"}`, "upstream"},
		{"billing off", 503, `{"error":"billing_disabled","code":"billing_disabled"}`, "off"},
		{"session refused", 401, `{"error":"unauthorized","code":"unauthorized"}`, "signed_out"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFakeBilling(t)
			f.checkout = func(string) (int, string) { return tc.status, tc.body }
			phone := billingPhone(t, f)
			_, err := phone.Checkout("monthly", "en")
			if err == nil || err.Error() != tc.want {
				t.Errorf("Checkout error = %v, want %q", err, tc.want)
			}
		})
	}

	f := newFakeBilling(t)
	phone := billingPhone(t, f)
	if _, err := phone.Checkout("weekly", "en"); err == nil || err.Error() != "bad_plan" {
		t.Errorf("a plan the server does not sell: %v", err)
	}
	f.Close()
	if _, err := phone.Checkout("monthly", "en"); err == nil || err.Error() != "network" {
		t.Errorf("an unreachable server: %v", err)
	}
}

func TestCheckoutSignedOutHasNoAccountToAsk(t *testing.T) {
	phone, err := open(t.TempDir(), newMapSecrets(), func() time.Time { return demoNow })
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = phone.Close() }()
	if _, err := phone.Checkout("monthly", "en"); err == nil || err.Error() != "signed_out" {
		t.Errorf("Checkout signed out: %v", err)
	}
}

// TestTheBillingEventRefreshesThePlan: the runner's events stream says "billing" after a purchase,
// and the core reads the plan again and tells the listener without the app asking.
func TestTheBillingEventRefreshesThePlan(t *testing.T) {
	f := newFakeBilling(t)
	phone := billingPhone(t, f)
	if _, err := phone.db.Exec(`INSERT INTO sync_state (key, value) VALUES ('device_id', 'phone-1')`); err != nil {
		t.Fatalf("link the phone: %v", err)
	}
	statuses := make(chan string, 64)
	phone.SetListener(&statusListener{statuses: statuses})
	waitBilling(t, phone, func(b billingJSON) bool { return b.Plan == "free" })
	if err := phone.StartSync(); err != nil {
		t.Fatalf("StartSync: %v", err)
	}
	defer func() { _ = phone.StopSync() }()

	f.set("pro", "trialing", "2026-11-11T09:00:00Z", true)
	time.Sleep(300 * time.Millisecond) // the stream connects
	f.event <- struct{}{}
	got := waitBilling(t, phone, func(b billingJSON) bool { return b.Plan == "pro" })
	if got.Status != "trialing" {
		t.Errorf("after the event = %+v", got)
	}
}

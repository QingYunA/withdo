package data

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestPerMonthAndSavingPercentReadTheServersPrices(t *testing.T) {
	for _, c := range []struct{ yearly, perMonth string }{
		{"$58.72", "$4.89"}, {"€60", "€5.00"}, {"58.72", "4.89"}, {"", ""}, {"free", ""},
	} {
		if got := PerMonth(c.yearly); got != c.perMonth {
			t.Errorf("PerMonth(%q) = %q, want %q", c.yearly, got, c.perMonth)
		}
	}
	for _, c := range []struct {
		monthly, yearly string
		pct             int
	}{
		{"$6.99", "$58.72", 30}, {"$10", "$120", 0}, {"$10", "$150", 0}, {"", "$58.72", 0}, {"$6.99", "", 0},
	} {
		if got := SavingPercent(c.monthly, c.yearly); got != c.pct {
			t.Errorf("SavingPercent(%q, %q) = %d, want %d", c.monthly, c.yearly, got, c.pct)
		}
	}
}

// billingServer is a sync server that sells (or not) Withdo Pro to the account whose token is "tok".
type billingServer struct {
	*httptest.Server
	offered   atomic.Bool
	pro       atomic.Bool
	trialUsed atomic.Bool
	checkout  atomic.Int32 // the status the next checkout answers, 0 for 200
	portals   atomic.Int32
	plans     chan string
}

func newBillingServer(t *testing.T, offered bool) *billingServer {
	t.Helper()
	f := &billingServer{plans: make(chan string, 8)}
	f.offered.Store(offered)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/config", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"auth":"pocketbase","llm":true,"billing":` + strconv.FormatBool(f.offered.Load()) + `}`))
	})
	mux.HandleFunc("GET /v1/llm/quota", func(w http.ResponseWriter, _ *http.Request) {
		limit, plan := 20, "free"
		if f.pro.Load() {
			limit, plan = 100, "pro"
		}
		reset := time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"limit":` + strconv.Itoa(limit) + `,"left":` + strconv.Itoa(limit-6) + `,"resets_at":"` + reset + `","plan":"` + plan + `"}`))
	})
	mux.HandleFunc("GET /v1/billing", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		body := `{"plan":"free","status":"","period_end":"","trial_used":` + strconv.FormatBool(f.trialUsed.Load()) +
			`,"monthly":"$6.99","yearly":"$58.72","manage_url":""}`
		if f.pro.Load() {
			f.portals.Add(1)
			body = `{"plan":"pro","status":"trialing","period_end":"2026-11-11T00:00:00Z","trial_used":true,` +
				`"monthly":"$6.99","yearly":"$58.72","manage_url":"https://portal.example.com/s/` + strconv.Itoa(int(f.portals.Load())) + `"}`
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	})
	mux.HandleFunc("POST /v1/billing/checkout", func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 256)
		n, _ := r.Body.Read(buf)
		f.plans <- string(buf[:n])
		switch f.checkout.Load() {
		case http.StatusConflict:
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"error":"x","code":"already_subscribed"}`))
		case http.StatusTooManyRequests:
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"x","code":"rate_limited"}`))
		default:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"url":"https://pay.example.com/c/1"}`))
		}
	})
	f.Server = httptest.NewServer(mux)
	t.Cleanup(f.Close)
	return f
}

// billingIs polls until the plan reads ok, since the answer comes from a background request.
func billingIs(t *testing.T, r *Real, ok func(BillingInfo) bool) BillingInfo {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		info := r.BillingInfo()
		if ok(info) {
			return info
		}
		if time.Now().After(deadline) {
			t.Fatalf("the plan stayed %+v", info)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestBillingFreeAccountReadsThePlanAndThePrices(t *testing.T) {
	f := newBillingServer(t, true)
	r := officialReal(t, f.URL, true)
	info := billingIs(t, r, func(b BillingInfo) bool { return b.Known })
	if !info.Offered || info.Pro || info.Subscribed() || info.TrialUsed || info.Monthly != "$6.99" || info.Yearly != "$58.72" {
		t.Fatalf("info = %+v", info)
	}
}

func TestBillingFollowsAPurchaseAndTheLimitFollowsThePlan(t *testing.T) {
	f := newBillingServer(t, true)
	r := officialReal(t, f.URL, true)
	billingIs(t, r, func(b BillingInfo) bool { return b.Known })
	if got := officialIs(t, r, OfficialReady).Limit; got != 20 {
		t.Fatalf("limit = %d, want the free plan's", got)
	}

	f.pro.Store(true) // the "billing" event arrives: the runner calls RefreshBilling
	r.RefreshBilling()
	info := billingIs(t, r, func(b BillingInfo) bool { return b.Pro })
	if !info.Subscribed() || !info.Trialing() || info.PeriodEnd.IsZero() || !info.TrialUsed {
		t.Fatalf("info = %+v", info)
	}
	deadline := time.Now().Add(5 * time.Second) // the plan's change asks the quota again
	for r.OfficialInfo().Limit != 100 {
		if time.Now().After(deadline) {
			t.Fatalf("limit stayed %d after the plan became Pro", r.OfficialInfo().Limit)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestBillingIsNotOfferedBySomeServersOrSignedOut(t *testing.T) {
	off := newBillingServer(t, false)
	r := officialReal(t, off.URL, true)
	r.BillingInfo() // starts the question
	time.Sleep(150 * time.Millisecond)
	if info := r.BillingInfo(); info.Offered || info.Known {
		t.Fatalf("a server that does not sell Pro offers %+v", info)
	}

	on := newBillingServer(t, true)
	out := officialReal(t, on.URL, false)
	out.BillingInfo()
	time.Sleep(150 * time.Millisecond)
	if info := out.BillingInfo(); info.Offered {
		t.Fatalf("signed out offers %+v", info)
	}
}

func TestStartCheckoutReturnsThePageAndWordsTheRefusals(t *testing.T) {
	f := newBillingServer(t, true)
	r := officialReal(t, f.URL, true)
	url, err := r.StartCheckout(context.Background(), PlanYearly, "zh-Hans")
	if err != nil || url != "https://pay.example.com/c/1" {
		t.Fatalf("StartCheckout = %q, %v", url, err)
	}
	if body := <-f.plans; body == "" || !strings.Contains(body, `"yearly"`) || !strings.Contains(body, `"zh-Hans"`) {
		t.Fatalf("request = %q", body)
	}
	f.checkout.Store(http.StatusConflict)
	if _, err := r.StartCheckout(context.Background(), PlanMonthly, "en"); !errors.Is(err, ErrAlreadySubscribed) {
		t.Fatalf("a Pro account's checkout = %v", err)
	}
	f.checkout.Store(http.StatusTooManyRequests)
	if _, err := r.StartCheckout(context.Background(), PlanMonthly, "en"); !errors.Is(err, ErrCheckoutThrottled) {
		t.Fatalf("a throttled checkout = %v", err)
	}
}

func TestManageURLIsFreshOnEveryAsk(t *testing.T) {
	f := newBillingServer(t, true)
	r := officialReal(t, f.URL, true)
	if _, err := r.ManageURL(context.Background()); err == nil {
		t.Fatal("a Free account got a portal link")
	}
	f.pro.Store(true)
	a, errA := r.ManageURL(context.Background())
	b, errB := r.ManageURL(context.Background())
	if errA != nil || errB != nil || a == "" || a == b {
		t.Fatalf("ManageURL = %q, %v then %q, %v: want a new link each time", a, errA, b, errB)
	}
}

package syncclient

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const billingBody = `{"plan":"pro","status":"trialing","interval":"yearly","period_end":"2026-11-11T12:00:00Z","trial_used":true,` +
	`"monthly":"$6.99","yearly":"$58.72","manage_url":"https://portal.example/s#token=t"}`

func TestBilling(t *testing.T) {
	t.Run("reads the plan with the token", func(t *testing.T) {
		srv, seen, _ := voiceServer(t, 200, billingBody)
		b, err := voiceClient(srv.URL, StaticToken("tok")).Billing(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		want := Billing{
			Plan: "pro", Status: "trialing", Interval: "yearly", PeriodEnd: time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC), TrialUsed: true,
			Monthly: "$6.99", Yearly: "$58.72", ManageURL: "https://portal.example/s#token=t",
		}
		if b != want || !b.Pro() {
			t.Fatalf("got %+v, want %+v", b, want)
		}
		if seen.Header.Get("Authorization") != "Bearer tok" || seen.URL.Path != "/v1/billing" || seen.Method != http.MethodGet {
			t.Fatalf("request: %v %v", seen.Header, seen.URL)
		}
	})
	t.Run("an older server says no interval", func(t *testing.T) {
		srv, _, _ := voiceServer(t, 200, `{"plan":"pro","status":"active","period_end":"2026-11-11T12:00:00Z","trial_used":true,"monthly":"$6.99","yearly":"$58.72","manage_url":""}`)
		b, err := voiceClient(srv.URL, StaticToken("tok")).Billing(context.Background())
		if err != nil || b.Interval != "" || !b.Pro() {
			t.Fatalf("got %+v, %v", b, err)
		}
	})
	t.Run("a free account has no period", func(t *testing.T) {
		srv, _, _ := voiceServer(t, 200, `{"plan":"free","status":"","period_end":"","trial_used":false,"monthly":"$6.99","yearly":"$58.72","manage_url":""}`)
		b, err := voiceClient(srv.URL, StaticToken("tok")).Billing(context.Background())
		if err != nil || b.Pro() || !b.PeriodEnd.IsZero() || b.Plan != "free" {
			t.Fatalf("got %+v, %v", b, err)
		}
	})
	cases := []struct {
		name   string
		status int
		body   string
		want   error
	}{
		{"401 is not signed in", 401, `{"error":"unauthenticated"}`, ErrBillingNotSignedIn},
		{"503 is off", 503, `{"error":"billing_disabled","code":"billing_disabled"}`, ErrBillingOff},
		{"404 is an older server", 404, `404 page not found`, ErrBillingOff},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv, _, _ := voiceServer(t, tc.status, tc.body)
			if _, err := voiceClient(srv.URL, StaticToken("tok")).Billing(context.Background()); !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
		})
	}
	t.Run("no token and a signed-out source", func(t *testing.T) {
		if _, err := voiceClient("http://127.0.0.1:1", failingToken{ErrSignedOut}).Billing(context.Background()); !errors.Is(err, ErrBillingNotSignedIn) {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("unreachable", func(t *testing.T) {
		srv := httptest.NewServer(http.NotFoundHandler())
		url := srv.URL
		srv.Close()
		if _, err := voiceClient(url, StaticToken("tok")).Billing(context.Background()); !errors.Is(err, ErrBillingNetwork) {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("an answer without a plan is an error", func(t *testing.T) {
		srv, _, _ := voiceServer(t, 200, `{}`)
		if _, err := voiceClient(srv.URL, StaticToken("tok")).Billing(context.Background()); err == nil {
			t.Fatal("an empty answer was accepted")
		}
	})
}

func TestCheckout(t *testing.T) {
	t.Run("posts the plan and language and returns the URL", func(t *testing.T) {
		srv, seen, body := voiceServer(t, 200, `{"url":"https://checkout.example/c/1"}`)
		url, err := voiceClient(srv.URL, StaticToken("tok")).Checkout(context.Background(), PlanYearly, "", "zh-Hans")
		if err != nil || url != "https://checkout.example/c/1" {
			t.Fatalf("got %q, %v", url, err)
		}
		if seen.Method != http.MethodPost || seen.URL.Path != "/v1/billing/checkout" || seen.Header.Get("Authorization") != "Bearer tok" {
			t.Fatalf("request: %v %v %v", seen.Method, seen.URL, seen.Header)
		}
		if string(*body) != `{"plan":"yearly","lang":"zh-Hans"}` {
			t.Fatalf("body = %s", *body)
		}
	})
	cases := []struct {
		name   string
		status int
		body   string
		want   error
	}{
		{"401", 401, `{"error":"unauthenticated"}`, ErrBillingNotSignedIn},
		{"503 billing_disabled", 503, `{"error":"billing_disabled","code":"billing_disabled"}`, ErrBillingOff},
		{"409 already_subscribed", 409, `{"error":"already_subscribed","code":"already_subscribed"}`, ErrAlreadySubscribed},
		{"429", 429, `{"error":"rate_limited","code":"rate_limited"}`, ErrCheckoutThrottled},
		{"502 billing_upstream", 502, `{"error":"billing_upstream","code":"billing_upstream"}`, ErrBillingUpstream},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv, _, _ := voiceServer(t, tc.status, tc.body)
			if _, err := voiceClient(srv.URL, StaticToken("tok")).Checkout(context.Background(), PlanMonthly, "", ""); !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
		})
	}
	t.Run("a bad plan is the server's 400, not a sentinel", func(t *testing.T) {
		srv, _, _ := voiceServer(t, 400, `{"error":"bad_plan","code":"bad_plan"}`)
		_, err := voiceClient(srv.URL, StaticToken("tok")).Checkout(context.Background(), "weekly", "", "")
		if HTTPStatus(err) != 400 {
			t.Fatalf("err = %v", err)
		}
	})
}

func TestBillingOffered(t *testing.T) {
	for body, want := range map[string]bool{`{"auth":"pocketbase","llm":true,"billing":true}`: true, `{"billing":false}`: false, `{"llm":true}`: false} {
		srv, _, _ := voiceServer(t, 200, body)
		if got := voiceClient(srv.URL, nil).BillingOffered(context.Background()); got != want {
			t.Fatalf("%s: BillingOffered = %v", body, got)
		}
	}
}

func TestEventsCallsOnBillingForTheBillingEvent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, ": connected\n\nevent: poke\ndata: 3\n\nevent: billing\ndata: changed\n\nevent: other\ndata: x\n\n")
	}))
	t.Cleanup(srv.Close)
	var pokes, billings int
	err := voiceClient(srv.URL, StaticToken("tok")).events(context.Background(), "dev", func() { pokes++ }, func() { billings++ })
	if !errors.Is(err, io.EOF) || pokes != 1 || billings != 1 {
		t.Fatalf("err=%v pokes=%d billings=%d", err, pokes, billings)
	}
	// A client that does not listen for billing notices ignores them.
	pokes = 0
	err = voiceClient(srv.URL, StaticToken("tok")).events(context.Background(), "dev", func() { pokes++ }, nil)
	if !errors.Is(err, io.EOF) || pokes != 1 {
		t.Fatalf("nil handler: err=%v pokes=%d", err, pokes)
	}
}

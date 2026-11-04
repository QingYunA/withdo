package syncclient

import (
	"context"
	"errors"
	"testing"
)

func TestCheckoutProvider(t *testing.T) {
	t.Run("paypal is named in the body", func(t *testing.T) {
		srv, seen, body := voiceServer(t, 200, `{"url":"https://paypal.example/approve"}`)
		url, err := voiceClient(srv.URL, StaticToken("tok")).Checkout(context.Background(), PlanMonthly, ProviderPayPal, "en")
		if err != nil || url != "https://paypal.example/approve" {
			t.Fatalf("got %q, %v", url, err)
		}
		if seen.URL.Path != "/v1/billing/checkout" || string(*body) != `{"plan":"monthly","lang":"en","provider":"paypal"}` {
			t.Fatalf("request: %v %s", seen.URL, *body)
		}
	})
	t.Run("waffo and the empty provider send none, as before", func(t *testing.T) {
		for _, provider := range []string{"", ProviderWaffo} {
			srv, _, body := voiceServer(t, 200, `{"url":"https://checkout.example/c/1"}`)
			if _, err := voiceClient(srv.URL, StaticToken("tok")).Checkout(context.Background(), PlanYearly, provider, ""); err != nil {
				t.Fatal(err)
			}
			if string(*body) != `{"plan":"yearly"}` {
				t.Fatalf("provider %q: body = %s", provider, *body)
			}
		}
	})
	t.Run("503 paypal_disabled is ErrPayPalOff, not ErrBillingOff", func(t *testing.T) {
		srv, _, _ := voiceServer(t, 503, `{"error":"paypal_disabled","code":"paypal_disabled"}`)
		_, err := voiceClient(srv.URL, StaticToken("tok")).Checkout(context.Background(), PlanMonthly, ProviderPayPal, "")
		if !errors.Is(err, ErrPayPalOff) || errors.Is(err, ErrBillingOff) {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("a bad provider is the server's 400", func(t *testing.T) {
		srv, _, _ := voiceServer(t, 400, `{"error":"bad_provider","code":"bad_provider"}`)
		_, err := voiceClient(srv.URL, StaticToken("tok")).Checkout(context.Background(), PlanMonthly, "stripe", "")
		if HTTPStatus(err) != 400 {
			t.Fatalf("err = %v", err)
		}
	})
}

func TestBillingProvider(t *testing.T) {
	srv, _, _ := voiceServer(t, 200, `{"plan":"pro","status":"active","interval":"monthly","period_end":"2026-11-11T12:00:00Z","trial_used":true,`+
		`"monthly":"$6.99","yearly":"$58.72","manage_url":"https://www.paypal.com/myaccount/autopay/","provider":"paypal"}`)
	b, err := voiceClient(srv.URL, StaticToken("tok")).Billing(context.Background())
	if err != nil || b.Provider != ProviderPayPal || b.ManageURL != "https://www.paypal.com/myaccount/autopay/" {
		t.Fatalf("got %+v, %v", b, err)
	}
	// An older server says none.
	srv, _, _ = voiceServer(t, 200, billingBody)
	if b, err := voiceClient(srv.URL, StaticToken("tok")).Billing(context.Background()); err != nil || b.Provider != "" {
		t.Fatalf("older server: %+v, %v", b, err)
	}
}

func TestPayPalOffered(t *testing.T) {
	for body, want := range map[string]bool{
		`{"auth":"pocketbase","llm":true,"billing":true,"paypal":true}`: true,
		`{"billing":true,"paypal":false}`:                               false,
		`{"billing":true}`:                                              false, // an older server
		`{"billing":false,"paypal":true}`:                               false,
	} {
		srv, _, _ := voiceServer(t, 200, body)
		if got := voiceClient(srv.URL, nil).PayPalOffered(context.Background()); got != want {
			t.Fatalf("%s: PayPalOffered = %v", body, got)
		}
	}
	srv, _, _ := voiceServer(t, 500, `{}`)
	if voiceClient(srv.URL, nil).PayPalOffered(context.Background()) {
		t.Fatal("a failing server offers PayPal")
	}
}

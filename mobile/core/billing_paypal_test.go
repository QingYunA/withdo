package core

import (
	"encoding/json"
	"testing"

	"github.com/QingYunA/adhd-todo/internal/syncclient"
)

// PayPal (M17): Billing's JSON says whether the server takes PayPal and who bills the plan, and
// Checkout takes the provider.

func TestBillingJSONCarriesPaypalAndProvider(t *testing.T) {
	s := &billingState{known: true, offered: true, paypal: true, have: true, b: syncclient.Billing{Plan: "free"}}
	raw, err := json.Marshal(s.info())
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got["paypal"] != true || got["provider"] == nil {
		t.Fatalf("billing JSON = %s", raw)
	}

	s.paypal = false
	if info := s.info(); info.PayPal {
		t.Fatalf("a server without PayPal offers %+v", info)
	}
	off := &billingState{known: true, offered: false, paypal: true}
	if info := off.info(); info.Offered || info.PayPal {
		t.Fatalf("billing off: %+v", info)
	}
}

func TestCheckoutKnowsOnlyTheProvidersOfTheServer(t *testing.T) {
	phone := billingPhone(t, newFakeBilling(t))
	if _, err := phone.Checkout("monthly", "stripe", "en"); err == nil || err.Error() != "bad_provider" {
		t.Errorf("a provider the server does not have: %v", err)
	}
}

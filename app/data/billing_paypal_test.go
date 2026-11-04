package data

import (
	"testing"

	"github.com/QingYunA/adhd-todo/internal/syncclient"
)

// PayPal (M17): the plan says whether the server takes it and who bills the subscription.

func TestBillingInfoCarriesWhetherPayPalIsOffered(t *testing.T) {
	b := &billingTask{known: true, offered: true, signedIn: true, paypal: true}
	if got := b.info(""); !got.Offered || !got.PayPal {
		t.Fatalf("before the plan is read: %+v", got)
	}
	b.haveBill, b.bill = true, syncclient.Billing{Plan: "free", Monthly: "$6.99", Yearly: "$58.72"}
	if got := b.info(""); !got.Known || !got.PayPal {
		t.Fatalf("with the plan: %+v", got)
	}
	b.paypal = false
	if got := b.info(""); got.PayPal {
		t.Fatalf("a server without PayPal offers %+v", got)
	}
	// Nothing is offered while the server has not answered or billing is off: PayPal is not either.
	off := &billingTask{known: true, offered: false, signedIn: true, paypal: true}
	if got := off.info(""); got.Offered || got.PayPal {
		t.Fatalf("billing off: %+v", got)
	}
}

func TestViaPayPalReadsTheProvider(t *testing.T) {
	for _, c := range []struct {
		provider string
		want     bool
	}{{ProviderPayPal, true}, {ProviderWaffo, false}, {"", false}} {
		if got := (BillingInfo{Provider: c.provider}).ViaPayPal(); got != c.want {
			t.Errorf("provider %q: ViaPayPal = %v", c.provider, got)
		}
	}
}

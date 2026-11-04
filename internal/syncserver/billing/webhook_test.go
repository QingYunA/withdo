package billing

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func (ts *testService) deliver(t *testing.T, body []byte, signedAt time.Time) (Outcome, error) {
	t.Helper()
	_, outcome, err := ts.HandleWebhook(context.Background(), signedHeader(t, ts.webhook, signedAt, body), body)
	return outcome, err
}

func TestWebhookVerification(t *testing.T) {
	ts := newTestService(t, 7)
	body := webhookBody("e1", "subscription.activated", 7, testNow, map[string]any{
		"currentPeriodEnd": testNow.Add(7 * 24 * time.Hour).Format(time.RFC3339), "periodNumber": 0,
	})

	if _, err := ts.deliver(t, body, testNow); err != nil {
		t.Fatalf("genuine: %v", err)
	}
	// Signed by someone else.
	_, _, err := ts.HandleWebhook(context.Background(), signedHeader(t, genKey(t), testNow, body), body)
	if !errors.Is(err, ErrBadSignature) {
		t.Fatalf("a stranger's signature = %v", err)
	}
	// The body changed after signing.
	header := signedHeader(t, ts.webhook, testNow, body)
	if _, _, err := ts.HandleWebhook(context.Background(), header, append([]byte(" "), body...)); !errors.Is(err, ErrBadSignature) {
		t.Fatalf("a changed body = %v", err)
	}
	// A prod payload is checked with the prod key, which did not sign it.
	prod := []byte(strings.Replace(string(body), `"mode":"test"`, `"mode":"prod"`, 1))
	if _, _, err := ts.HandleWebhook(context.Background(), signedHeader(t, ts.webhook, testNow, prod), prod); !errors.Is(err, ErrBadSignature) {
		t.Fatalf("test key on a prod payload = %v", err)
	}
	// Broken headers and bodies.
	for _, h := range []string{"", "garbage", "t=abc,v1=AAAA", "t=1", "v1=AAAA", "t=1,v1=%%%"} {
		if _, _, err := ts.HandleWebhook(context.Background(), h, body); !errors.Is(err, ErrBadSignature) {
			t.Errorf("header %q = %v", h, err)
		}
	}
	notJSON := []byte("not json")
	if _, _, err := ts.HandleWebhook(context.Background(), signedHeader(t, ts.webhook, testNow, notJSON), notJSON); !errors.Is(err, ErrBadSignature) {
		t.Fatalf("junk body = %v", err)
	}
}

func TestWebhookTimestampTolerance(t *testing.T) {
	ts := newTestService(t, 7)
	for i, tc := range []struct {
		age  time.Duration
		want error
	}{
		{44 * time.Minute, nil},
		{-44 * time.Minute, nil},
		{46 * time.Minute, ErrStaleSignature},
		{-46 * time.Minute, ErrStaleSignature},
	} {
		body := webhookBody("tol-"+string(rune('a'+i)), "subscription.renewed", 7, testNow, nil)
		if _, err := ts.deliver(t, body, testNow.Add(-tc.age)); !errors.Is(err, tc.want) && !(tc.want == nil && err == nil) {
			t.Errorf("age %v = %v, want %v", tc.age, err, tc.want)
		}
	}
}

func TestWebhookDeduplicatesByEventID(t *testing.T) {
	ts := newTestService(t, 7)
	body := webhookBody("dup", "subscription.activated", 7, testNow, map[string]any{"periodNumber": 0})
	if out, err := ts.deliver(t, body, testNow); err != nil || out != OutcomeApplied {
		t.Fatalf("first = %q, %v", out, err)
	}
	if out, err := ts.deliver(t, body, testNow); err != nil || out != OutcomeDuplicate {
		t.Fatalf("second = %q, %v", out, err)
	}
	if n := len(ts.notifications()); n != 1 {
		t.Fatalf("%d notifications for one event", n)
	}
}

func TestWebhookIgnoresWhatIsNotOurs(t *testing.T) {
	ts := newTestService(t, 7)
	otherStore := webhookBody("s1", "subscription.activated", 7, testNow, nil)
	otherStore = []byte(strings.Replace(string(otherStore), `"storeId":"STO_withdo"`, `"storeId":"STO_picpocket"`, 1))
	noAccount := webhookBody("s2", "subscription.activated", 7, testNow, map[string]any{"orderMetadata": map[string]any{}})
	badAccount := webhookBody("s3", "subscription.activated", 7, testNow, map[string]any{"orderMetadata": map[string]any{"account": "seven"}})
	unknownAccount := webhookBody("s4", "subscription.activated", 99, testNow, nil)
	prodMode := webhookBody("s5", "subscription.activated", 7, testNow, nil)
	prodMode = []byte(strings.Replace(string(prodMode), `"mode":"test"`, `"mode":"prod"`, 1))
	otherEvent := webhookBody("s6", "refund.succeeded", 7, testNow, nil)

	for name, body := range map[string][]byte{
		"other store": otherStore, "no account": noAccount, "bad account": badAccount,
		"unknown account": unknownAccount, "refund": otherEvent,
	} {
		if out, err := ts.deliver(t, body, testNow); err != nil || out == OutcomeApplied || out == OutcomeDuplicate {
			t.Errorf("%s = %q, %v", name, out, err)
		}
	}
	// A prod event is signed by the prod key; the server is in test mode and ignores it.
	prodKey := genKey(t)
	ts.verify.keys[ModeProd] = &prodKey.PublicKey
	if _, out, err := ts.HandleWebhook(context.Background(), signedHeader(t, prodKey, testNow, prodMode), prodMode); err != nil || out != OutcomeIgnored {
		t.Errorf("prod event on a test server = %q, %v", out, err)
	}
	if len(ts.store.subs) != 0 || len(ts.notifications()) != 0 {
		t.Fatalf("ignored events changed state: %+v", ts.store.subs)
	}
}

func TestAccountMetadataMayBeANumber(t *testing.T) {
	ts := newTestService(t, 7)
	body := webhookBody("num", "subscription.activated", 7, testNow, map[string]any{"orderMetadata": map[string]any{"account": 7}})
	if out, err := ts.deliver(t, body, testNow); err != nil || out != OutcomeApplied {
		t.Fatalf("= %q, %v", out, err)
	}
}

func TestEmbeddedWebhookKeysParse(t *testing.T) {
	keys, err := embeddedWebhookKeys()
	if err != nil || keys[ModeTest] == nil || keys[ModeProd] == nil {
		t.Fatalf("keys = %v, %v", keys, err)
	}
	if keys[ModeTest].N.Cmp(keys[ModeProd].N) == 0 {
		t.Fatal("the test and live keys are the same")
	}
}

func TestLoadPrivateKey(t *testing.T) {
	key := genKey(t)
	dir := t.TempDir()
	pkcs8, _ := x509.MarshalPKCS8PrivateKey(key)
	pkcs1 := x509.MarshalPKCS1PrivateKey(key)
	for name, block := range map[string]*pem.Block{
		"pkcs8.pem": {Type: "PRIVATE KEY", Bytes: pkcs8},
		"pkcs1.pem": {Type: "RSA PRIVATE KEY", Bytes: pkcs1},
	} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, pem.EncodeToMemory(block), 0o600); err != nil {
			t.Fatal(err)
		}
		got, err := LoadPrivateKey(path)
		if err != nil || !got.Equal(key) {
			t.Fatalf("%s: %v", name, err)
		}
	}
	bad := filepath.Join(dir, "bad.pem")
	secret := "secret-bytes-do-not-leak"
	if err := os.WriteFile(bad, []byte(secret), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadPrivateKey(bad); err == nil || strings.Contains(err.Error(), secret) {
		t.Fatalf("bad file = %v", err)
	}
	if _, err := LoadPrivateKey(filepath.Join(dir, "missing.pem")); err == nil {
		t.Fatal("a missing file loaded")
	}
}

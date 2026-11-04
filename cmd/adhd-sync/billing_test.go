package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func envFrom(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func TestBillingFromEnv(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	der, _ := x509.MarshalPKCS8PrivateKey(key)
	keyFile := filepath.Join(t.TempDir(), "waffo.pem")
	if err := os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}
	full := map[string]string{
		"WAFFO_KEY_FILE": keyFile, "WAFFO_MERCHANT_ID": "MER_1", "WAFFO_STORE_ID": "STO_1",
		"WAFFO_PRODUCT_MONTHLY": "PROD_m", "WAFFO_PRODUCT_YEARLY": "PROD_y",
	}
	with := func(k, v string) map[string]string {
		m := map[string]string{}
		for key, val := range full {
			m[key] = val
		}
		m[k] = v
		return m
	}

	if _, on, err := billingFromEnv(envFrom(nil)); on || err != nil {
		t.Fatalf("no key file: on=%v err=%v", on, err)
	}
	cfg, on, err := billingFromEnv(envFrom(full))
	if !on || err != nil || cfg.Mode != "test" || cfg.MerchantID != "MER_1" || cfg.Key == nil || !cfg.Key.Equal(key) {
		t.Fatalf("full = %+v on=%v err=%v", cfg, on, err)
	}
	if cfg, _, _ := billingFromEnv(envFrom(with("WAFFO_MODE", "prod"))); cfg.Mode != "prod" {
		t.Fatalf("mode = %q", cfg.Mode)
	}
	for _, name := range []string{"WAFFO_MERCHANT_ID", "WAFFO_STORE_ID", "WAFFO_PRODUCT_MONTHLY", "WAFFO_PRODUCT_YEARLY"} {
		if _, _, err := billingFromEnv(envFrom(with(name, ""))); err == nil || !strings.Contains(err.Error(), name) {
			t.Errorf("%s unset: %v", name, err)
		}
	}
	if _, _, err := billingFromEnv(envFrom(with("WAFFO_MODE", "live"))); err == nil {
		t.Error("a bad mode was accepted")
	}
	if _, _, err := billingFromEnv(envFrom(with("WAFFO_KEY_FILE", filepath.Join(t.TempDir(), "missing.pem")))); err == nil {
		t.Error("a missing key file was accepted")
	}
}

func TestPayPalFromEnv(t *testing.T) {
	dir := t.TempDir()
	creds := filepath.Join(dir, "paypal-sandbox.json")
	if err := os.WriteFile(creds, []byte(`{"client_id":"cid","secret":"sec","env":"sandbox"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	full := map[string]string{
		"PAYPAL_CREDENTIALS_FILE": creds, "PAYPAL_WEBHOOK_ID": "WH-1", "PAYPAL_PLAN_MONTHLY": "P-M",
		"PAYPAL_PLAN_MONTHLY_TRIAL": "P-MT", "PAYPAL_PLAN_YEARLY": "P-Y", "PAYPAL_PLAN_YEARLY_TRIAL": "P-YT",
	}
	with := func(k, v string) map[string]string {
		m := map[string]string{}
		for key, val := range full {
			m[key] = val
		}
		m[k] = v
		return m
	}

	if cfg, err := paypalFromEnv(envFrom(nil)); cfg != nil || err != nil {
		t.Fatalf("no credentials file: %+v %v", cfg, err)
	}
	cfg, err := paypalFromEnv(envFrom(full))
	if err != nil || cfg.ClientID != "cid" || cfg.Secret != "sec" || cfg.Env != "sandbox" || cfg.WebhookID != "WH-1" ||
		cfg.PlanMonthly != "P-M" || cfg.PlanMonthlyTrial != "P-MT" || cfg.PlanYearly != "P-Y" || cfg.PlanYearlyTrial != "P-YT" {
		t.Fatalf("full = %+v %v", cfg, err)
	}
	// PAYPAL_ENV wins over the file.
	if cfg, err := paypalFromEnv(envFrom(with("PAYPAL_ENV", "live"))); err != nil || cfg.Env != "live" {
		t.Fatalf("PAYPAL_ENV = %+v %v", cfg, err)
	}
	if _, err := paypalFromEnv(envFrom(with("PAYPAL_ENV", "prod"))); err == nil {
		t.Error("a bad PAYPAL_ENV was accepted")
	}
	for _, name := range []string{"PAYPAL_WEBHOOK_ID", "PAYPAL_PLAN_MONTHLY", "PAYPAL_PLAN_MONTHLY_TRIAL", "PAYPAL_PLAN_YEARLY", "PAYPAL_PLAN_YEARLY_TRIAL"} {
		if _, err := paypalFromEnv(envFrom(with(name, ""))); err == nil || !strings.Contains(err.Error(), name) {
			t.Errorf("%s unset: %v", name, err)
		}
	}
	if _, err := paypalFromEnv(envFrom(with("PAYPAL_CREDENTIALS_FILE", filepath.Join(dir, "missing.json")))); err == nil {
		t.Error("a missing credentials file was accepted")
	}
	for name, content := range map[string]string{"junk.json": "not json", "partial.json": `{"client_id":"cid"}`} {
		path := filepath.Join(dir, name)
		_ = os.WriteFile(path, []byte(content), 0o600)
		_, err := paypalFromEnv(envFrom(with("PAYPAL_CREDENTIALS_FILE", path)))
		if err == nil || strings.Contains(err.Error(), "cid") {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestBillingFromEnvCarriesPayPal(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	der, _ := x509.MarshalPKCS8PrivateKey(key)
	dir := t.TempDir()
	keyFile := filepath.Join(dir, "waffo.pem")
	_ = os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), 0o600)
	creds := filepath.Join(dir, "paypal.json")
	_ = os.WriteFile(creds, []byte(`{"client_id":"cid","secret":"sec"}`), 0o600)
	env := map[string]string{
		"WAFFO_KEY_FILE": keyFile, "WAFFO_MERCHANT_ID": "MER_1", "WAFFO_STORE_ID": "STO_1",
		"WAFFO_PRODUCT_MONTHLY": "PROD_m", "WAFFO_PRODUCT_YEARLY": "PROD_y",
	}
	if cfg, on, err := billingFromEnv(envFrom(env)); !on || err != nil || cfg.PayPal != nil {
		t.Fatalf("Waffo only: %+v on=%v err=%v", cfg, on, err)
	}
	env["PAYPAL_CREDENTIALS_FILE"] = creds
	if _, _, err := billingFromEnv(envFrom(env)); err == nil {
		t.Fatal("a half-set PayPal started")
	}
	for k, v := range map[string]string{"PAYPAL_WEBHOOK_ID": "WH-1", "PAYPAL_PLAN_MONTHLY": "a", "PAYPAL_PLAN_MONTHLY_TRIAL": "b", "PAYPAL_PLAN_YEARLY": "c", "PAYPAL_PLAN_YEARLY_TRIAL": "d"} {
		env[k] = v
	}
	cfg, on, err := billingFromEnv(envFrom(env))
	if !on || err != nil || cfg.PayPal == nil || cfg.PayPal.Env != "" {
		t.Fatalf("both: %+v on=%v err=%v", cfg, on, err)
	}
}

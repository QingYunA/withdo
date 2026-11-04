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

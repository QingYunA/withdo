package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/QingYunA/adhd-todo/internal/syncserver/billing"
)

// billingFromEnv reads Withdo Pro's settings (M16). Without WAFFO_KEY_FILE billing is off (on is
// false, no error) and the routes answer 503 billing_disabled. With it, the merchant, the store and
// both products are required and the key file must load: a half-set deployment fails at start
// instead of failing a customer's checkout.
func billingFromEnv(getenv func(string) string) (cfg billing.Config, on bool, err error) {
	keyFile := strings.TrimSpace(getenv("WAFFO_KEY_FILE"))
	if keyFile == "" {
		return cfg, false, nil
	}
	cfg = billing.Config{
		MerchantID:     strings.TrimSpace(getenv("WAFFO_MERCHANT_ID")),
		StoreID:        strings.TrimSpace(getenv("WAFFO_STORE_ID")),
		ProductMonthly: strings.TrimSpace(getenv("WAFFO_PRODUCT_MONTHLY")),
		ProductYearly:  strings.TrimSpace(getenv("WAFFO_PRODUCT_YEARLY")),
		Mode:           strings.TrimSpace(getenv("WAFFO_MODE")),
	}
	for name, value := range map[string]string{
		"WAFFO_MERCHANT_ID": cfg.MerchantID, "WAFFO_STORE_ID": cfg.StoreID,
		"WAFFO_PRODUCT_MONTHLY": cfg.ProductMonthly, "WAFFO_PRODUCT_YEARLY": cfg.ProductYearly,
	} {
		if value == "" {
			return cfg, false, fmt.Errorf("%s is required when WAFFO_KEY_FILE is set", name)
		}
	}
	if cfg.Mode != "" && cfg.Mode != billing.ModeTest && cfg.Mode != billing.ModeProd {
		return cfg, false, errors.New("WAFFO_MODE must be test or prod")
	}
	if cfg.Mode == "" {
		cfg.Mode = billing.ModeTest
	}
	if cfg.Key, err = billing.LoadPrivateKey(keyFile); err != nil {
		return cfg, false, err
	}
	if cfg.PayPal, err = paypalFromEnv(getenv); err != nil {
		return cfg, false, err
	}
	return cfg, true, nil
}

// paypalFromEnv reads PayPal's settings (M17). Without PAYPAL_CREDENTIALS_FILE PayPal is off (nil, no
// error). With it, the webhook id and the four plan ids are required and the file must load, so a
// half-set deployment fails at start. PAYPAL_ENV (sandbox or live) wins over the file's "env"; with
// neither it is the sandbox.
func paypalFromEnv(getenv func(string) string) (*billing.PayPalConfig, error) {
	file := strings.TrimSpace(getenv("PAYPAL_CREDENTIALS_FILE"))
	if file == "" {
		return nil, nil
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("read the PayPal credentials file: %w", err)
	}
	var creds struct {
		ClientID string `json:"client_id"`
		Secret   string `json:"secret"`
		Env      string `json:"env"`
	}
	if err := json.Unmarshal(raw, &creds); err != nil {
		return nil, errors.New("the PayPal credentials file is not JSON")
	}
	if creds.ClientID == "" || creds.Secret == "" {
		return nil, errors.New("the PayPal credentials file needs client_id and secret")
	}
	cfg := &billing.PayPalConfig{
		ClientID: creds.ClientID, Secret: creds.Secret, Env: strings.TrimSpace(getenv("PAYPAL_ENV")),
		WebhookID:        strings.TrimSpace(getenv("PAYPAL_WEBHOOK_ID")),
		PlanMonthly:      strings.TrimSpace(getenv("PAYPAL_PLAN_MONTHLY")),
		PlanMonthlyTrial: strings.TrimSpace(getenv("PAYPAL_PLAN_MONTHLY_TRIAL")),
		PlanYearly:       strings.TrimSpace(getenv("PAYPAL_PLAN_YEARLY")),
		PlanYearlyTrial:  strings.TrimSpace(getenv("PAYPAL_PLAN_YEARLY_TRIAL")),
	}
	if cfg.Env == "" {
		cfg.Env = creds.Env
	}
	if cfg.Env != "" && cfg.Env != billing.PayPalSandbox && cfg.Env != billing.PayPalLive {
		return nil, errors.New("PAYPAL_ENV must be sandbox or live")
	}
	for name, value := range map[string]string{
		"PAYPAL_WEBHOOK_ID": cfg.WebhookID, "PAYPAL_PLAN_MONTHLY": cfg.PlanMonthly,
		"PAYPAL_PLAN_MONTHLY_TRIAL": cfg.PlanMonthlyTrial, "PAYPAL_PLAN_YEARLY": cfg.PlanYearly,
		"PAYPAL_PLAN_YEARLY_TRIAL": cfg.PlanYearlyTrial,
	} {
		if value == "" {
			return nil, fmt.Errorf("%s is required when PAYPAL_CREDENTIALS_FILE is set", name)
		}
	}
	return cfg, nil
}

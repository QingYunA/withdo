package main

import (
	"errors"
	"fmt"
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
	return cfg, true, nil
}

-- Withdo Pro through PayPal (M17): which channel the account's subscription is on, 'waffo' or
-- 'paypal'. For a 'paypal' row order_id is the PayPal subscription id (I-...) and mode is 'test'
-- for the sandbox, 'prod' for live. There is still one row per account.
ALTER TABLE billing_subscriptions ADD COLUMN provider TEXT NOT NULL DEFAULT 'waffo';

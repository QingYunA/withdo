-- The subscription's billing interval, "monthly" or "yearly" ('' when no event has said), so the
-- apps can say which plan the account is on. Taken from the webhooks' billingPeriod.
ALTER TABLE billing_subscriptions ADD COLUMN billing_interval TEXT NOT NULL DEFAULT '';

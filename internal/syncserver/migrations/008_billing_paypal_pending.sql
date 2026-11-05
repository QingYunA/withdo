-- Withdo Pro through PayPal (M17): the PayPal subscription an account's latest checkout made and has
-- not been seen in its row yet. Webhooks are the way a subscription reaches billing_subscriptions;
-- when one is lost, GET /v1/billing reads this subscription from PayPal instead. One per account:
-- a new checkout replaces it, and it is deleted once the row holds the subscription.
CREATE TABLE billing_paypal_pending (
  user_id         BIGINT PRIMARY KEY REFERENCES users (id),
  subscription_id TEXT NOT NULL,
  created_at      TIMESTAMPTZ NOT NULL
);

-- Withdo Pro (M16): the subscription Waffo Pancake's webhooks last put each account in, and the
-- ids of the webhook events already handled (Waffo sends an event more than once).
CREATE TABLE billing_subscriptions (
  user_id      BIGINT PRIMARY KEY REFERENCES users (id),
  order_id     TEXT NOT NULL DEFAULT '',
  product_name TEXT NOT NULL DEFAULT '',
  -- trialing, active, past_due, canceling or canceled
  status       TEXT NOT NULL,
  period_start TIMESTAMPTZ,
  period_end   TIMESTAMPTZ,
  trial_used   BOOLEAN NOT NULL DEFAULT false,
  -- test or prod: the mode of the newest event
  mode         TEXT NOT NULL,
  -- the timestamp of the newest event applied; an older one arriving late is dropped
  event_at     TIMESTAMPTZ NOT NULL,
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE billing_events (
  id          TEXT PRIMARY KEY,
  event_type  TEXT NOT NULL,
  received_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX billing_events_received_idx ON billing_events (received_at);

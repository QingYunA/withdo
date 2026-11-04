# M17: PayPal as a second way to pay for Withdo Pro

Decided 2026-11-04. Waffo (M16) stays the main checkout (cards, Apple Pay, Google Pay, as merchant
of record). PayPal is a separate channel the user asked for: a PayPal button under the plan sheet's
main button. Both write the same per-account subscription, so Pro, the quota and the apps do not
care which one was used.

**Tax.** PayPal is not a merchant of record: for PayPal sales the seller (the owner) collects and
remits VAT and sales tax where they are due (EU VAT on digital services from the first sale).

## PayPal (sandbox, set up 2026-11-04)

- Developer app **Withdo** (Merchant) on the owner's PayPal account; its client id and secret are
  only on the server: `/opt/adhd-todo/sync/secrets/paypal-sandbox.json` (`{"client_id","secret","env"}`,
  mode 600, owner 65532).
- Created through the API: product, four plans and the webhook:

```json
{"product": "PROD-1NV88459X9209392Y", "monthly_trial": "P-3MA95878A4870393LNLVOKMA", "monthly": "P-6FL403268U768440TNLVOKMQ", "yearly_trial": "P-4UX93808L6969313JNLVOKMY", "yearly": "P-6MY38608R0206392FNLVOKNA", "webhook": "82659267CT025681J"}
```

  Plans: monthly $6.99 and yearly $58.72, each with a 7-day $0 trial (`*_trial`) and without one
  (for accounts that already had a trial, on either channel).
- Webhook `https://api.withdo.si/v1/billing/paypal` for BILLING.SUBSCRIPTION.ACTIVATED, UPDATED,
  CANCELLED, SUSPENDED, EXPIRED, RE-ACTIVATED, PAYMENT.FAILED, PAYMENT.SALE.COMPLETED, REFUNDED.
- Going live: a live app, the same product, plans and webhook created against
  `https://api-m.paypal.com` (the setup script is in the PR), `secrets/paypal-live.json`, then the env.

## Server

- `POST /v1/billing/checkout` takes `"provider": "waffo"|"paypal"` (default waffo). For PayPal it
  creates `POST /v1/billing/subscriptions` with the plan (trial or not), `custom_id` = the account,
  `subscriber.email_address`, `application_context` (brand `Withdo`, `SUBSCRIBE_NOW`,
  `NO_SHIPPING`, return and cancel URLs `https://withdo.si/pro/`) and answers the `approve` link.
- `POST /v1/billing/paypal`: verify with `POST /v1/notifications/verify-webhook-signature` (the
  five PAYPAL-* headers, the webhook id, the raw event); then **read the subscription from PayPal**
  (`GET /v1/billing/subscriptions/{id}`) and apply its state, never the payload's: ACTIVE in the
  trial cycle → trialing, ACTIVE → active, SUSPENDED → past_due, CANCELLED → canceling until
  `billing_info.next_billing_time` (or the last payment's period end), then canceled; EXPIRED →
  canceled. The account is `custom_id`. Dedupe by the event id. 200 for every verified event.
- One row per account as in M16, with `provider` (`waffo`, `paypal`); `GET /v1/billing` says it.
  `manage_url` for PayPal is the payer's automatic-payments page
  (`https://www.paypal.com/myaccount/autopay/`, sandbox `https://www.sandbox.paypal.com/myaccount/autopay/`).
- Env: `PAYPAL_CREDENTIALS_FILE`, `PAYPAL_ENV` (`sandbox`|`live`), `PAYPAL_WEBHOOK_ID`,
  `PAYPAL_PLAN_MONTHLY`, `PAYPAL_PLAN_MONTHLY_TRIAL`, `PAYPAL_PLAN_YEARLY`, `PAYPAL_PLAN_YEARLY_TRIAL`.
  Without the credentials file PayPal is off and `/v1/config` says `"paypal": false`; the apps hide the button.

## Clients (design v46)

The plan sheet shows, under the main button, `或` and a PayPal-branded button `用 PayPal 订阅`
(`#FFC439`, the PayPal wordmark), which checks out the chosen plan with `provider: "paypal"`; the
waiting state says `在 PayPal 里完成订阅` / `重新打开 PayPal`. A PayPal subscription's Pro line adds
`通过 PayPal` / `via PayPal`.
